// Metryki aplikacji (pomocnicze; główne dane pochodzą z k6 i docker stats).
package metrics

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	reqTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "api_requests_total"},
		[]string{"iface", "op", "code"},
	)
	reqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "api_request_duration_seconds", Buckets: prometheus.ExponentialBuckets(0.0005, 2, 16)},
		[]string{"iface", "op"},
	)
)

func init() { prometheus.MustRegister(reqTotal, reqDuration) }

func Observe(iface, op, code string, d time.Duration) {
	reqTotal.WithLabelValues(iface, op, code).Inc()
	reqDuration.WithLabelValues(iface, op).Observe(d.Seconds())
}

// Serve wystawia /metrics na osobnym porcie (bez TLS i JWT).
func Serve(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	go func() { log.Println("metrics:", http.ListenAndServe(addr, mux)) }()
}

type StatusRecorder struct {
	http.ResponseWriter
	Code int
}

func (s *StatusRecorder) WriteHeader(c int) { s.Code = c; s.ResponseWriter.WriteHeader(c) }

// HTTP mierzy żądania; opFn wyznacza etykietę operacji po obsłużeniu żądania.
func HTTP(iface string, opFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &StatusRecorder{ResponseWriter: w, Code: 200}
			t := time.Now()
			next.ServeHTTP(rec, r)
			Observe(iface, opFn(r), strconv.Itoa(rec.Code), time.Since(t))
		})
	}
}
