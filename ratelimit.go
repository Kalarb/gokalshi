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
	ReadCapacity  float64 // max read tokens; 0 = ReadRate
	WriteCapacity float64 // max write tokens; 0 = WriteRate
	WindowSize    float64 // Deprecated: retained for API compatibility; ignored by the algorithm.
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

// TokenBucketStatus is a read-only snapshot of the bucket state.
type TokenBucketStatus struct {
	ReadTokens  float64
	WriteTokens float64
}

// ReadWriteTokenBucket implements disjoint read/write token bucket rate limiting.
// A request consumes EITHER read tokens OR write tokens, never both.
// Tokens refill continuously at the configured rate up to capacity.
// Goroutine-safe via sync.Mutex.
type ReadWriteTokenBucket struct {
	mu          sync.Mutex
	cfg         TokenBucketConfig
	readTokens  float64
	writeTokens float64
	lastRefill  float64        // timestamp of last refill (monotonic seconds)
	clock       func() float64 // injectable for testing; returns monotonic seconds
}

// readCap returns the effective read capacity.
func (cfg TokenBucketConfig) readCap() float64 {
	if cfg.ReadCapacity > 0 {
		return cfg.ReadCapacity
	}
	return cfg.ReadRate
}

// writeCap returns the effective write capacity.
func (cfg TokenBucketConfig) writeCap() float64 {
	if cfg.WriteCapacity > 0 {
		return cfg.WriteCapacity
	}
	return cfg.WriteRate
}

// NewReadWriteTokenBucket creates a new rate limiter with the given config.
func NewReadWriteTokenBucket(cfg TokenBucketConfig) *ReadWriteTokenBucket {
	b := &ReadWriteTokenBucket{
		cfg:         cfg,
		readTokens:  cfg.readCap(),
		writeTokens: cfg.writeCap(),
		clock:       defaultClock,
	}
	b.lastRefill = b.clock()
	return b
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
		ReadTokens:  b.readTokens,
		WriteTokens: b.writeTokens,
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

// refill adds tokens based on elapsed time since last refill, up to capacity.
// Must be called with mu held.
func (b *ReadWriteTokenBucket) refill() {
	now := b.clock()
	elapsed := now - b.lastRefill
	if elapsed <= 0 {
		return
	}
	b.lastRefill = now

	readCap := b.cfg.readCap()
	b.readTokens += elapsed * b.cfg.ReadRate
	if b.readTokens > readCap {
		b.readTokens = readCap
	}

	writeCap := b.cfg.writeCap()
	b.writeTokens += elapsed * b.cfg.WriteRate
	if b.writeTokens > writeCap {
		b.writeTokens = writeCap
	}
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

// consume deducts tokens.
// Must be called with mu held.
func (b *ReadWriteTokenBucket) consume(readCost, writeCost float64) {
	if readCost > 0 {
		b.readTokens -= readCost
	}
	if writeCost > 0 {
		b.writeTokens -= writeCost
	}
}

// calculateWaitTime estimates how long until enough tokens are available.
// wait = deficit / rate + SafetyPadding
// Must be called with mu held.
func (b *ReadWriteTokenBucket) calculateWaitTime(readCost, writeCost float64) time.Duration {
	var maxWait float64

	if readCost > 0 && b.readTokens < readCost {
		deficit := readCost - b.readTokens
		wait := deficit / b.cfg.ReadRate
		if wait > maxWait {
			maxWait = wait
		}
	}
	if writeCost > 0 && b.writeTokens < writeCost {
		deficit := writeCost - b.writeTokens
		wait := deficit / b.cfg.WriteRate
		if wait > maxWait {
			maxWait = wait
		}
	}

	maxWait += b.cfg.SafetyPadding
	return time.Duration(maxWait * float64(time.Second))
}
