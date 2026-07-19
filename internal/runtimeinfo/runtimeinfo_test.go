package runtimeinfo

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReadSnapshotIncludesMemoryAndGCSignals(t *testing.T) {
	snapshot := ReadSnapshot(time.Unix(1, 2))

	if snapshot.Timestamp != "1970-01-01T00:00:01.000000002Z" {
		t.Fatalf("timestamp=%q", snapshot.Timestamp)
	}
	if snapshot.Go.Version == "" {
		t.Fatalf("missing go version")
	}
	if snapshot.Go.NumCPU <= 0 {
		t.Fatalf("num_cpu=%d want > 0", snapshot.Go.NumCPU)
	}
	if snapshot.Go.GOMAXPROCS <= 0 {
		t.Fatalf("gomaxprocs=%d want > 0", snapshot.Go.GOMAXPROCS)
	}
	if snapshot.MemStats.NextGCBytes == 0 {
		t.Fatalf("next_gc_bytes=0")
	}

	for _, name := range []string{
		"/gc/gogc:percent",
		"/gc/gomemlimit:bytes",
		"/gc/heap/goal:bytes",
		"/memory/classes/total:bytes",
		"/sched/goroutines:goroutines",
	} {
		if _, ok := snapshot.Metrics[name]; !ok {
			t.Fatalf("missing runtime metric %s", name)
		}
	}
}

func TestHandlerRejectsNonGET(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/debug/runtime", nil)
	rec := httptest.NewRecorder()

	Handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want=%d", rec.Code, http.StatusMethodNotAllowed)
	}
	if rec.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("Allow=%q want=%q", rec.Header().Get("Allow"), http.MethodGet)
	}
}
