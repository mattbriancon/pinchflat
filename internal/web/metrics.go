package web

// Prometheus metrics on /metrics (ENABLE_PROMETHEUS), replacing PromEx.
// Names changed from PromEx's; see docs/go-port/STRATEGY.md §4.6.
// Hand-written W4 infrastructure.

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	Registry = prometheus.NewRegistry()

	httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "pinchflat_http_requests_total", Help: "HTTP requests by route pattern, method and status.",
	}, []string{"route", "method", "status"})
	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "pinchflat_http_request_duration_seconds", Help: "HTTP request latency.", Buckets: prometheus.DefBuckets,
	}, []string{"route", "method"})

	// JobsTotal and JobDuration are updated from the job runner's OnEvent hook.
	JobsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "pinchflat_jobs_total", Help: "Finished jobs by queue, worker and outcome.",
	}, []string{"queue", "worker", "outcome"})
	JobDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "pinchflat_job_duration_seconds", Help: "Job run time.", Buckets: []float64{.1, .5, 1, 5, 15, 60, 300, 900, 3600},
	}, []string{"queue", "worker"})
)

func init() {
	Registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		httpRequests, httpDuration, JobsTotal, JobDuration,
	)
}

// MetricsHandler serves the registry in Prometheus/OpenMetrics text format.
func MetricsHandler() http.Handler {
	return promhttp.HandlerFor(Registry, promhttp.HandlerOpts{EnableOpenMetrics: true})
}

// instrument records request counts and latency by route pattern (not raw
// path, to keep label cardinality bounded).
func instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		route := routeLabel(r.URL.Path)
		httpRequests.WithLabelValues(route, r.Method, strconv.Itoa(rec.status)).Inc()
		httpDuration.WithLabelValues(route, r.Method).Observe(time.Since(start).Seconds())
	})
}

// routeLabel collapses ids/uuids so /sources/12/edit -> /sources/:id/edit.
func routeLabel(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		if s == "" {
			continue
		}
		if _, err := strconv.Atoi(s); err == nil || len(s) == 36 && strings.Count(s, "-") == 4 {
			parts[i] = ":id"
		}
	}
	return strings.Join(parts, "/")
}
