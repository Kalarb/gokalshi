// ratelimit_test validates Kalshi's documented rate limit behavior against
// the live PROD API. It runs burst and sustained-rate tests for reads, writes,
// and mixed traffic to confirm the token bucket model.
//
// Each test runs two phases:
//   - Phase 1 (Sequential): single-goroutine requests to establish baseline
//   - Phase 2 (Concurrent): goroutine pool to actually saturate the budget
//
// Usage:
//
//	go run ./tools/ratelimit_test                 # run all tests
//	go run ./tools/ratelimit_test -test read     # read-only test
//	go run ./tools/ratelimit_test -test write    # write-only test
//	go run ./tools/ratelimit_test -test mixed    # mixed test
//	go run ./tools/ratelimit_test -test limiter  # client-side limiter prevents 429s
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Kalarb/gokalshi"
	"github.com/joho/godotenv"
)

const (
	testMarket    = "KXELONMARS-99"
	orderPrice    = "0.01"
	burstWaitSecs = 5
	sustainedDur  = 30 * time.Second
	overrateDur   = 10 * time.Second
	overrateMult  = 1.5
	concurrency   = 50
)

type tierInfo struct {
	Tier          string
	ReadRate      int
	ReadCapacity  int
	WriteRate     int
	WriteCapacity int
}

type endpointCost struct {
	ReadCost  float64
	WriteCost float64
	Default   float64
}

// result is a concurrent-safe test result counter.
type result struct {
	sent     atomic.Int64
	success  atomic.Int64
	rate429  atomic.Int64
	errs     atomic.Int64
	first429 atomic.Int64
	Duration time.Duration

	errMu      sync.Mutex
	errSamples []string // first N non-429 error messages
}

const maxErrSamples = 5

func (r *result) record(err error) {
	n := r.sent.Add(1)
	if err == nil {
		r.success.Add(1)
		return
	}
	if is429(err) {
		r.rate429.Add(1)
		r.first429.CompareAndSwap(0, n)
	} else {
		r.errs.Add(1)
		r.errMu.Lock()
		if len(r.errSamples) < maxErrSamples {
			// Extract status code if available
			var apiErr *gokalshi.APIError
			if errors.As(err, &apiErr) {
				r.errSamples = append(r.errSamples, fmt.Sprintf("#%d: HTTP %d: %s", n, apiErr.StatusCode, apiErr.Code))
			} else {
				r.errSamples = append(r.errSamples, fmt.Sprintf("#%d: %v", n, err))
			}
		}
		r.errMu.Unlock()
	}
}

