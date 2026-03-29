package middleware

import (
	"net/http"
	"pet-study/internal/metrics"
	"strings"
	"time"
)

// Metrics — middleware для метрик.
// Важно: pattern читаем ПОСЛЕ next.ServeHTTP (ServeMux проставит r.Pattern).
func Metrics(m *metrics.HTTPMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			m.IncInFlight()
			defer m.DecInFlight()

			start := time.Now()

			sr, ok := w.(*statusRecorder)
			if !ok {
				sr = newStatusRecorder(w)
				w = sr
			}

			next.ServeHTTP(w, r)

			// pattern читаем ПОСЛЕ next.ServeHTTP
			pattern := r.Pattern
			if pattern == "" {
				pattern = "<unmatched>"
			}

			// Не считаем debug endpoints / events (эвристика)
			if strings.HasPrefix(r.URL.Path, "/debug/") || pattern == "/api/v1/jobs/{id}/events" {
				return
			}
			// Если pattern вида "GET /path" — режем до "/path"
			if i := strings.IndexByte(pattern, ' '); i != -1 {
				pattern = pattern[i+1:]
			}

			d := time.Since(start)
			m.Observe(r.Method, pattern, sr.Status(), d)
		})
	}
}
