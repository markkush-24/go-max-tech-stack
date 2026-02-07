package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"pet-study/internal/httputils"
	"pet-study/internal/middleware"
	"testing"
	"time"
)

func TestBulkheadRejected(t *testing.T) {
	bh := middleware.NewBulkhead(1)

	entered := make(chan struct{})
	release := make(chan struct{})

	// искусственный AppHandler, который держит слот
	slow := func(w http.ResponseWriter, r *http.Request) error {
		close(entered) // сигнал: заняли слот и внутри handler
		<-release      // держим, пока тест не отпустит
		w.WriteHeader(http.StatusOK)
		return nil
	}

	// оборачиваем bulkhead
	wrapped := bh.Bulkhead(slow)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := wrapped(w, r); err != nil {
			mp := httputils.MapError(r, err)
			for k, vals := range mp.Headers {
				for _, v := range vals {
					w.Header().Add(k, v)
				}
			}
			err := httputils.WriteProblem(w, r, mp.Problem)
			if err != nil {
				return
			}
		}
	})

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	// 1-й запрос запускаем в горутине — он зависнет
	done1 := make(chan struct{})
	go func() {
		resp, err := srv.Client().Get(srv.URL)
		if err == nil {
			resp.Body.Close()
		}
		close(done1)
	}()

	select {
	case <-entered:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for first request to enter handler")
	}

	// 2-й запрос должен быть 503
	resp2, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatalf("get #2: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("get #2 status=%d want=%d", resp2.StatusCode, http.StatusServiceUnavailable)
	}

	close(release) // отпускаем 1-й
	<-done1
}
