package gokalshi

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TokenBucketConfig holds rate limiter settings.
type TokenBucketConfig struct {
	ReadRate      float64 // read token refill rate (tokens per second)
	WriteRate     float64 // write token refill rate (tokens per second)
	ReadCapacity  float64 // max read tokens; 0 = ReadRate * WindowSize
	WriteCapacity float64 // max write tokens; 0 = WriteRate * WindowSize
	WindowSize    float64 // sliding window duration in seconds (default: 1.0)
	SafetyPadding float64 // extra wait buffer in seconds (default: 0.1)
}

// DefaultTokenBucketConfig returns production defaults matching Kalshi Basic tier.
func DefaultTokenBucketConfig() TokenBucketConfig {
	return TokenBucketConfig{
		ReadRate:      200.0,
		WriteRate:     100.0,
		ReadCapacity:  200.0,
		WriteCapacity: 100.0,
		WindowSize:    1.0,
		SafetyPadding: 0.1,
	}
}

// tokenRecord stores a single consumption event in the sliding window.
type tokenRecord struct {
	timestamp float64
	cost      float64
}

// TokenBucketStatus is a read-only snapshot of the bucket state.
type TokenBucketStatus struct {
	ReadTokens      float64
	WriteTokens     float64
	ReadHistoryLen  int
	WriteHistoryLen int
}

// ReadWriteTokenBucket implements disjoint read/write sliding-window rate limiting.
// A request consumes EITHER read tokens OR write tokens, never both.
// Goroutine-safe via sync.Mutex.
type ReadWriteTokenBucket struct {
	mu           sync.Mutex
	cfg          TokenBucketConfig
	readTokens   float64
	writeTokens  float64
	readHistory  []tokenRecord
	writeHistory []tokenRecord
	clock        func() float64 // injectable for testing; returns monotonic seconds
}

// readCap returns the effective read capacity.
func (cfg TokenBucketConfig) readCap() float64 {
	if cfg.ReadCapacity > 0 {
		return cfg.ReadCapacity
	}
	return cfg.ReadRate * cfg.WindowSize
}

// writeCap returns the effective write capacity.
func (cfg TokenBucketConfig) writeCap() float64 {
	if cfg.WriteCapacity > 0 {
		return cfg.WriteCapacity
	}
	return cfg.WriteRate * cfg.WindowSize
}

// NewReadWriteTokenBucket creates a new rate limiter with the given config.
func NewReadWriteTokenBucket(cfg TokenBucketConfig) *ReadWriteTokenBucket {
	return &ReadWriteTokenBucket{
		cfg:         cfg,
		readTokens:  cfg.readCap(),
		writeTokens: cfg.writeCap(),
		clock:       defaultClock,
	}
}

func defaultClock() float64 {
	return float64(time.Now().UnixNano()) / 1e9
}

// Acquire blocks until tokens are available, then consumes them.
// For read requests: readCost > 0, writeCost = 0.
// For write requests: readCost = 0, writeCost > 0.
func (b *ReadWriteTokenBucket) Acquire(ctx context.Context, readCost, writeCost float64) error {
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("rate limiter acquire cancelled: %w", err)
		}

		wait := b.tryAcquireOrWait(readCost, writeCost)
		if wait == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("rate limiter acquire cancelled: %w", ctx.Err())
		case <-time.After(wait):
			// Retry after waiting
		}
	}
}

// TryAcquire attempts to consume tokens without blocking.
// Returns true if tokens were consumed, false otherwise.
func (b *ReadWriteTokenBucket) TryAcquire(readCost, writeCost float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill()

	if !b.canProceed(readCost, writeCost) {
		return false
	}

	b.consume(readCost, writeCost)
	return true
}

// GetWaitTime returns the estimated wait time until tokens become available.
func (b *ReadWriteTokenBucket) GetWaitTime(readCost, writeCost float64) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill()

	if b.canProceed(readCost, writeCost) {
		return 0
	}

	return b.calculateWaitTime(readCost, writeCost)
}

