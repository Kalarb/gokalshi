package gokalshi

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testBucket(readRate, writeRate float64) *ReadWriteTokenBucket {
	b := NewReadWriteTokenBucket(TokenBucketConfig{
		ReadRate:      readRate,
		WriteRate:     writeRate,
		SafetyPadding: 0.0,
	})
	// Freeze clock so micro-elapsed real time doesn't cause spurious refills.
	now := b.clock()
	b.clock = func() float64 { return now }
	return b
}

func testBucketWithClock(readRate, writeRate float64, clock func() float64) *ReadWriteTokenBucket {
	b := testBucket(readRate, writeRate)
	b.clock = clock
	b.lastRefill = clock()
	return b
}

func TestNewReadWriteTokenBucket(t *testing.T) {
	cfg := DefaultTokenBucketConfig()
	b := NewReadWriteTokenBucket(cfg)

	status := b.Status()
	assert.Equal(t, cfg.ReadRate, status.ReadTokens)
	assert.Equal(t, cfg.WriteRate, status.WriteTokens)
}

func TestTryAcquire_Read_Success(t *testing.T) {
	b := testBucket(5.0, 5.0)

	ok := b.TryAcquire(1.0, 0)
	assert.True(t, ok)

	status := b.Status()
	assert.Equal(t, 4.0, status.ReadTokens)
	assert.Equal(t, 5.0, status.WriteTokens)
}

func TestTryAcquire_Write_Success(t *testing.T) {
	b := testBucket(5.0, 5.0)

	ok := b.TryAcquire(0, 1.0)
	assert.True(t, ok)

	status := b.Status()
	assert.Equal(t, 5.0, status.ReadTokens)
	assert.Equal(t, 4.0, status.WriteTokens)
}

func TestTryAcquire_Read_Exhausted(t *testing.T) {
	b := testBucket(2.0, 5.0)

	assert.True(t, b.TryAcquire(1.0, 0))
	assert.True(t, b.TryAcquire(1.0, 0))
	assert.False(t, b.TryAcquire(1.0, 0))
}

func TestTryAcquire_Write_Exhausted(t *testing.T) {
	b := testBucket(5.0, 2.0)

	assert.True(t, b.TryAcquire(0, 1.0))
	assert.True(t, b.TryAcquire(0, 1.0))
	assert.False(t, b.TryAcquire(0, 1.0))
}

func TestTryAcquire_ReadDoesNotAffectWrite(t *testing.T) {
	b := testBucket(2.0, 2.0)

	b.TryAcquire(1.0, 0)
	b.TryAcquire(1.0, 0)

	assert.True(t, b.TryAcquire(0, 1.0))
}

func TestTryAcquire_WriteDoesNotAffectRead(t *testing.T) {
	b := testBucket(2.0, 2.0)

	b.TryAcquire(0, 1.0)
	b.TryAcquire(0, 1.0)

	assert.True(t, b.TryAcquire(1.0, 0))
}

func TestTryAcquire_FractionalCosts(t *testing.T) {
	b := testBucket(5.0, 1.0)

	for i := 0; i < 5; i++ {
		assert.True(t, b.TryAcquire(0, 0.2), "cancel %d should succeed", i)
	}
	assert.False(t, b.TryAcquire(0, 0.2))
}

func TestRefill_FullAfterElapsed(t *testing.T) {
	now := 100.0
	b := testBucketWithClock(5.0, 5.0, func() float64 { return now })

	for i := 0; i < 5; i++ {
		b.TryAcquire(1.0, 0)
	}
	assert.False(t, b.TryAcquire(1.0, 0))

	// Advance 1.1s: refill = 5.0 * 1.1 = 5.5, capped at 5.0
	now = 101.1
	assert.True(t, b.TryAcquire(1.0, 0))

	status := b.Status()
	assert.Equal(t, 4.0, status.ReadTokens)
}

func TestAcquire_BlocksUntilAvailable(t *testing.T) {
	now := 100.0
	mu := sync.Mutex{}
	b := testBucketWithClock(1.0, 1.0, func() float64 {
		mu.Lock()
		defer mu.Unlock()
		return now
	})

	b.TryAcquire(1.0, 0)

	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		done <- b.Acquire(ctx, 1.0, 0)
	}()

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	now = 101.2
	mu.Unlock()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Acquire did not unblock after clock advance")
	}
}

