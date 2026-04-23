package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/metrics"
	"golang.org/x/time/rate"
)

// DefaultRequestsPerMinute is the number of requests to allow per minute.
// Any requests over this interval will return a HTTP 429 error.
const DefaultRequestsPerMinute = 5

// DefaultCleanupInterval determines how frequently the cleanup routine
// executes.
const DefaultCleanupInterval = 1 * time.Minute

// DefaultExpiry is the amount of time to track a bucket for a particular
// visitor.
const DefaultExpiry = 10 * time.Minute

// DefaultMaxBuckets is the maximum number of per-IP buckets held in memory.
// Once this limit is reached, new IPs are allowed through (fail-open) to
// prevent a memory exhaustion attack from DOSing legitimate users.
const DefaultMaxBuckets = 50_000

type bucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// PostLimiter is a simple rate limiting middleware which only allows n POST
// requests per minute.
type PostLimiter struct {
	visitors        map[string]*bucket
	requestLimit    int
	cleanupInterval time.Duration
	expiry          time.Duration
	maxBuckets      int
	sync.RWMutex
}

// PostLimiterOption is a functional option that allows callers to configure
// the rate limiter.
type PostLimiterOption func(*PostLimiter)

// WithRequestsPerMinute sets the number of requests to allow per minute.
func WithRequestsPerMinute(requestLimit int) PostLimiterOption {
	return func(p *PostLimiter) {
		p.requestLimit = requestLimit
	}
}

// WithCleanupInterval sets the interval between cleaning up stale entries in
// the rate limit client list
func WithCleanupInterval(interval time.Duration) PostLimiterOption {
	return func(p *PostLimiter) {
		p.cleanupInterval = interval
	}
}

// WithExpiry sets the amount of time to store client entries before they are
// considered stale.
func WithExpiry(expiry time.Duration) PostLimiterOption {
	return func(p *PostLimiter) {
		p.expiry = expiry
	}
}

// WithMaxBuckets sets the maximum number of per-IP buckets to hold in memory.
// When the limit is reached, new IPs are allowed through without rate limiting
// (fail-open) to avoid a memory exhaustion attack blocking legitimate users.
func WithMaxBuckets(n int) PostLimiterOption {
	return func(p *PostLimiter) {
		p.maxBuckets = n
	}
}

// NewPostLimiter returns a new instance of a PostLimiter
func NewPostLimiter(opts ...PostLimiterOption) *PostLimiter {
	limiter := &PostLimiter{
		visitors:        make(map[string]*bucket),
		requestLimit:    DefaultRequestsPerMinute,
		cleanupInterval: DefaultCleanupInterval,
		expiry:          DefaultExpiry,
		maxBuckets:      DefaultMaxBuckets,
	}
	for _, opt := range opts {
		opt(limiter)
	}
	go limiter.pollCleanup()
	return limiter
}

func (limiter *PostLimiter) pollCleanup() {
	ticker := time.NewTicker(limiter.cleanupInterval)
	for range ticker.C {
		limiter.Cleanup()
	}
}

// Cleanup removes any buckets that were last seen past the configured expiry.
func (limiter *PostLimiter) Cleanup() {
	limiter.Lock()
	defer limiter.Unlock()
	for ip, bucket := range limiter.visitors {
		if time.Since(bucket.lastSeen) >= limiter.expiry {
			delete(limiter.visitors, ip)
		}
	}
}

// addBucket creates a new rate-limit bucket for the given IP under a write
// lock, using double-checked locking to avoid a race where two goroutines
// both find the bucket absent and both try to create it. If the bucket map
// has reached maxBuckets the function returns nil (caller must fail-open).
func (limiter *PostLimiter) addBucket(ip string) *bucket {
	limiter.Lock()
	defer limiter.Unlock()
	// Double-check: another goroutine may have inserted it while we waited.
	if b, exists := limiter.visitors[ip]; exists {
		return b
	}
	if len(limiter.visitors) >= limiter.maxBuckets {
		log.Warnf("rate limiter bucket cap (%d) reached; allowing %s without rate limiting", limiter.maxBuckets, ip)
		return nil
	}
	limit := rate.NewLimiter(rate.Every(time.Minute/time.Duration(limiter.requestLimit)), limiter.requestLimit)
	b := &bucket{
		limiter: limit,
	}
	limiter.visitors[ip] = b
	return b
}

func (limiter *PostLimiter) allow(ip string) bool {
	// Check if we have a limiter already active for this clientIP
	limiter.RLock()
	b, exists := limiter.visitors[ip]
	limiter.RUnlock()
	if !exists {
		b = limiter.addBucket(ip)
	}
	if b == nil {
		// Bucket cap reached — fail open
		return true
	}
	// Update the lastSeen for this bucket to assist with cleanup
	limiter.Lock()
	defer limiter.Unlock()
	b.lastSeen = time.Now()
	return b.limiter.Allow()
}

// clientIP extracts the real client IP from the request, preferring
// X-Real-IP, then the first address in X-Forwarded-For, then RemoteAddr.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For may be a comma-separated list; take the first entry.
		if idx := strings.IndexByte(xff, ','); idx >= 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// Limit enforces the configured rate limit for POST requests.
// It returns an http.HandlerFunc for compatibility with the current
// Gophish routing setup.
func (limiter *PostLimiter) Limit(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if r.Method == http.MethodPost && !limiter.allow(ip) {
			log.Warnf("Rate limit exceeded for %s on %s", ip, r.URL.Path)
			metrics.RateLimitRejected.Inc()
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// LimitAll enforces the configured rate limit for all HTTP methods.
func (limiter *PostLimiter) LimitAll(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !limiter.allow(ip) {
			metrics.RateLimitRejected.Inc()
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
