// Command pnl_probe reconciles a member's settled-market realized P&L against
// Kalshi's Portfolio > History UI, straight from the API (via gokalshi).
//
// Per settled market it prints the settlement + fills-derived figures so you can
// diff them against Kalshi's columns. It was written to lock the History P&L
// formula for terminal issue #107 and is kept as a diagnostic for that surface:
//   - SET(ret) = revenue + min(yes,no)*$1 - (yesCost+noCost) - fees  (exact return)
//   - fifoC    = FIFO net-position open cost keyed on outcome_side + ts
//   - Kcost/Kpay = fifoC + fees / Kcost + SET(ret)  (Kalshi's cost / payout)
//
// The matched-pair term and the outcome_side FIFO netting are the two non-obvious
// bits -- see the terminal PR for the full write-up.
//
// It is read-only (only GET endpoints) and prefers the read-only prod key.
//
// Run from the gokalshi repo root (so .env loads):
//
//	go run ./cmd/pnl_probe                 # reconcile the N most recent settlements
//	go run ./cmd/pnl_probe -n 80           # widen the window
//	go run ./cmd/pnl_probe -ticker KXBTC-. # one market + raw JSON dump
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/Kalarb/gokalshi"
	"github.com/joho/godotenv"
)

func main() {
	tickerFlag := flag.String("ticker", "", "restrict to one market ticker (also dumps raw JSON)")
	maxSettlements := flag.Int("n", 40, "max most-recent settlements to reconcile (fills fan-out)")
	flag.Parse()

	_ = godotenv.Load() // load gokalshi/.env from CWD (repo root)

	client, err := newReadOnlyProdClient()
	if err != nil {
		log.Fatalf("client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	settlements, err := allSettlements(ctx, client, *tickerFlag, *maxSettlements)
	if err != nil {
		log.Fatalf("settlements: %v", err)
	}
	if len(settlements) == 0 {
		fmt.Println("no settlements returned")
		return
	}
	fmt.Printf("reconciling %d settlement(s)\n\n", len(settlements))

	printHeader()
	for _, s := range settlements {
		fills, err := allFillsForTicker(ctx, client, s.Ticker)
		if err != nil {
			log.Printf("fills for %s failed: %v", s.Ticker, err)
			continue
		}
		printRow(s, fills)

		if *tickerFlag != "" {
			dumpRaw(s, fills)
		}
	}

	fmt.Println("\nLEGEND (all dollars) -- Kcost/Kpay use ONLY outcome_side + ts (NO deprecated action):")
	fmt.Println("  yesC/noC = settlement yes/no_total_cost_dollars")
	fmt.Println("  SET(ret) = revenue + min(yes,no)*$1 - (yesC+noC) - fees   (EXACT return, settlement-only)")
	fmt.Println("  fifoC    = FIFO net-position open cost (round-trips netted, by ts)")
	fmt.Println("  Kcost    = fifoC + fees        (candidate Kalshi 'Total cost', incl fees)")
	fmt.Println("  Kpay     = Kcost + SET(ret)    (candidate Kalshi 'Total payout')")
	fmt.Println("\nKalshi UI to match: BTC 63340 cost 1.11/pay 0.85/ret -0.25; BTC 63057 0.06/0.06/+0.01;")
	fmt.Println("Oil 0.39/0.39/0; Colombia 4.17/0/-4.17; France 2.91/2.84/-0.06.")
}

// newReadOnlyProdClient builds a prod client from the read-only key when
// present, else falls back to the default prod key via NewClientConfig.
func newReadOnlyProdClient() (*gokalshi.Client, error) {
	keyID := os.Getenv("KALSHI_PROD_READ_ONLY_API_KEY_ID")
	keyFile := os.Getenv("KALSHI_PROD_READ_ONLY_PRIVATE_KEY_FILE")
	if keyID != "" && keyFile != "" {
		creds, err := gokalshi.LoadCredentials(keyID, keyFile)
		if err != nil {
			return nil, fmt.Errorf("load read-only creds: %w", err)
		}
		return gokalshi.NewClient(&gokalshi.ClientConfig{
			Environment: gokalshi.Prod,
			Credentials: creds,
		})
	}
	cfg, err := gokalshi.NewClientConfig()
	if err != nil {
		return nil, err
	}
	if cfg.Environment != gokalshi.Prod {
		return nil, fmt.Errorf("expected PROD environment, got %s", cfg.Environment)
	}
	return gokalshi.NewClient(cfg)
}

func allSettlements(ctx context.Context, c *gokalshi.Client, ticker string, max int) ([]gokalshi.Settlement, error) {
	var out []gokalshi.Settlement
	params := gokalshi.GetSettlementsParams{Ticker: ticker, Limit: 100}
	for {
		resp, err := c.GetSettlements(ctx, params)
		if err != nil {
			return nil, err
		}
		out = append(out, resp.Settlements...)
		if resp.Cursor == "" || (max > 0 && len(out) >= max) {
			break
		}
		params.Cursor = resp.Cursor
	}
	if max > 0 && len(out) > max {
		out = out[:max]
	}
	return out, nil
}

func allFillsForTicker(ctx context.Context, c *gokalshi.Client, ticker string) ([]gokalshi.Fill, error) {
	var out []gokalshi.Fill
	params := gokalshi.GetFillsParams{Ticker: ticker, Limit: 200}
	for i := 0; i < 50; i++ { // bounded safety cap
		resp, err := c.GetFills(ctx, params)
		if err != nil {
			return nil, err
		}
		out = append(out, resp.Fills...)
		if resp.Cursor == "" {
			return out, nil
		}
		params.Cursor = resp.Cursor
	}
	return out, nil
}

// metrics holds the candidate P&L numbers for one settled market.
type metrics struct {
	yesCost, noCost       float64
	buyCost, sellProceeds float64 // raw yes|no price per outcome_side
	buyCost1p, sellProc1p float64 // yes price with 1-p flip for no
	fillFees, settleFee   float64
	settleRev             float64 // dollars
	yesCount, noCount     float64
	netPos                float64
	fifoOpenCost          float64 // FIFO net-position open cost (outcome_side + ts, NO action)
	nFills                int
}

// fifoOpenCost computes the cost basis of positions OPENED, walking fills by ts
// and netting round-trips: a fill that grows |netPos| opens (cost at its
// outcome-side price); one that shrinks it closes (no cost). Uses ONLY
// outcome_side + ts + price -- the canonical fields terminal-api sends (action is
// deprecated). netPos is yes-equivalent (os=yes +, os=no -).
func fifoOpenCost(fills []gokalshi.Fill) float64 {
	sorted := make([]gokalshi.Fill, len(fills))
	copy(sorted, fills)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TS < sorted[j].TS })

	netPos := 0.0
	cost := 0.0
	for _, f := range sorted {
		qty := money(f.CountFP)
		signed := qty
		sidePrice := money(f.YesPriceDollars)
		if f.OutcomeSide == "no" {
			signed = -qty
			sidePrice = money(f.NoPriceDollars)
		}
		openQty := qty
		if netPos != 0 && (netPos > 0) != (signed > 0) {
			closeQty := math.Min(qty, math.Abs(netPos))
			openQty = qty - closeQty
		}
		cost += sidePrice * openQty
		netPos += signed
	}
	return cost
}