// Status returns a snapshot of the current token state.
func (b *ReadWriteTokenBucket) Status() TokenBucketStatus {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill()

	return TokenBucketStatus{
		ReadTokens:      b.readTokens,
		WriteTokens:     b.writeTokens,
		ReadHistoryLen:  len(b.readHistory),
		WriteHistoryLen: len(b.writeHistory),
	}
}

// tryAcquireOrWait attempts to acquire, returning 0 on success or the wait duration.
func (b *ReadWriteTokenBucket) tryAcquireOrWait(readCost, writeCost float64) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill()

	if b.canProceed(readCost, writeCost) {
		b.consume(readCost, writeCost)
		return 0
	}

	return b.calculateWaitTime(readCost, writeCost)
}

// refill expires old entries from the sliding window and restores tokens.
// Must be called with mu held.
func (b *ReadWriteTokenBucket) refill() {
	now := b.clock()
	cutoff := now - b.cfg.WindowSize

	b.readTokens, b.readHistory = refillBucket(
		b.readTokens, b.cfg.readCap(), b.readHistory, cutoff,
	)
	b.writeTokens, b.writeHistory = refillBucket(
		b.writeTokens, b.cfg.writeCap(), b.writeHistory, cutoff,
	)
}

// refillBucket expires old records and restores their token costs.
func refillBucket(tokens, maxTokens float64, history []tokenRecord, cutoff float64) (float64, []tokenRecord) {
	firstValid := len(history) // assume all expired
	for i, rec := range history {
		if rec.timestamp > cutoff {
			firstValid = i
			break
		}
		tokens += rec.cost
	}

	if tokens > maxTokens {
		tokens = maxTokens
	}

	if firstValid == 0 {
		return tokens, history
	}

	// Compact slice to avoid holding expired records in memory.
	remaining := history[firstValid:]
	compacted := make([]tokenRecord, len(remaining))
	copy(compacted, remaining)

	return tokens, compacted
}

// canProceed checks whether sufficient tokens exist.
// Must be called with mu held.
func (b *ReadWriteTokenBucket) canProceed(readCost, writeCost float64) bool {
	if readCost > 0 && b.readTokens < readCost {
		return false
	}
	if writeCost > 0 && b.writeTokens < writeCost {
		return false
	}
	return true
}

// consume deducts tokens and records the consumption.
// Must be called with mu held.
func (b *ReadWriteTokenBucket) consume(readCost, writeCost float64) {
	now := b.clock()

	if readCost > 0 {
		b.readTokens -= readCost
		b.readHistory = append(b.readHistory, tokenRecord{timestamp: now, cost: readCost})
	}
	if writeCost > 0 {
		b.writeTokens -= writeCost
		b.writeHistory = append(b.writeHistory, tokenRecord{timestamp: now, cost: writeCost})
	}
}

// calculateWaitTime estimates how long until enough tokens are available.
// Must be called with mu held.
func (b *ReadWriteTokenBucket) calculateWaitTime(readCost, writeCost float64) time.Duration {
	now := b.clock()
	var maxWait float64

	if readCost > 0 && b.readTokens < readCost {
		wait := waitForBucket(b.readHistory, b.readTokens, readCost, now, b.cfg.WindowSize)
		if wait > maxWait {
			maxWait = wait
		}
	}
	if writeCost > 0 && b.writeTokens < writeCost {
		wait := waitForBucket(b.writeHistory, b.writeTokens, writeCost, now, b.cfg.WindowSize)
		if wait > maxWait {
			maxWait = wait
		}
	}

	maxWait += b.cfg.SafetyPadding
	return time.Duration(maxWait * float64(time.Second))
}

// waitForBucket calculates how long until enough tokens expire from the window.
func waitForBucket(history []tokenRecord, currentTokens, needed float64, now, windowSize float64) float64 {
	deficit := needed - currentTokens
	var accumulated float64

	for _, rec := range history {
		accumulated += rec.cost
		if accumulated >= deficit {
			// This record's expiry time will free enough tokens
			expiresAt := rec.timestamp + windowSize
			wait := expiresAt - now
			if wait < 0 {
				return 0
			}
			return wait
		}
	}

	// Should not happen if called correctly, but return a safe default
	return windowSize
}
