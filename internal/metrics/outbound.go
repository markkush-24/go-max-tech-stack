package metrics

import (
	"expvar"
	"strconv"
	"sync"
	"time"
)

// OutboundMetrics — метрики исходящих HTTP-запросов.
// Экспортируются через expvar (обычно /debug/vars).
//
// Ключи в expvar.Map кодируются строкой через '|':
// - outbound_requests_total: host|route|status_class
// - outbound_latency_ns_sum: host|route
// - outbound_latency_ns_count: host|route
// - outbound_errors_total: kind
type OutboundMetrics struct {
	requestsTotal *expvar.Map
	latencySumNS  *expvar.Map
	latencyCount  *expvar.Map
	errorsTotal   *expvar.Map
}

var (
	defaultOutbound *OutboundMetrics
	outOnce         sync.Once
)

func DefaultOutbound() *OutboundMetrics {
	outOnce.Do(func() {
		defaultOutbound = &OutboundMetrics{
			requestsTotal: expvar.NewMap("outbound_requests_total"),
			latencySumNS:  expvar.NewMap("outbound_latency_ns_sum"),
			latencyCount:  expvar.NewMap("outbound_latency_ns_count"),
			errorsTotal:   expvar.NewMap("outbound_errors_total"),
		}
	})
	return defaultOutbound
}

func (m *OutboundMetrics) Observe(host, route string, status int, d time.Duration) {
	if host == "" {
		host = "<unknown>"
	}
	if route == "" {
		route = "<unknown>"
	}
	statusClass := statusClass(status)

	reqKey := host + "|" + route + "|" + statusClass
	baseKey := host + "|" + route

	m.requestsTotal.Add(reqKey, 1)
	m.latencySumNS.Add(baseKey, d.Nanoseconds())
	m.latencyCount.Add(baseKey, 1)
}

func (m *OutboundMetrics) IncError(kind string) {
	if kind == "" {
		kind = "unknown"
	}
	m.errorsTotal.Add(kind, 1)
}

func statusClass(status int) string {
	if status == 0 {
		return "error"
	}
	if status < 100 || status > 599 {
		return "unknown"
	}
	return strconv.Itoa(status/100) + "xx"
}