func compute(s gokalshi.Settlement, fills []gokalshi.Fill) metrics {
	m := metrics{
		yesCost:      money(s.YesTotalCostDollars),
		noCost:       money(s.NoTotalCostDollars),
		settleFee:    money(s.FeeCost),
		settleRev:    float64(s.Revenue) / 100.0,
		yesCount:     money(s.YesCountFP),
		noCount:      money(s.NoCountFP),
		netPos:       money(s.YesCountFP) - money(s.NoCountFP),
		fifoOpenCost: fifoOpenCost(fills),
		nFills:       len(fills),
	}
	for _, f := range fills {
		yes := money(f.YesPriceDollars)
		no := money(f.NoPriceDollars)
		count := money(f.CountFP)
		// outcome_side is the RESULTING position; a sell trades the opposite
		// contract. So the contract actually traded is:
		//   buy  -> outcome_side
		//   sell -> opposite(outcome_side)
		contractIsYes := (f.Action == gokalshi.ActionBuy && f.OutcomeSide == "yes") ||
			(f.Action == gokalshi.ActionSell && f.OutcomeSide == "no")
		price := no
		price1p := 1 - yes // terminal has only the yes price (price_dollars)
		if contractIsYes {
			price = yes
			price1p = yes
		}
		notional := price * count
		notional1p := price1p * count
		if f.Action == gokalshi.ActionBuy {
			m.buyCost += notional
			m.buyCost1p += notional1p
		} else {
			m.sellProceeds += notional
			m.sellProc1p += notional1p
		}
		m.fillFees += money(f.FeeCost)
	}
	return m
}

func (m metrics) reconReturn() float64 { return m.sellProceeds + m.settleRev - m.buyCost - m.fillFees }
func (m metrics) curReturn() float64   { return (m.settleRev - m.settleFee) - (m.yesCost + m.noCost) }
func (m metrics) return1p() float64    { return m.sellProc1p + m.settleRev - m.buyCost1p - m.fillFees }

// SET is the SETTLEMENT-ONLY candidate: no fills needed. It adds the matched-pair
// self-settlement payout (min(yes,no) contracts x $1) that curReturn() omits.
func (m metrics) setReturn() float64 {
	return m.settleRev + min(m.yesCount, m.noCount) - (m.yesCost + m.noCost) - m.settleFee
}

func printHeader() {
	fmt.Printf("%-26s %-4s %5s | %7s %7s | %8s %8s | %8s %8s %8s | %s\n",
		"ticker", "res", "net", "yesC", "noC",
		"SET(ret)", "fifoC", "Kcost", "Kpay", "sRev", "nF")
}

func printRow(s gokalshi.Settlement, fills []gokalshi.Fill) {
	m := compute(s, fills)
	kCost := m.fifoOpenCost + m.settleFee // Kalshi cost = net open cost + fees
	kPay := kCost + m.setReturn()         // Kalshi payout = cost + return
	fmt.Printf("%-26s %-4s %5.0f | %7.2f %7.2f | %8.2f %8.2f | %8.2f %8.2f %8.2f | %d\n",
		trunc(s.Ticker, 26), s.MarketResult, m.netPos,
		m.yesCost, m.noCost,
		m.setReturn(), m.fifoOpenCost, kCost, kPay, m.settleRev, m.nFills)
}

func dumpRaw(s gokalshi.Settlement, fills []gokalshi.Fill) {
	fmt.Println("\n--- RAW settlement ---")
	printJSON(s)
	fmt.Printf("--- RAW fills (%d) ---\n", len(fills))
	printJSON(fills)
}

func printJSON(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Printf("json: %v", err)
		return
	}
	fmt.Println(string(b))
}

func money(s string) float64 {
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Printf("parse %q: %v", s, err)
		return 0
	}
	return f
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
