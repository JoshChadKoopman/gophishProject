package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var successHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
})

const testRemoteAddr = "127.0.0.1:"

func reachLimit(t *testing.T, handler http.Handler, limit int) {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = testRemoteAddr
	for i := 0; i < limit; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("no 200 on req %d got %d", i, w.Code)
		}
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after exceeding limit, got %d", w.Code)
	}
}

func TestRateLimitEnforcement(t *testing.T) {
	expectedLimit := 3
	limiter := NewPostLimiter(WithRequestsPerMinute(expectedLimit))
	handler := limiter.Limit(successHandler)
	reachLimit(t, handler, expectedLimit)
}

func TestRateLimitCleanup(t *testing.T) {
	expectedLimit := 3
	limiter := NewPostLimiter(WithRequestsPerMinute(expectedLimit))
	handler := limiter.Limit(successHandler)
	reachLimit(t, handler, expectedLimit)

	bucket, exists := limiter.visitors["127.0.0.1"]
	if !exists {
		t.Fatal("expected visitor bucket to exist")
	}
	bucket.lastSeen = bucket.lastSeen.Add(-limiter.expiry)
	limiter.Cleanup()
	_, exists = limiter.visitors["127.0.0.1"]
	if exists {
		t.Fatal("expected visitor bucket to be cleaned up after expiry")
	}
	reachLimit(t, handler, expectedLimit)
}

// ---------- GET requests bypass POST limiter ----------

func TestGETRequestsNotLimited(t *testing.T) {
	limiter := NewPostLimiter(WithRequestsPerMinute(1))
	handler := limiter.Limit(successHandler)

	// First POST uses the single allowed request
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = testRemoteAddr
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on first POST, got %d", w.Code)
	}

	// Second POST should be rate-limited
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on second POST, got %d", w.Code)
	}

	// GET requests should still pass through even though POST is limited
	for i := 0; i < 10; i++ {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = testRemoteAddr
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 on GET %d, got %d", i, w.Code)
		}
	}
}

// ---------- LimitAll limits all methods ----------

func TestLimitAllEnforcement(t *testing.T) {
	limit := 2
	limiter := NewPostLimiter(WithRequestsPerMinute(limit))
	handler := limiter.LimitAll(successHandler)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = testRemoteAddr

	for i := 0; i < limit; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 on GET %d, got %d", i, w.Code)
		}
	}

	// Next request should be rate-limited
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after exceeding LimitAll, got %d", w.Code)
	}
}

// ---------- Multiple IPs are tracked independently ----------

func TestMultipleIPsIndependent(t *testing.T) {
	limit := 1
	limiter := NewPostLimiter(WithRequestsPerMinute(limit))
	handler := limiter.Limit(successHandler)

	// First IP: use up the quota
	r1 := httptest.NewRequest(http.MethodPost, "/", nil)
	r1.RemoteAddr = "10.0.0.1:"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r1)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for first IP, got %d", w.Code)
	}

	// First IP: should be rate-limited now
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r1)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for first IP, got %d", w.Code)
	}

	// Second IP: should still be allowed
	r2 := httptest.NewRequest(http.MethodPost, "/", nil)
	r2.RemoteAddr = "10.0.0.2:"
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r2)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for second IP, got %d", w.Code)
	}
}

// ---------- Functional options ----------

func TestWithRequestsPerMinute(t *testing.T) {
	limiter := NewPostLimiter(WithRequestsPerMinute(42))
	if limiter.requestLimit != 42 {
		t.Fatalf("expected requestLimit 42, got %d", limiter.requestLimit)
	}
}

func TestWithCleanupInterval(t *testing.T) {
	limiter := NewPostLimiter(WithCleanupInterval(5 * time.Minute))
	if limiter.cleanupInterval != 5*time.Minute {
		t.Fatalf("expected cleanupInterval 5m, got %v", limiter.cleanupInterval)
	}
}

func TestWithExpiry(t *testing.T) {
	limiter := NewPostLimiter(WithExpiry(30 * time.Minute))
	if limiter.expiry != 30*time.Minute {
		t.Fatalf("expected expiry 30m, got %v", limiter.expiry)
	}
}

func TestDefaultOptions(t *testing.T) {
	limiter := NewPostLimiter()
	if limiter.requestLimit != DefaultRequestsPerMinute {
		t.Fatalf("expected default requestLimit %d, got %d", DefaultRequestsPerMinute, limiter.requestLimit)
	}
	if limiter.cleanupInterval != DefaultCleanupInterval {
		t.Fatalf("expected default cleanupInterval %v, got %v", DefaultCleanupInterval, limiter.cleanupInterval)
	}
	if limiter.expiry != DefaultExpiry {
		t.Fatalf("expected default expiry %v, got %v", DefaultExpiry, limiter.expiry)
	}
}

// ---------- RemoteAddr without port ----------

func TestRemoteAddrWithoutPort(t *testing.T) {
	limiter := NewPostLimiter(WithRequestsPerMinute(1))
	handler := limiter.Limit(successHandler)

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = "192.168.1.1" // no port — net.SplitHostPort will fail
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 even when RemoteAddr has no port, got %d", w.Code)
	}
}

// ---------- Cleanup does not remove recent visitors ----------

func TestCleanupKeepsRecentVisitors(t *testing.T) {
	limiter := NewPostLimiter(WithRequestsPerMinute(5))
	handler := limiter.Limit(successHandler)

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = "10.0.0.50:"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	limiter.Cleanup()

	limiter.RLock()
	_, exists := limiter.visitors["10.0.0.50"]
	limiter.RUnlock()
	if !exists {
		t.Fatal("expected recent visitor to survive cleanup")
	}
}
