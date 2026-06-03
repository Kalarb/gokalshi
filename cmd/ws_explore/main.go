// Command ws_explore characterizes Kalshi WebSocket behavior for global
// (ticker-less) subscriptions. It connects, subscribes globally, then tests
// what happens when update_subscription (add_markets) is sent on a global sub.
//
// Usage:
//
//	KALSHI_ENVIRONMENT=PROD go run ./cmd/ws_explore
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Kalarb/gokalshi"
	"github.com/joho/godotenv"
	"nhooyr.io/websocket"
)

type wsCommand struct {
	ID     int    `json:"id"`
	Cmd    string `json:"cmd"`
	Params any    `json:"params,omitempty"`
}

type subscribeParams struct {
	Channels []string `json:"channels"`
}

type updateSubParams struct {
	SIDs          []int  `json:"sids"`
	MarketTickers []string `json:"market_tickers"`
	Action        string   `json:"action"`
}

// incoming is a minimal envelope for parsing responses.
type incoming struct {
	Type string          `json:"type"`
	ID   int             `json:"id,omitempty"`
	SID  int             `json:"sid,omitempty"`
	Seq  int             `json:"seq,omitempty"`
	Msg  json.RawMessage `json:"msg,omitempty"`
}

func send(ctx context.Context, conn *websocket.Conn, cmd wsCommand) {
	data, _ := json.Marshal(cmd)
	fmt.Fprintf(os.Stderr, ">>> SEND (id=%d): %s\n", cmd.ID, string(data))
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		log.Fatalf("write: %v", err)
	}
}

func readOne(ctx context.Context, conn *websocket.Conn) (incoming, []byte) {
	_, data, err := conn.Read(ctx)
	if err != nil {
		log.Fatalf("read: %v", err)
	}
	var msg incoming
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Fatalf("unmarshal: %v", err)
	}
	return msg, data
}

func printMsg(n int, data []byte) {
	var raw json.RawMessage
	if json.Unmarshal(data, &raw) == nil {
		pretty, _ := json.MarshalIndent(raw, "", "  ")
		fmt.Printf("[msg %d] %s\n\n", n, string(pretty))
	} else {
		fmt.Printf("[msg %d] %s\n\n", n, string(data))
	}
}

func main() {
	_ = godotenv.Load()

	cfg, err := gokalshi.NewClientConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	url := cfg.WSBaseURL + "/trade-api/ws/v2"
	headers, err := cfg.Credentials.RequestHeaders("GET", "/trade-api/ws/v2")
	if err != nil {
		log.Fatalf("auth headers: %v", err)
	}
	httpHeaders := http.Header{}
	for k, v := range headers {
		httpHeaders.Set(k, v)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPHeader: httpHeaders,
	})
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "done") }()
	conn.SetReadLimit(10 * 1024 * 1024)

	fmt.Fprintf(os.Stderr, "connected to %s\n\n", url)

	// Step 1: Subscribe to "trade" globally (no tickers).
	fmt.Fprintf(os.Stderr, "=== STEP 1: Global subscribe to 'trade' ===\n")
	send(ctx, conn, wsCommand{
		ID:  1,
		Cmd: "subscribe",
		Params: subscribeParams{
			Channels: []string{"trade"},
		},
	})

	// Read until we get the "subscribed" response and capture the SID.
	var tradeSID int
	msgCount := 0
	for {
		msg, data := readOne(ctx, conn)
		msgCount++
		printMsg(msgCount, data)
		if msg.Type == "subscribed" && msg.ID == 1 {
			// Parse SID from msg body.
			var body struct{ SID int `json:"sid"` }
			_ = json.Unmarshal(msg.Msg, &body)
			tradeSID = body.SID
			fmt.Fprintf(os.Stderr, ">>> Got SID=%d for trade channel\n\n", tradeSID)
			break
		}
	}

	// Read a few trade messages to confirm global data is flowing.
	fmt.Fprintf(os.Stderr, "=== Reading 3 trade messages to confirm global data flow ===\n")
	for i := 0; i < 3; i++ {
		_, data := readOne(ctx, conn)
		msgCount++
		printMsg(msgCount, data)
	}

	// Step 2: Send update_subscription with add_markets for a specific ticker.
	ticker := "KXBTC-100K"
	fmt.Fprintf(os.Stderr, "=== STEP 2: update_subscription add_markets ticker=%s on SID=%d ===\n", ticker, tradeSID)
	send(ctx, conn, wsCommand{
		ID:  2,
		Cmd: "update_subscription",
		Params: updateSubParams{
			SIDs:          []int{tradeSID},
			MarketTickers: []string{ticker},
			Action:        "add_markets",
		},
	})

	// Step 3: Read messages for 10 seconds to see what changes.
	fmt.Fprintf(os.Stderr, "\n=== STEP 3: Listening 10s after add_markets ===\n\n")
	listenCtx, listenCancel := context.WithTimeout(ctx, 10*time.Second)
	defer listenCancel()

	for {
		_, data, err := conn.Read(listenCtx)
		if err != nil {
			if listenCtx.Err() != nil {
				fmt.Fprintf(os.Stderr, "\n--- timeout reached, %d total messages ---\n", msgCount)
				break
			}
			log.Fatalf("read: %v", err)
		}
		msgCount++
		printMsg(msgCount, data)
	}
}
