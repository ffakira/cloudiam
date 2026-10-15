package httpadapter

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestLocalRateLimiter(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	limiter := NewLocalRateLimiter(func() time.Time { return now })

	if !limiter.Allow(context.Background(), "device:127.0.0.1", 2, time.Minute) {
		t.Fatal("first request was rate limited")
	}
	if !limiter.Allow(context.Background(), "device:127.0.0.1", 2, time.Minute) {
		t.Fatal("second request was rate limited")
	}
	if limiter.Allow(context.Background(), "device:127.0.0.1", 2, time.Minute) {
		t.Fatal("third request should be rate limited")
	}
	if !limiter.Allow(context.Background(), "device:127.0.0.2", 2, time.Minute) {
		t.Fatal("independent key was rate limited")
	}

	now = now.Add(time.Minute)
	if !limiter.Allow(context.Background(), "device:127.0.0.1", 2, time.Minute) {
		t.Fatal("new window should allow request")
	}
}

func TestLocalRateLimiterConcurrentRequests(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	limiter := NewLocalRateLimiter(func() time.Time { return now })

	const limit = 10
	const requests = 64
	var wg sync.WaitGroup
	results := make(chan bool, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- limiter.Allow(context.Background(), "device:127.0.0.1", limit, time.Minute)
		}()
	}
	wg.Wait()
	close(results)

	allowed := 0
	for result := range results {
		if result {
			allowed++
		}
	}
	if allowed != limit {
		t.Fatalf("allowed requests = %d, want %d", allowed, limit)
	}
}

func TestRateLimiterRejectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	local := NewLocalRateLimiter(nil)
	if local.Allow(ctx, "device:127.0.0.1", 1, time.Minute) {
		t.Fatal("local limiter allowed canceled context")
	}

	redis := &RedisRateLimiter{
		counter: succeedingRedisRateLimitCounter{},
		fallback: NewLocalRateLimiter(func() time.Time {
			return time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
		}),
		prefix: "cloudiam:rate_limit",
		now: func() time.Time {
			return time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
		},
	}
	if redis.Allow(ctx, "device:127.0.0.1", 1, time.Minute) {
		t.Fatal("redis limiter allowed canceled context")
	}
}

func TestRedisRateLimiterFallsBackToLocal(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	limiter := &RedisRateLimiter{
		counter: failingRedisRateLimitCounter{},
		fallback: NewLocalRateLimiter(func() time.Time {
			return now
		}),
		prefix: "cloudiam:rate_limit",
		now: func() time.Time {
			return now
		},
	}

	if !limiter.Allow(context.Background(), "device:127.0.0.1", 1, time.Minute) {
		t.Fatal("fallback first request was rate limited")
	}
	if limiter.Allow(context.Background(), "device:127.0.0.1", 1, time.Minute) {
		t.Fatal("fallback should rate limit after Redis failure")
	}
}

type failingRedisRateLimitCounter struct{}

func (failingRedisRateLimitCounter) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	return 0, errors.New("redis unavailable")
}

type succeedingRedisRateLimitCounter struct{}

func (succeedingRedisRateLimitCounter) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	return 1, nil
}