func TestAcquire_RespectsContext(t *testing.T) {
	b := testBucket(1.0, 1.0)
	b.TryAcquire(1.0, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := b.Acquire(ctx, 1.0, 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

func TestGetWaitTime_Zero(t *testing.T) {
	b := testBucket(5.0, 5.0)
	wait := b.GetWaitTime(1.0, 0)
	assert.Equal(t, time.Duration(0), wait)
}

func TestGetWaitTime_Positive(t *testing.T) {
	b := testBucket(1.0, 1.0)
	b.TryAcquire(1.0, 0)

	wait := b.GetWaitTime(1.0, 0)
	assert.Greater(t, wait, time.Duration(0))
}

func TestStatus_Snapshot(t *testing.T) {
	b := testBucket(10.0, 5.0)
	b.TryAcquire(3.0, 0)
	b.TryAcquire(0, 2.0)

	status := b.Status()
	assert.Equal(t, 7.0, status.ReadTokens)
	assert.Equal(t, 3.0, status.WriteTokens)
}

func TestConcurrentAcquire(t *testing.T) {
	b := testBucket(10.0, 10.0)
	ctx := context.Background()

	var wg sync.WaitGroup
	errors := make(chan error, 20)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := b.Acquire(ctx, 1.0, 0); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		require.NoError(t, err)
	}

	status := b.Status()
	assert.Equal(t, 0.0, status.ReadTokens)
	assert.Equal(t, 10.0, status.WriteTokens)
}

func TestDefaultTokenBucketConfig(t *testing.T) {
	cfg := DefaultTokenBucketConfig()
	assert.Equal(t, 200.0, cfg.ReadRate)
	assert.Equal(t, 100.0, cfg.WriteRate)
	assert.Equal(t, 200.0, cfg.ReadCapacity)
	assert.Equal(t, 100.0, cfg.WriteCapacity)
	assert.Equal(t, 1.0, cfg.WindowSize)
	assert.Equal(t, 0.1, cfg.SafetyPadding)
}

func TestGetWaitTime_WriteBucket(t *testing.T) {
	b := testBucket(5.0, 1.0)
	b.TryAcquire(0, 1.0)

	wait := b.GetWaitTime(0, 1.0)
	assert.Greater(t, wait, time.Duration(0))
}

func TestCalculateWaitTime_BothBuckets(t *testing.T) {
	b := testBucket(1.0, 1.0)
	b.TryAcquire(1.0, 0)
	b.TryAcquire(0, 1.0)

	wait := b.GetWaitTime(1.0, 1.0)
	assert.Greater(t, wait, time.Duration(0))
}

// --- New token-bucket-specific tests ---

func TestRefill_Continuous(t *testing.T) {
	now := 100.0
	b := testBucketWithClock(5.0, 5.0, func() float64 { return now })

	// Consume all 5 read tokens
	for i := 0; i < 5; i++ {
		b.TryAcquire(1.0, 0)
	}
	assert.False(t, b.TryAcquire(1.0, 0))

	// Advance 0.5s: continuous refill should give 5.0 * 0.5 = 2.5 tokens
	now = 100.5
	status := b.Status()
	assert.InDelta(t, 2.5, status.ReadTokens, 0.001)

	// Can acquire 2 but not 3
	assert.True(t, b.TryAcquire(1.0, 0))
	assert.True(t, b.TryAcquire(1.0, 0))
	assert.False(t, b.TryAcquire(1.0, 0))
}

func TestRefill_CapsAtCapacity(t *testing.T) {
	now := 100.0
	b := testBucketWithClock(10.0, 10.0, func() float64 { return now })

	// Consume 5 of 10 read tokens
	b.TryAcquire(5.0, 0)

	// Advance 10 seconds (would add 100 tokens, but cap is 10)
	now = 110.0
	status := b.Status()
	assert.Equal(t, 10.0, status.ReadTokens)
	assert.Equal(t, 10.0, status.WriteTokens)
}

func TestGetWaitTime_ExactCalculation(t *testing.T) {
	now := 100.0
	b := testBucketWithClock(10.0, 10.0, func() float64 { return now })
	b.cfg.SafetyPadding = 0.05

	// Consume all 10 read tokens
	for i := 0; i < 10; i++ {
		b.TryAcquire(1.0, 0)
	}

	// Need 5 tokens, have 0. Token bucket: wait = 5/10 + 0.05 = 0.55s
	wait := b.GetWaitTime(5.0, 0)
	expected := time.Duration(0.55 * float64(time.Second))
	assert.Equal(t, expected, wait)
}

func TestCapacityIndependentFromRate(t *testing.T) {
	cfg := TokenBucketConfig{
		ReadRate:      10.0,
		WriteRate:     10.0,
		ReadCapacity:  50.0,
		WriteCapacity: 25.0,
	}
	b := NewReadWriteTokenBucket(cfg)
	status := b.Status()
	assert.Equal(t, 50.0, status.ReadTokens)
	assert.Equal(t, 25.0, status.WriteTokens)
}

func TestNewReadWriteTokenBucket_PanicsOnZeroRate(t *testing.T) {
	assert.Panics(t, func() {
		NewReadWriteTokenBucket(TokenBucketConfig{ReadRate: 0, WriteRate: 1})
	})
	assert.Panics(t, func() {
		NewReadWriteTokenBucket(TokenBucketConfig{ReadRate: 1, WriteRate: 0})
	})
	assert.Panics(t, func() {
		NewReadWriteTokenBucket(TokenBucketConfig{ReadRate: -1, WriteRate: 1})
	})
}
