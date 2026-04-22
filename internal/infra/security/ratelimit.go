package security

import (
	"errors"
	"sync"
	"time"
)

// ErrRateLimitExceeded is returned when the rate limit is exceeded.
var ErrRateLimitExceeded = errors.New("rate limit exceeded")

// tokenBucket represents a token bucket for rate limiting.
// It holds the current tokens, maximum tokens, and the next refill time.
type tokenBucket struct {
	tokens    int
	maxTokens int
	refillAt  time.Time
	mu        sync.RWMutex
}

// RateLimiter provides rate limiting using the token bucket algorithm.
// It tracks requests per client key and allows bursts up to a maximum.
type RateLimiter struct {
	buckets    map[string]*tokenBucket
	mu         sync.RWMutex
	refillRate time.Duration
	maxBurst   int
}

// Option is a functional option for configuring the RateLimiter.
type Option func(*RateLimiter)

// WithRefillRate sets the refill rate for tokens.
func WithRefillRate(d time.Duration) Option {
	return func(r *RateLimiter) {
		r.refillRate = d
	}
}

// WithMaxBurst sets the maximum burst size.
func WithMaxBurst(burst int) Option {
	return func(r *RateLimiter) {
		r.maxBurst = burst
	}
}

// NewRateLimiter creates a new RateLimiter with the given options.
// The default refill rate is 1 minute and default burst is 10.
func NewRateLimiter(opts ...Option) *RateLimiter {
	r := &RateLimiter{
		buckets:    make(map[string]*tokenBucket),
		refillRate: time.Minute,
		maxBurst:   10,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// newTokenBucket creates a new token bucket with the given maximum tokens.
func newTokenBucket(maxTokens int) *tokenBucket {
	return &tokenBucket{
		tokens:    maxTokens,
		maxTokens: maxTokens,
		refillAt:  time.Now().Add(time.Minute),
	}
}

// Allow checks if a request is allowed for the given key and cost.
// It returns true if the request is allowed, false if rate limited.
func (r *RateLimiter) Allow(key string, cost int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	bucket := r.buckets[key]
	if bucket == nil {
		bucket = newTokenBucket(r.maxBurst)
		r.buckets[key] = bucket
	}

	// Check if we need to refill tokens
	if time.Now().After(bucket.refillAt) {
		bucket.tokens = r.maxBurst
		bucket.refillAt = time.Now().Add(r.refillRate)
	}

	if bucket.tokens >= cost {
		bucket.tokens -= cost
		return true
	}

	return false
}

// AllowWithCost is an alias for Allow for clarity.
func (r *RateLimiter) AllowWithCost(key string, cost int) bool {
	return r.Allow(key, cost)
}

// GetRemainingTokens returns the number of remaining tokens for the given key.
func (r *RateLimiter) GetRemainingTokens(key string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bucket, ok := r.buckets[key]
	if !ok {
		return r.maxBurst
	}

	// Check if we need to refresh
	if time.Now().After(bucket.refillAt) {
		return r.maxBurst
	}

	return bucket.tokens
}

// Reset resets the rate limiter for the given key.
func (r *RateLimiter) Reset(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.buckets, key)
}

// ResetAll resets all rate limiters.
func (r *RateLimiter) ResetAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.buckets = make(map[string]*tokenBucket)
}

// Cleanup removes expired buckets.
// This should be called periodically to prevent memory leaks.
func (r *RateLimiter) Cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for key, bucket := range r.buckets {
		if now.After(bucket.refillAt) && bucket.tokens >= bucket.maxTokens {
			delete(r.buckets, key)
		}
	}
}

// RateLimitMiddleware creates a rate limiting middleware.
// It uses the given RateLimiter and returns a middleware function.
func RateLimitMiddleware(limiter *RateLimiter, cost int) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			key := GetClientKey(ctx)
			if key == "" {
				key = GetIPAddress(ctx)
			}

			if !limiter.Allow(key, cost) {
				ctx.Set("error", ErrRateLimitExceeded)
				return ErrRateLimitExceeded
			}

			return next(ctx)
		}
	}
}
