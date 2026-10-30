package middleware

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/metrics"
)

func LogRequests(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			log.Printf("%s %s %s (%s)", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))

			// Record request latency into a histogram
			metrics.GetOrCreateHistogram(fmt.Sprintf(`http_request_duration_seconds{path="%s"}`, r.URL.Path)).UpdateDuration(start)
		}()

		// Increment hit counter
		metrics.GetOrCreateCounter(fmt.Sprintf(`http_requests_total{path="%s"}`, r.URL.Path)).Inc()

		next(w, r)
	}
}
