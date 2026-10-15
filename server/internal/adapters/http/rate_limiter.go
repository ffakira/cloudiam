package httpadapter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) bool
}

type LocalRateLimiter struct {
	mu      sync.Mutex
	now     func() time.Time
	buckets map[string]localRateLimitBucket
}

type localRateLimitBucket struct {
	window int64
	count  int
}

func NewLocalRateLimiter(now func() time.Time) *LocalRateLimiter {
	if now == nil {
		now = time.Now
	}
	return &LocalRateLimiter{
		now:     now,
		buckets: make(map[string]localRateLimitBucket),
	}
}

func (l *LocalRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) bool {
	if limit <= 0 || window <= 0 {
		return true
	}
	select {
	case <-ctx.Done():
		return false
	default:
	}

	windowID := l.now().UnixNano() / int64(window)

	l.mu.Lock()
	defer l.mu.Unlock()

	bucket := l.buckets[key]
	if bucket.window != windowID {
		bucket = localRateLimitBucket{window: windowID}
	}
	bucket.count++
	l.buckets[key] = bucket
	return bucket.count <= limit
}

type RedisRateLimiter struct {
	counter  redisRateLimitCounter
	fallback RateLimiter
	checker  redisHealthChecker
	prefix   string
	now      func() time.Time
}

type redisRateLimitCounter interface {
	Increment(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

type redisHealthChecker interface {
	Ping(ctx context.Context) *redis.StatusCmd
}

func NewRedisRateLimiter(client *redis.Client, fallback RateLimiter, now func() time.Time) *RedisRateLimiter {
	if fallback == nil {
		fallback = NewLocalRateLimiter(now)
	}
	if now == nil {
		now = time.Now
	}
	return &RedisRateLimiter{
		counter:  redisClientRateLimitCounter{client: client},
		fallback: fallback,
		checker:  client,
		prefix:   "cloudiam:rate_limit",
		now:      now,
	}
}

func NewRedisRateLimiterFromURL(redisURL string, fallback RateLimiter, now func() time.Time) (*RedisRateLimiter, func() error, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse redis rate limiter url: %w", err)
	}
	client := redis.NewClient(opts)
	limiter := NewRedisRateLimiter(client, fallback, now)
	return limiter, client.Close, nil
}

func (l *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) bool {
	if limit <= 0 || window <= 0 {
		return true
	}
	select {
	case <-ctx.Done():
		return false
	default:
	}
	if l == nil || l.counter == nil {
		return true
	}

	windowID := l.now().UnixNano() / int64(window)
	count, err := l.counter.Increment(ctx, fmt.Sprintf("%s:%s:%d", l.prefix, key, windowID), window+time.Second)
	if err != nil {
		return l.fallback.Allow(ctx, key, limit, window)
	}
	return count <= int64(limit)
}

func (l *RedisRateLimiter) Check(ctx context.Context) error {
	if l == nil || l.checker == nil {
		return fmt.Errorf("redis rate limiter is not configured")
	}
	if err := l.checker.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis rate limiter ping failed: %w", err)
	}
	return nil
}

type redisClientRateLimitCounter struct {
	client *redis.Client
}

func (c redisClientRateLimitCounter) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	count, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if count == 1 {
		if err := c.client.Expire(ctx, key, ttl).Err(); err != nil {
			return 0, err
		}
	}
	return count, nil
}
