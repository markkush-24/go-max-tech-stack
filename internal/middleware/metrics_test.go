package middleware_test

import (
	"encoding/json"
	"expvar"
	"fmt"
	"net/http"
	"net/http/httptest"
	"pet-study/internal/metrics"
	"pet-study/internal/middleware"
	"strings"
	"testing"
	"time"
)

func expvarMapGetInt(t *testing.T, varName, key string) int64 {
	t.Helper()
	v := expvar.Get(varName)
	if v == nil {
		t.Fatalf("expvar %q not found", varName)
	}
	raw := []byte(v.String())
	m := map[string]float64{}
	if len(raw) != 0 {
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("unmarshal expvar %q: %v raw=%q", varName, err, v.String())
		}
	}
	if x, ok := m[key]; ok {
		return int64(x)
	}
	return 0
}

func expvarIntGet(t *testing.T, varName string) int64 {
	t.Helper()
	v := expvar.Get(varName)
	if v == nil {
		t.Fatalf("expvar %q not found", varName)
	}
	var n int64
	if err := json.Unmarshal([]byte(v.String()), &n); err != nil {
		t.Fatalf("unmarshal expvar int %q: %v raw=%q", varName, err, v.String())
	}
	return n
}

func TestMetrics_RequestsAndLatencyIncrease(t *testing.T) {
	m := metrics.DefaultHTTP()
	pat := fmt.Sprintf("/test-metrics/%d/{id}", time.Now().UnixNano())

	mux := http.NewServeMux()
	mux.HandleFunc("GET "+pat, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	h := middleware.Metrics(m)(mux)

	reqKey := "GET|" + pat + "|200"
	baseKey := "GET|" + pat

	beforeReq := expvarMapGetInt(t, "http_requests_total", reqKey)
	beforeCnt := expvarMapGetInt(t, "http_latency_ns_count", baseKey)
	beforeSum := expvarMapGetInt(t, "http_latency_ns_sum", baseKey)

	path := strings.Replace(pat, "{id}", "1", 1)
	req := httptest.NewRequest(http.MethodGet, "http://example.com"+path[:len(path)-len("{id}")]+"1", nil)
	req.URL.Path = pat[:len(pat)-len("/{id}")] + "/1"

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want=200", rec.Code)
	}

	afterReq := expvarMapGetInt(t, "http_requests_total", reqKey)
	afterCnt := expvarMapGetInt(t, "http_latency_ns_count", baseKey)
	afterSum := expvarMapGetInt(t, "http_latency_ns_sum", baseKey)

	if afterReq != beforeReq+1 {
		t.Fatalf("requests_total[%q]=%d want=%d", reqKey, afterReq, beforeReq+1)
	}
	if afterCnt != beforeCnt+1 {
		t.Fatalf("latency_count[%q]=%d want=%d", baseKey, afterCnt, beforeCnt+1)
	}
	if afterSum <= beforeSum {
		t.Fatalf("latency_sum[%q]=%d want > %d", baseKey, afterSum, beforeSum)
	}

	if got := expvarIntGet(t, "http_in_flight"); got < 0 {
		t.Fatalf("http_in_flight=%d invalid", got)
	}
}

func TestMetrics_ErrorsTotalIncreasesOn5xx(t *testing.T) {
	m := metrics.DefaultHTTP()
	pat := fmt.Sprintf("/test-errors/%d", time.Now().UnixNano())

	mux := http.NewServeMux()
	mux.HandleFunc("GET "+pat, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	h := middleware.Metrics(m)(mux)

	baseKey := "GET|" + pat
	reqKey := baseKey + "|500"

	beforeErr := expvarMapGetInt(t, "http_errors_total", baseKey)
	beforeReq := expvarMapGetInt(t, "http_requests_total", reqKey)

	path := strings.Replace(pat, "{id}", "1", 1)
	req := httptest.NewRequest(http.MethodGet, "http://example.com"+path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d want=500", rec.Code)
	}

	afterErr := expvarMapGetInt(t, "http_errors_total", baseKey)
	afterReq := expvarMapGetInt(t, "http_requests_total", reqKey)

	if afterReq != beforeReq+1 {
		t.Fatalf("requests_total[%q]=%d want=%d", reqKey, afterReq, beforeReq+1)
	}
	if afterErr != beforeErr+1 {
		t.Fatalf("errors_total[%q]=%d want=%d", baseKey, afterErr, beforeErr+1)
	}
}

func TestMetrics_SkipsDebugPaths(t *testing.T) {
	m := metrics.DefaultHTTP()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /debug/vars", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := middleware.Metrics(m)(mux)

	baseKey := "GET|/debug/vars"
	reqKey := baseKey + "|200"

	beforeReq := expvarMapGetInt(t, "http_requests_total", reqKey)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/debug/vars", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want=200", rec.Code)
	}
	afterReq := expvarMapGetInt(t, "http_requests_total", reqKey)
	if afterReq != beforeReq {
		t.Fatalf("debug endpoints must be skipped by metrics: before=%d after=%d", beforeReq, afterReq)
	}
}
