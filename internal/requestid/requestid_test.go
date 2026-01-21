package requestid_test

import (
	"net/http"
	"net/http/httptest"
	"pet-study/internal/requestid"
	"testing"
)

func TestRequestIDMiddleware_PreservesIncomingHeaderAndPutsInContext(t *testing.T) {
	want := "abc-123"

	var gotCtx string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rid, ok := requestid.RequestID(r.Context()); ok {
			gotCtx = rid
		}
		w.WriteHeader(http.StatusNoContent)
	})

	h := requestid.RequestIDMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set(requestid.HeaderName, want)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=204", rec.Code)
	}
	if got := rec.Header().Get(requestid.HeaderName); got != want {
		t.Fatalf("%s=%q want=%q", requestid.HeaderName, got, want)
	}
	if gotCtx != want {
		t.Fatalf("ctx request id=%q want=%q", gotCtx, want)
	}
}

func TestRequestIDMiddleware_GeneratesWhenMissing(t *testing.T) {
	var gotCtx string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rid, ok := requestid.RequestID(r.Context()); ok {
			gotCtx = rid
		}
		w.WriteHeader(http.StatusNoContent)
	})
	h := requestid.RequestIDMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	rid := rec.Header().Get(requestid.HeaderName)
	if rid == "" {
		t.Fatalf("%s empty", requestid.HeaderName)
	}
	if gotCtx == "" {
		t.Fatalf("ctx request id empty")
	}
	if rid != gotCtx {
		t.Fatalf("header rid=%q ctx rid=%q must match", rid, gotCtx)
	}
}

func TestRequestIDMiddleware_IgnoresInvalidIncomingHeader(t *testing.T) {
	// invalid: control char
	bad := "bad\nvalue"

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	h := requestid.RequestIDMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set(requestid.HeaderName, bad)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	rid := rec.Header().Get(requestid.HeaderName)
	if rid == "" {
		t.Fatalf("%s empty", requestid.HeaderName)
	}
	if rid == bad {
		t.Fatalf("invalid incoming id must not be echoed")
	}
}