func main() {
	testFlag := flag.String("test", "all", "test to run: read, write, mixed, all")
	flag.Parse()

	loadEnv()

	keyID := os.Getenv("KALSHI_PROD_API_KEY_ID")
	keyFile := os.Getenv("KALSHI_PROD_PRIVATE_KEY_FILE")
	if keyID == "" || keyFile == "" {
		log.Fatal("KALSHI_PROD_API_KEY_ID and KALSHI_PROD_PRIVATE_KEY_FILE must be set")
	}

	creds, err := gokalshi.LoadCredentials(keyID, keyFile)
	if err != nil {
		log.Fatalf("load credentials: %v", err)
	}

	cfg := &gokalshi.ClientConfig{
		Environment: gokalshi.Prod,
		Credentials: creds,
		HTTPBaseURL: "https://api.elections.kalshi.com",
	}

	limiter := gokalshi.NewReadWriteTokenBucket(gokalshi.TokenBucketConfig{
		ReadRate: 99999, WriteRate: 99999, WindowSize: 1.0,
	})
	client, err := gokalshi.NewClient(cfg, gokalshi.WithRateLimiter(limiter))
	if err != nil {
		log.Fatalf("create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	tier, costs := fetchTierInfo(ctx, client)
	printHeader(tier, costs)

	switch *testFlag {
	case "read":
		runReadTest(ctx, client, tier, costs)
	case "write":
		runWriteTest(ctx, client, tier, costs)
	case "mixed":
		runMixedTest(ctx, client, tier, costs)
	case "batch":
		runBatchTest(ctx, client, tier, costs)
	case "limiter":
		runClientLimiterTest(ctx, cfg, tier, costs)
	case "all":
		runReadTest(ctx, client, tier, costs)
		fmt.Println()
		runWriteTest(ctx, client, tier, costs)
		fmt.Println()
		runMixedTest(ctx, client, tier, costs)
		fmt.Println()
		runBatchTest(ctx, client, tier, costs)
		fmt.Println()
		runClientLimiterTest(ctx, cfg, tier, costs)
	default:
		log.Fatalf("unknown test: %s", *testFlag)
	}
}

func fetchTierInfo(ctx context.Context, client *gokalshi.Client) (tierInfo, endpointCost) {
	limits, err := client.GetAccountAPILimits(ctx)
	if err != nil {
		log.Fatalf("fetch limits: %v", err)
	}
	costsResp, err := client.GetAccountEndpointCosts(ctx)
	if err != nil {
		log.Fatalf("fetch endpoint costs: %v", err)
	}

	tier := tierInfo{
		Tier:          limits.UsageTier,
		ReadRate:      limits.Read.RefillRate,
		ReadCapacity:  limits.Read.BucketCapacity,
		WriteRate:     limits.Write.RefillRate,
		WriteCapacity: limits.Write.BucketCapacity,
	}

	costs := endpointCost{
		ReadCost:  float64(costsResp.DefaultCost),
		WriteCost: float64(costsResp.DefaultCost),
		Default:   float64(costsResp.DefaultCost),
	}
	for _, ec := range costsResp.EndpointCosts {
		path := strings.ToLower(ec.Path)
		method := strings.ToUpper(ec.Method)
		if method == "GET" && strings.Contains(path, "portfolio/balance") {
			costs.ReadCost = float64(ec.Cost)
		}
		if method == "POST" && strings.Contains(path, "orders") && !strings.Contains(path, "batch") {
			costs.WriteCost = float64(ec.Cost)
		}
	}

	return tier, costs
}

func printHeader(tier tierInfo, costs endpointCost) {
	fmt.Println("=== Rate Limit Validation (PROD) ===")
	fmt.Printf("Tier:            %s\n", tier.Tier)
	fmt.Printf("Read:            %d tokens/sec, capacity %d\n", tier.ReadRate, tier.ReadCapacity)
	fmt.Printf("Write:           %d tokens/sec, capacity %d\n", tier.WriteRate, tier.WriteCapacity)
	fmt.Printf("Read endpoint:   GET /portfolio/balance (cost: %.0f)\n", costs.ReadCost)
	fmt.Printf("Write endpoint:  POST /portfolio/orders (cost: %.0f)\n", costs.WriteCost)
	fmt.Printf("Test market:     %s @ %s\n", testMarket, orderPrice)
	fmt.Printf("Concurrency:     %d goroutines (Phase 2)\n", concurrency)
	fmt.Println()
}

// ---------------------------------------------------------------------------
// Phase 1: Sequential (single goroutine)
// ---------------------------------------------------------------------------

func seqBurstRead(ctx context.Context, client *gokalshi.Client, count int) *result {
	r := &result{}
	start := time.Now()
	for i := 0; i < count; i++ {
		_, err := client.GetBalance(ctx)
		r.record(err)
	}
	r.Duration = time.Since(start)
	return r
}

func seqSustainedRead(ctx context.Context, client *gokalshi.Client, rps float64, dur time.Duration) *result {
	r := &result{}
	interval := time.Duration(float64(time.Second) / rps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	deadline := time.After(dur)
	for {
		select {
		case <-deadline:
			return r
		case <-ticker.C:
			_, err := client.GetBalance(ctx)
			r.record(err)
		}
	}
}

func seqBurstWrite(ctx context.Context, client *gokalshi.Client, count int) (*result, []string) {
	r := &result{}
	var ids []string
	start := time.Now()
	for i := 0; i < count; i++ {
		resp, err := client.CreateOrder(ctx, gokalshi.CreateOrderRequest{
			Ticker: testMarket, Side: gokalshi.SideYes, Action: gokalshi.ActionBuy,
			CountFP: ptr("1.00"), YesPriceDollars: orderPrice,
		})
		r.record(err)
		if err == nil {
			ids = append(ids, resp.Order.OrderID)
		}
	}
	r.Duration = time.Since(start)
	return r, ids
}

func seqSustainedWrite(ctx context.Context, client *gokalshi.Client, rps float64, dur time.Duration) (*result, []string) {
	r := &result{}
	var ids []string
	interval := time.Duration(float64(time.Second) / rps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	deadline := time.After(dur)
	for {
		select {
		case <-deadline:
			return r, ids
		case <-ticker.C:
			resp, err := client.CreateOrder(ctx, gokalshi.CreateOrderRequest{
				Ticker: testMarket, Side: gokalshi.SideYes, Action: gokalshi.ActionBuy,
				CountFP: ptr("1.00"), YesPriceDollars: orderPrice,
			})
			r.record(err)
			if err == nil {
				ids = append(ids, resp.Order.OrderID)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Phase 2: Concurrent (goroutine pool)
// ---------------------------------------------------------------------------

func concBurstRead(ctx context.Context, client *gokalshi.Client, count int) *result {
	r := &result{}
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	start := time.Now()

	for i := 0; i < count; i++ {
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			_, err := client.GetBalance(ctx)
			r.record(err)
		}()
	}
	wg.Wait()
	r.Duration = time.Since(start)
	return r
}

func concSustainedRead(ctx context.Context, client *gokalshi.Client, rps float64, dur time.Duration) *result {
	r := &result{}
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	interval := time.Duration(float64(time.Second) / rps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	deadline := time.After(dur)

	for {
		select {
		case <-deadline:
			wg.Wait()
			return r
		case <-ticker.C:
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				_, err := client.GetBalance(ctx)
				r.record(err)
			}()
		}
	}
}

func concBurstWrite(ctx context.Context, client *gokalshi.Client, count int) (*result, []string) {
	r := &result{}
	var ids []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	start := time.Now()

	for i := 0; i < count; i++ {
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			resp, err := client.CreateOrder(ctx, gokalshi.CreateOrderRequest{
				Ticker: testMarket, Side: gokalshi.SideYes, Action: gokalshi.ActionBuy,
				CountFP: ptr("1.00"), YesPriceDollars: orderPrice,
			})
			r.record(err)
			if err == nil {
				mu.Lock()
				ids = append(ids, resp.Order.OrderID)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	r.Duration = time.Since(start)
	return r, ids
}

func concSustainedWrite(ctx context.Context, client *gokalshi.Client, rps float64, dur time.Duration) (*result, []string) {
	r := &result{}
	var ids []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	interval := time.Duration(float64(time.Second) / rps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	deadline := time.After(dur)

	for {
		select {
		case <-deadline:
			wg.Wait()
			return r, ids
		case <-ticker.C:
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				resp, err := client.CreateOrder(ctx, gokalshi.CreateOrderRequest{
					Ticker: testMarket, Side: gokalshi.SideYes, Action: gokalshi.ActionBuy,
					CountFP: ptr("1.00"), YesPriceDollars: orderPrice,
				})
				r.record(err)
				if err == nil {
					mu.Lock()
					ids = append(ids, resp.Order.OrderID)
					mu.Unlock()
				}
			}()
		}
	}
}

// ---------------------------------------------------------------------------
// Test 1: Read-only
// ---------------------------------------------------------------------------

func runReadTest(ctx context.Context, client *gokalshi.Client, tier tierInfo, costs endpointCost) {
	fmt.Println("--- Test 1: Read-Only ---")
	expectedBurst := int(float64(tier.ReadCapacity) / costs.ReadCost)
	rps := float64(tier.ReadRate) / costs.ReadCost

	fmt.Println("\n  Phase 1: Sequential")
	r := seqBurstRead(ctx, client, expectedBurst*3)
	printResult("Read Burst", r, expectedBurst)
	waitRefill()
	r = seqSustainedRead(ctx, client, rps, sustainedDur)
	printResult("Read Sustained (at rate)", r, 0)
	waitRefill()
	r = seqSustainedRead(ctx, client, rps*overrateMult, overrateDur)
	printResult("Read Sustained (at 1.5x)", r, 0)
	waitRefill()

	fmt.Printf("\n  Phase 2: Concurrent (%d goroutines)\n", concurrency)
	r = concBurstRead(ctx, client, expectedBurst*3)
	printResult("Read Burst", r, expectedBurst)
	waitRefill()
	r = concSustainedRead(ctx, client, rps, sustainedDur)
	printResult("Read Sustained (at rate)", r, 0)
	waitRefill()
	r = concSustainedRead(ctx, client, rps*overrateMult, overrateDur)
	printResult("Read Sustained (at 1.5x)", r, 0)
}

// ---------------------------------------------------------------------------
// Test 2: Write-only
// ---------------------------------------------------------------------------

func runWriteTest(ctx context.Context, client *gokalshi.Client, tier tierInfo, costs endpointCost) {
	fmt.Println("--- Test 2: Write-Only ---")
	expectedBurst := int(float64(tier.WriteCapacity) / costs.WriteCost)
	rps := float64(tier.WriteRate) / costs.WriteCost
	var allIDs []string

	fmt.Println("\n  Phase 1: Sequential")
	r, ids := seqBurstWrite(ctx, client, expectedBurst*3)
	allIDs = append(allIDs, ids...)
	printResult("Write Burst", r, expectedBurst)
	waitRefill()
	r, ids = seqSustainedWrite(ctx, client, rps, sustainedDur)
	allIDs = append(allIDs, ids...)
	printResult("Write Sustained (at rate)", r, 0)
	waitRefill()
	r, ids = seqSustainedWrite(ctx, client, rps*overrateMult, overrateDur)
	allIDs = append(allIDs, ids...)
	printResult("Write Sustained (at 1.5x)", r, 0)
	waitRefill()

	fmt.Printf("\n  Phase 2: Concurrent (%d goroutines)\n", concurrency)
	r, ids = concBurstWrite(ctx, client, expectedBurst*3)
	allIDs = append(allIDs, ids...)
	printResult("Write Burst", r, expectedBurst)
	waitRefill()
	r, ids = concSustainedWrite(ctx, client, rps, sustainedDur)
	allIDs = append(allIDs, ids...)
	printResult("Write Sustained (at rate)", r, 0)
	waitRefill()
	r, ids = concSustainedWrite(ctx, client, rps*overrateMult, overrateDur)
	allIDs = append(allIDs, ids...)
	printResult("Write Sustained (at 1.5x)", r, 0)

	cleanupOrders(ctx, client, allIDs)
}

// ---------------------------------------------------------------------------
// Test 3: Mixed read + write (concurrent only — sequential can't saturate)
// ---------------------------------------------------------------------------

func runMixedTest(ctx context.Context, client *gokalshi.Client, tier tierInfo, costs endpointCost) {
	fmt.Println("--- Test 3: Mixed Read + Write (Concurrent) ---")
	fmt.Println("Exhaust write bucket while reads at 50% read rate (buckets should be independent)")

	readR, writeR := &result{}, &result{}
	var orderIDs []string
	var idsMu sync.Mutex

	writeBurst := int(float64(tier.WriteCapacity)/costs.WriteCost) + 10

	// Reads throttled to 50% of read budget — well under the limit so any
	// 429s must come from writes leaking into the read bucket.
	readRPS := float64(tier.ReadRate) / costs.ReadCost * 0.5
	readInterval := time.Duration(float64(time.Second) / readRPS)

	var wg sync.WaitGroup
	var readDone atomic.Bool

	// Throttled concurrent reads via ticker + goroutine pool
	readSem := make(chan struct{}, concurrency/2)
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(readInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if readDone.Load() {
					return
				}
				readSem <- struct{}{}
				go func() {
					defer func() { <-readSem }()
					_, err := client.GetBalance(ctx)
					readR.record(err)
				}()
			}
		}
	}()

	fmt.Printf("  Read rate: %.0f req/sec (50%% of budget)\n", readRPS)

	// Concurrent write burst
	writeSem := make(chan struct{}, concurrency/2)
	var writeWg sync.WaitGroup
	for i := 0; i < writeBurst; i++ {
		writeSem <- struct{}{}
		writeWg.Add(1)
		go func() {
			defer writeWg.Done()
			defer func() { <-writeSem }()
			resp, err := client.CreateOrder(ctx, gokalshi.CreateOrderRequest{
				Ticker: testMarket, Side: gokalshi.SideYes, Action: gokalshi.ActionBuy,
				CountFP: ptr("1.00"), YesPriceDollars: orderPrice,
			})
			writeR.record(err)
			if err == nil {
				idsMu.Lock()
				orderIDs = append(orderIDs, resp.Order.OrderID)
				idsMu.Unlock()
			}
		}()
	}
	writeWg.Wait()
	readDone.Store(true)
	wg.Wait()

	fmt.Printf("  Reads:  %d sent, %d success, %d 429s\n",
		readR.sent.Load(), readR.success.Load(), readR.rate429.Load())
	fmt.Printf("  Writes: %d sent, %d success, %d 429s\n",
		writeR.sent.Load(), writeR.success.Load(), writeR.rate429.Load())
	if readR.rate429.Load() == 0 {
		fmt.Println("  PASS: Read bucket unaffected by write exhaustion")
	} else {
		fmt.Println("  FAIL: Reads got 429s during write burst — buckets may not be independent")
	}

	cleanupOrders(ctx, client, orderIDs)
}

// ---------------------------------------------------------------------------
// Test 4: Batch cost billing
// ---------------------------------------------------------------------------

func runBatchTest(ctx context.Context, client *gokalshi.Client, tier tierInfo, costs endpointCost) {
	fmt.Println("--- Test 4: Batch Cost Billing ---")
	fmt.Println("Verifies that each item in a batch is billed individually")
	fmt.Printf("  Write capacity: %d tokens, per-order cost: %.0f tokens\n\n", tier.WriteCapacity, costs.WriteCost)

	batchSizes := []int{1, 5, 10}
	var allIDs []string

	for _, batchSize := range batchSizes {
		expectedBatches := int(float64(tier.WriteCapacity) / (float64(batchSize) * costs.WriteCost))
		totalBatches := expectedBatches * 3 // send 3x to ensure we hit 429

		fmt.Printf("  Batch size %d: expecting ~%d batches before 429 (capacity / (%d × %.0f))\n",
			batchSize, expectedBatches, batchSize, costs.WriteCost)

		waitRefill()

		r := &result{}
		var ids []string
		var mu sync.Mutex
		var wg sync.WaitGroup
		sem := make(chan struct{}, concurrency)
		start := time.Now()

		for i := 0; i < totalBatches; i++ {
			// Build a batch of N identical orders
			orders := make([]gokalshi.CreateOrderRequest, batchSize)
			for j := range orders {
				orders[j] = gokalshi.CreateOrderRequest{
					Ticker:          testMarket,
					Side:            gokalshi.SideYes,
					Action:          gokalshi.ActionBuy,
					CountFP:         ptr("1.00"),
					YesPriceDollars: orderPrice,
				}
			}

			sem <- struct{}{}
			wg.Add(1)
			go func(batch []gokalshi.CreateOrderRequest) {
				defer wg.Done()
				defer func() { <-sem }()
				resp, err := client.BatchCreateOrders(ctx, batch)
				r.record(err)
				if err == nil {
					mu.Lock()
					for _, o := range resp.Orders {
						if o.Order != nil {
							ids = append(ids, o.Order.OrderID)
						}
					}
					mu.Unlock()
				}
			}(orders)
		}
		wg.Wait()
		r.Duration = time.Since(start)

		allIDs = append(allIDs, ids...)
		printResult(fmt.Sprintf("Batch size %d", batchSize), r, expectedBatches)
	}

	cleanupOrders(ctx, client, allIDs)
}

// ---------------------------------------------------------------------------
// Test 5: Client-side limiter prevents 429s
// ---------------------------------------------------------------------------

func runClientLimiterTest(ctx context.Context, cfg *gokalshi.ClientConfig, tier tierInfo, costs endpointCost) {
	fmt.Println("--- Test 5: Client-Side Limiter Stress Test (auto-configured) ---")
	fmt.Println("Uses a client with real auto-configured rate limits (no bypass)")
	fmt.Println("Escalating demand at 100%, 200%, 300% of budget — expects 0 429s at every level")

	limiterClient, err := gokalshi.NewClient(cfg)
	if err != nil {
		log.Fatalf("create limiter client: %v", err)
	}
	defer limiterClient.Close()

	baseReadRPS := float64(tier.ReadRate) / costs.ReadCost
	baseWriteRPS := float64(tier.WriteRate) / costs.WriteCost
	var allIDs []string
	passed := true

	for _, mult := range []float64{1.0, 2.0, 3.0} {
		pct := int(mult * 100)
		readRPS := baseReadRPS * mult
		writeRPS := baseWriteRPS * mult

		fmt.Printf("\n  === %d%% budget ===\n", pct)

		fmt.Printf("  Read at %.0f req/sec (%.0fx budget, %s)\n", readRPS, mult, sustainedDur)
		waitRefill()
		readR := concSustainedRead(ctx, limiterClient, readRPS, sustainedDur)
		printResult(fmt.Sprintf("Read (%dx)", pct), readR, 0)
		if readR.rate429.Load() == 0 {
			fmt.Println("      PASS: 0 read 429s")
		} else {
			fmt.Printf("      FAIL: %d read 429s — client limiter too permissive\n", readR.rate429.Load())
			passed = false
		}

		fmt.Printf("\n  Write at %.0f req/sec (%.0fx budget, %s)\n", writeRPS, mult, sustainedDur)
		waitRefill()
		writeR, ids := concSustainedWrite(ctx, limiterClient, writeRPS, sustainedDur)
		allIDs = append(allIDs, ids...)
		printResult(fmt.Sprintf("Write (%dx)", pct), writeR, 0)
		if writeR.rate429.Load() == 0 {
			fmt.Println("      PASS: 0 write 429s")
		} else {
			fmt.Printf("      FAIL: %d write 429s — client limiter too permissive\n", writeR.rate429.Load())
			passed = false
		}
	}

	if passed {
		fmt.Println("\n  ALL LEVELS PASSED: limiter prevents 429s at 100-300% budget")
	} else {
		fmt.Println("\n  SOME LEVELS FAILED: limiter leaked 429s under stress")
	}

	cleanupOrders(ctx, limiterClient, allIDs)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func waitRefill() {
	fmt.Printf("  Waiting %ds for bucket refill...\n", burstWaitSecs)
	time.Sleep(time.Duration(burstWaitSecs) * time.Second)
}

func cleanupOrders(ctx context.Context, client *gokalshi.Client, orderIDs []string) {
	if len(orderIDs) == 0 {
		return
	}
	fmt.Printf("Cleaning up %d orders...\n", len(orderIDs))
	var cancelled atomic.Int64
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	for _, id := range orderIDs {
		sem <- struct{}{}
		wg.Add(1)
		go func(oid string) {
			defer wg.Done()
			defer func() { <-sem }()
			if _, err := client.CancelOrder(ctx, oid); err == nil {
				cancelled.Add(1)
			}
		}(id)
	}
	wg.Wait()
	fmt.Printf("Cancelled %d/%d orders\n", cancelled.Load(), len(orderIDs))
}

func is429(err error) bool {
	var apiErr *gokalshi.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 429
	}
	var rateLimitErr *gokalshi.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return true
	}
	return false
}

func printResult(name string, r *result, expected int) {
	fmt.Printf("\n    %s:\n", name)
	fmt.Printf("      Requests sent: %d\n", r.sent.Load())
	fmt.Printf("      Succeeded:     %d", r.success.Load())
	if expected > 0 {
		fmt.Printf("  (expected: %d = capacity/cost)", expected)
	}
	fmt.Println()
	fmt.Printf("      429s:          %d\n", r.rate429.Load())
	if r.errs.Load() > 0 {
		fmt.Printf("      Other errors:  %d\n", r.errs.Load())
		r.errMu.Lock()
		for _, s := range r.errSamples {
			fmt.Printf("        %s\n", s)
		}
		r.errMu.Unlock()
	}
	if r.first429.Load() > 0 {
		fmt.Printf("      First 429 at:  #%d\n", r.first429.Load())
	}
	if r.Duration > 0 {
		fmt.Printf("      Duration:      %s\n", r.Duration.Round(time.Millisecond))
		fmt.Printf("      Actual RPS:    %.1f\n", float64(r.sent.Load())/r.Duration.Seconds())
	}
}

func ptr(s string) *string { return &s }

func loadEnv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			_ = godotenv.Load(filepath.Join(dir, ".env"))
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}
