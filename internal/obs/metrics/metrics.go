package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	registerOnce sync.Once

	httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gophermind_http_requests_total",
			Help: "Total HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)
	httpLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gophermind_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	authLogin = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gophermind_auth_login_total",
			Help: "Total login attempts by result.",
		},
		[]string{"result"},
	)
	authRefresh = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gophermind_auth_refresh_total",
			Help: "Total refresh attempts by result.",
		},
		[]string{"result"},
	)

	mqRetry = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gophermind_mq_retry_total",
			Help: "Total retried MQ messages.",
		},
	)
	mqDLQ = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gophermind_mq_dlq_total",
			Help: "Total MQ messages sent to DLQ.",
		},
	)
	mqIdempotentHit = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gophermind_mq_idempotent_hit_total",
			Help: "Total duplicated MQ messages skipped by idempotency.",
		},
	)

	queryLatency = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gophermind_query_latency_seconds",
			Help:    "Query service latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
	)
	queryRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gophermind_query_requests_total",
			Help: "Total query requests by result, model type and rag flag.",
		},
		[]string{"result", "model_type", "use_rag"},
	)

	streamFirstToken = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gophermind_stream_first_token_seconds",
			Help:    "Stream first token latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
	)
	streamRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gophermind_stream_requests_total",
			Help: "Total stream requests by result, model type and rag flag.",
		},
		[]string{"result", "model_type", "use_rag"},
	)
)

// RegisterAll registers all application business metrics.
func RegisterAll() {
	registerOnce.Do(func() {
		prometheus.MustRegister(
			httpRequests,
			httpLatency,
			authLogin,
			authRefresh,
			mqRetry,
			mqDLQ,
			mqIdempotentHit,
			queryLatency,
			queryRequests,
			streamFirstToken,
			streamRequests,
		)
	})
}

// ObserveHTTPRequest records request count and latency.
func ObserveHTTPRequest(method string, path string, status string, d time.Duration) {
	httpRequests.WithLabelValues(method, path, status).Inc()
	httpLatency.WithLabelValues(method, path).Observe(d.Seconds())
}

// IncAuthLogin records login result.
func IncAuthLogin(success bool) {
	result := "success"
	if !success {
		result = "fail"
	}
	authLogin.WithLabelValues(result).Inc()
}

// IncAuthRefresh records refresh result.
func IncAuthRefresh(success bool) {
	result := "success"
	if !success {
		result = "fail"
	}
	authRefresh.WithLabelValues(result).Inc()
}

// IncMQRetry increments retry counter.
func IncMQRetry() { mqRetry.Inc() }

// IncMQDLQ increments DLQ counter.
func IncMQDLQ() { mqDLQ.Inc() }

// IncMQIdempotentHit increments duplicate-hit counter.
func IncMQIdempotentHit() { mqIdempotentHit.Inc() }

// ObserveQueryLatency records query latency.
func ObserveQueryLatency(d time.Duration) {
	queryLatency.Observe(d.Seconds())
}

// IncQueryRequest records query request result.
func IncQueryRequest(success bool, modelType string, useRAG bool) {
	result := "success"
	if !success {
		result = "fail"
	}
	queryRequests.WithLabelValues(result, labelModelType(modelType), labelBool(useRAG)).Inc()
}

// ObserveStreamFirstToken records first token latency.
func ObserveStreamFirstToken(d time.Duration) {
	streamFirstToken.Observe(d.Seconds())
}

// IncStreamRequest records stream request result.
func IncStreamRequest(success bool, modelType string, useRAG bool) {
	result := "success"
	if !success {
		result = "fail"
	}
	streamRequests.WithLabelValues(result, labelModelType(modelType), labelBool(useRAG)).Inc()
}

func labelBool(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func labelModelType(v string) string {
	if v == "" {
		return "auto"
	}
	return v
}
