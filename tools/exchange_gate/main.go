// Command exchange_gate checks whether the Kalshi DEMO exchange is active.
// It prints "active=true" or "active=false" for use in CI gate jobs.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Kalarb/gokalshi"
)

func main() {
	cfg, err := gokalshi.NewClientConfig()
	if err != nil {
		fmt.Println("active=false")
		log.Printf("failed to load config: %v", err)
		os.Exit(0)
	}

	client := gokalshi.NewClient(cfg)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status, err := client.GetExchangeStatus(ctx)
	if err != nil {
		fmt.Println("active=false")
		log.Printf("failed to get exchange status: %v", err)
		os.Exit(0)
	}

	active := status.ExchangeActive || status.TradingActive
	if !active {
		log.Println("Exchange is in maintenance — skipping integration tests")
	}
	fmt.Printf("active=%v\n", active)
}
