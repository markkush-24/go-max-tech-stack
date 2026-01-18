package middleware

import (
	"net/http"
	"pet-study/internal/metrics"
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

			// если какой-то внешний middleware уже обернул — переиспользуем
			sr, ok := w.(*statusRecorder)
			if !ok {
				sr = newStatusRecorder(w)
				w = sr
			}

			next.ServeHTTP(w, r)

			d := time.Since(start)
			m.Observe(r.Method, r.Pattern, sr.Status(), d)
		})
	}
}
