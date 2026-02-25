package outbound_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"pet-study/internal/outbound"
	"pet-study/internal/outbound/profile"
	"pet-study/internal/requestid"
)

func TestProfileClient_FetchProfile_Success_PropagatesRequestID(t *testing.T) {
	const (
		rid    = "rid-123"
		userID = int64(42)
	)

	var seenRID atomic.Value
	var seenAccept atomic.Value

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method=%s want=%s", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/profiles/42" {
			t.Fatalf("path=%q want=%q", r.URL.Path, "/profiles/42")
		}

		seenRID.Store(r.Header.Get(requestid.HeaderName))
		seenAccept.Store(r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"user_id":42,"bio":"hello","city":"NY"}`))
	}))
	t.Cleanup(upstream.Close)

	ci := outbound.NewClientImpl(mustParseURL(t, upstream.URL), upstream.Client())

	p, err := ci.FetchProfile(context.Background(), userID, rid)
	if err != nil {
		t.Fatalf("FetchProfile err=%v", err)
	}
	if p.UserID != userID || p.Bio != "hello" || p.City != "NY" {
		t.Fatalf("profile=%+v want user_id=%d bio=%q city=%q", p, userID, "hello", "NY")
	}

	if got := seenRID.Load(); got != rid {
		t.Fatalf("X-Request-Id=%v want=%v", got, rid)
	}
	if got := seenAccept.Load(); got != "application/json" {
		t.Fatalf("Accept=%v want=%v", got, "application/json")
	}
}

func TestRetryingProfileClient_RetriesOn5xxUntilSuccess(t *testing.T) {
	const (
		rid    = "rid-5xx"
		userID = int64(1)
	)

	var calls atomic.Int64

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := calls.Add(1)

		if r.Header.Get(requestid.HeaderName) != rid {
			t.Fatalf("call %d: X-Request-Id=%q want=%q", c, r.Header.Get(requestid.HeaderName), rid)
		}

		if c <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"user_id":1,"bio":"ok","city":"LA"}`))
	}))
	t.Cleanup(upstream.Close)

	raw := outbound.NewClientImpl(mustParseURL(t, upstream.URL), upstream.Client())
	retrying := outbound.NewRetryingProfileClient(3, 0, 0, raw)

	p, err := retrying.FetchProfile(context.Background(), userID, rid)
	if err != nil {
		t.Fatalf("FetchProfile err=%v", err)
	}
	if p.UserID != 1 || p.Bio != "ok" || p.City != "LA" {
		t.Fatalf("profile=%+v want user_id=1 bio=%q city=%q", p, "ok", "LA")
	}

	if got := calls.Load(); got != 3 {
		t.Fatalf("upstream calls=%d want=%d", got, 3)
	}
}

func TestRetryingProfileClient_DoesNotRetryOn4xx(t *testing.T) {
	const (
		rid    = "rid-4xx"
		userID = int64(1)
	)

	var calls atomic.Int64

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("nope"))
	}))
	t.Cleanup(upstream.Close)

	raw := outbound.NewClientImpl(mustParseURL(t, upstream.URL), upstream.Client())
	retrying := outbound.NewRetryingProfileClient(5, 0, 0, raw)

	_, err := retrying.FetchProfile(context.Background(), userID, rid)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, profile.ErrUpstream4xx) {
		t.Fatalf("err=%v want errors.Is(..., ErrUpstream4xx)=true", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("upstream calls=%d want=%d", got, 1)
	}
}

func TestRetryingProfileClient_RetriesOnTimeout(t *testing.T) {
	const (
		rid    = "rid-timeout"
		userID = int64(7)
	)

	var calls atomic.Int64

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)

		// Чтобы не держать горутины после таймаута клиента:
		select {
		case <-time.After(200 * time.Millisecond):
			// если клиент вдруг дождался — вернём валидный JSON
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"user_id":7,"bio":"slow","city":"SF"}`))
		case <-r.Context().Done():
			return
		}
	}))
	t.Cleanup(upstream.Close)

	// Клиент с ResponseHeaderTimeout, чтобы получить timeout из Transport (не из ctx).
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.ResponseHeaderTimeout = 50 * time.Millisecond
	t.Cleanup(tr.CloseIdleConnections)

	httpClient := &http.Client{Transport: tr}

	raw := outbound.NewClientImpl(mustParseURL(t, upstream.URL), httpClient)
	retrying := outbound.NewRetryingProfileClient(3, 0, 0, raw)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err := retrying.FetchProfile(ctx, userID, rid)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, profile.ErrTimeout) {
		t.Fatalf("err=%v want errors.Is(..., ErrTimeout)=true", err)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("upstream calls=%d want=%d", got, 3)
	}
}

func TestRetryingProfileClient_DoesNotRetryOnParseError(t *testing.T) {
	const (
		rid    = "rid-parse"
		userID = int64(2)
	)

	var calls atomic.Int64

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{broken-json`))
	}))
	t.Cleanup(upstream.Close)

	raw := outbound.NewClientImpl(mustParseURL(t, upstream.URL), upstream.Client())
	retrying := outbound.NewRetryingProfileClient(3, 0, 0, raw)

	_, err := retrying.FetchProfile(context.Background(), userID, rid)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, profile.ErrParse) {
		t.Fatalf("err=%v want errors.Is(..., ErrParse)=true", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("upstream calls=%d want=%d", got, 1)
	}
}

func TestProfileClient_ClassifiesNetworkError(t *testing.T) {
	const (
		rid    = "rid-net"
		userID = int64(1)
	)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// не важно
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"user_id":1,"bio":"x","city":"y"}`))
	}))
	// Закрываем сразу -> dial должен упасть
	upstream.Close()

	ci := outbound.NewClientImpl(mustParseURL(t, upstream.URL), upstream.Client())
	_, err := ci.FetchProfile(context.Background(), userID, rid)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, profile.ErrNetwork) {
		t.Fatalf("err=%v want errors.Is(..., ErrNetwork)=true", err)
	}
}

func mustParseURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatalf("parse url %q: %v", s, err)
	}
	return u
}
