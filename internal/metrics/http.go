package metrics

import (
	"expvar"
	"strconv"
	"sync"
	"time"
)

// HTTPMetrics — минимальные HTTP-метрики.
// Экспортируются через expvar (обычно /debug/vars).
type HTTPMetrics struct {
	inFlight      *expvar.Int
	requestsTotal *expvar.Map
	errorsTotal   *expvar.Map
	latencySumNS  *expvar.Map
	latencyCount  *expvar.Map
}

var (
	defaultHTTP *HTTPMetrics
	once        sync.Once
)

func DefaultHTTP() *HTTPMetrics {
	once.Do(func() {
		defaultHTTP = &HTTPMetrics{
			inFlight:      expvar.NewInt("http_in_flight"),
			requestsTotal: expvar.NewMap("http_requests_total"),
			errorsTotal:   expvar.NewMap("http_errors_total"),
			latencySumNS:  expvar.NewMap("http_latency_ns_sum"),
			latencyCount:  expvar.NewMap("http_latency_ns_count"),
		}
	})
	return defaultHTTP
}

func (m *HTTPMetrics) IncInFlight() { m.inFlight.Add(1) }
func (m *HTTPMetrics) DecInFlight() { m.inFlight.Add(-1) }

func (m *HTTPMetrics) Observe(method, pattern string, status int, d time.Duration) {
	if pattern == "" {
		pattern = "<unmatched>"
	}

	// Ключи:
	// - requests_total: method|pattern|status  (чтобы видеть распределение статусов)
	// - errors_total:   method|pattern        (минимум)
	// - latency_*:      method|pattern        (счётчик/сумма для среднего)
	reqKey := method + "|" + pattern + "|" + strconv.Itoa(status)
	baseKey := method + "|" + pattern

	m.requestsTotal.Add(reqKey, 1)
	m.latencySumNS.Add(baseKey, d.Nanoseconds())
	m.latencyCount.Add(baseKey, 1)

	if status >= 500 {
		m.errorsTotal.Add(baseKey, 1)
	}
}
