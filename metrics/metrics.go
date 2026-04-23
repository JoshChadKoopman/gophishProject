// Package metrics registers Prometheus metrics for the Nivoxis / GoPhish
// platform and provides an Instrument middleware that records per-request
// counters and latency histograms.
package metrics

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal counts every HTTP request by method, path pattern, and
	// response status code.
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gophish_http_requests_total",
		Help: "Total HTTP requests by method, path pattern, and status code.",
	}, []string{"method", "path", "status"})

	// HTTPDuration measures request latency per method and path pattern.
	HTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gophish_http_request_duration_seconds",
		Help:    "HTTP request latency.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	// EmailsSentTotal counts successfully sent phishing emails.
	EmailsSentTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gophish_emails_sent_total",
		Help: "Total phishing emails sent successfully.",
	})

	// EmailErrorsTotal counts failed phishing email send attempts.
	EmailErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gophish_email_errors_total",
		Help: "Total phishing email send failures.",
	})

	// RateLimitRejected counts requests rejected by the rate limiter.
	RateLimitRejected = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gophish_ratelimit_rejected_total",
		Help: "Total requests rejected by the rate limiter.",
	})

	// WorkerLastRunTimestamp records the Unix timestamp of the last campaign
	// worker tick. A stale value here surfaces worker health issues.
	WorkerLastRunTimestamp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gophish_worker_last_run_timestamp_seconds",
		Help: "Unix timestamp of the last campaign worker tick.",
	})

	// ActiveCampaigns tracks the number of currently in-progress campaigns.
	ActiveCampaigns = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gophish_campaigns_active",
		Help: "Number of currently in-progress campaigns.",
	})
)

// statusRecorder wraps http.ResponseWriter to capture the status code written
// by the downstream handler.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Instrument wraps next with Prometheus HTTP instrumentation, labelled by the
// given path pattern string (e.g. "admin", "phish").
func Instrument(pattern string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		timer := prometheus.NewTimer(HTTPDuration.WithLabelValues(r.Method, pattern))
		next.ServeHTTP(rw, r)
		timer.ObserveDuration()
		HTTPRequestsTotal.WithLabelValues(
			r.Method, pattern, strconv.Itoa(rw.status),
		).Inc()
	})
}
