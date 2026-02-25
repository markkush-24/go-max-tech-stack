package middleware_test

import (
	"mime"
	"net/http"
	"pet-study/internal/requestid"
	"pet-study/internal/testkit"
	"strconv"
	"testing"
)

func TestRateLimiterRetryAfter(t *testing.T) {
	server, _ := testkit.NewServer(t, testkit.WithRateLimit(1, 1))

	resp, err := server.Client().Get(server.URL + "/api/v1/users")
	if err != nil {
		t.Fatalf("get #1: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get #1 status=%d want=%d", resp.StatusCode, http.StatusOK)
	}

	resp2, err2 := server.Client().Get(server.URL + "/api/v1/users")
	if err2 != nil {
		t.Fatalf("get #2: %v", err2)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("get #2 status=%d want=%d", resp2.StatusCode, http.StatusTooManyRequests)
	}

	rawCT := resp2.Header.Get("Content-Type")
	mt, _, err := mime.ParseMediaType(rawCT)
	if err != nil {
		t.Fatalf("bad Content-Type %q: %v", rawCT, err)
	}
	if mt != "application/problem+json" {
		t.Fatalf("Content-Type=%q want=%q", mt, "application/problem+json")
	}

	rid := resp2.Header.Get(requestid.HeaderName)
	if rid == "" {
		t.Fatalf("%s empty", requestid.HeaderName)
	}

	retryAfter2 := resp2.Header.Get("Retry-After")
	if retryAfter2 == "" {
		t.Fatalf("%s empty", "Retry-After")
	}
	n, err := strconv.Atoi(retryAfter2)
	if err != nil {
		t.Fatalf("%v convert error", err)
	}
	if n < 1 {
		t.Fatalf("Retry-After must be >= 1 , current %v", n)
	}
}
