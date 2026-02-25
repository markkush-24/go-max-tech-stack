package router_test

import (
	"bytes"
	"encoding/json"
	"mime"
	"net/http"
	"net/http/httptest"
	"pet-study/internal/testkit"
	"testing"
)

type problem struct {
	Type          string         `json:"type"`
	Title         string         `json:"title"`
	Status        int            `json:"status"`
	Detail        string         `json:"detail"`
	Instance      string         `json:"instance"`
	InvalidParams []invalidParam `json:"invalid_params,omitempty"`
}

func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder) problem {
	t.Helper()
	var p problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("unmarshal problem: %v, body=%q", err, rec.Body.String())
	}
	return p
}

func TestRouting_StatusCodes(t *testing.T) {

	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"v1 list", "GET", "/api/v1/users", 200},
		{"v2 list", "GET", "/api/v2/users", 200},
		{"not found", "GET", "/api/v1/nope", 404},
		{"v1 item bad id", "GET", "/api/v1/users/abc", 400},
		{"v1 item extra segment", "GET", "/api/v1/users/1/extra", 404},
		{"v1 405 collection", "PUT", "/api/v1/users", 405},
		{"v2 405 collection", "PUT", "/api/v2/users", 405},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			h, _ := testkit.NewUserRouter(t)
			h.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}

func TestMethodNotAllowed_HasAllowHeader(t *testing.T) {

	tests := []struct {
		name      string
		method    string
		path      string
		wantAllow string
	}{
		{"v1 users", "PUT", "/api/v1/users", "GET, POST"},
		{"v1 item", "POST", "/api/v1/users/1", "GET"},
		{"v2 users", "PUT", "/api/v2/users", "GET, POST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			h, _ := testkit.NewUserRouter(t)
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status=%d want=405 body=%s", rec.Code, rec.Body.String())
			}
			if got := rec.Header().Get("Allow"); got != tt.wantAllow {
				t.Fatalf("Allow=%q want=%q", got, tt.wantAllow)
			}
			if ct := rec.Header().Get("Content-Type"); ct == "" || ct[:len("application/problem+json")] != "application/problem+json" {
				t.Fatalf("Content-Type=%q", ct)
			}
			p := decodeProblem(t, rec)
			if p.Status != 405 {
				t.Fatalf("problem.status=%d want=405", p.Status)
			}
		})
	}
}

func TestDecodeErrors_ReturnProblemJSON(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		body   string
		want   int
		detail string
	}{
		{"v1 empty body", "/api/v1/users", "", 400, "request body must not be empty"},
		{"v1 bad json", "/api/v1/users", "{", 400, "malformed JSON"},
		{"v1 trailing data", "/api/v1/users", `{"name":"a","email":"a@b.c"}{"name":"b","email":"b@c.d"}`, 400, "request body must contain a single JSON value"},
		{"v1 unknown field", "/api/v1/users", `{"name":"a","email":"a@b.c","wat":1}`, 400, `unknown field "wat"`},

		{"v2 unknown field", "/api/v2/users", `{"name":"a","email":"a@b.c"}`, 400, `unknown field "name"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", tt.path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			h, _ := testkit.NewUserRouter(t)
			h.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tt.want, rec.Body.String())
			}
			if got := mustMediaType(t, rec); got != "application/problem+json" {
				t.Fatalf("mediaType=%q", got)
			}
			p := decodeProblem(t, rec)
			if p.Status != tt.want {
				t.Fatalf("problem.status=%d want=%d", p.Status, tt.want)
			}
			if tt.detail != "" && p.Detail != tt.detail {
				t.Fatalf("problem.detail=%q want=%q", p.Detail, tt.detail)
			}
			if p.Instance == "" {
				t.Fatalf("problem.instance empty")
			}
		})
	}
}
func TestPayloadTooLarge_Return413Problem(t *testing.T) {
	// 70KB строка
	big := bytes.Repeat([]byte("a"), 70_000)
	body := append([]byte(`{"name":"`), big...)
	body = append(body, []byte(`","email":"a@b.c","age":1}`)...)

	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h, _ := testkit.NewUserRouter(t)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d want=413 body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct == "" || ct[:len("application/problem+json")] != "application/problem+json" {
		t.Fatalf("Content-Type=%q", ct)
	}
	p := decodeProblem(t, rec)
	if p.Status != 413 {
		t.Fatalf("problem.status=%d want=413", p.Status)
	}
}

type invalidParam struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

func TestValidation_Return422WithInvalidParams(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(`{"name":"","email":"abc","age":-1}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h, _ := testkit.NewUserRouter(t)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d want=422 body=%s", rec.Code, rec.Body.String())
	}

	p := decodeProblem(t, rec)
	if p.Status != 422 {
		t.Fatalf("problem.status=%d want=422", p.Status)
	}
	if len(p.InvalidParams) == 0 {
		t.Fatalf("expected invalid_params, got none")
	}
}
func TestUnsupportedMediaType_Return415Problem(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(`{"name":"a","email":"a@b.c"}`))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	h, _ := testkit.NewUserRouter(t)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status=%d want=415 body=%s", rec.Code, rec.Body.String())
	}
}
func mustMediaType(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	mt, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
	if err != nil {
		t.Fatalf("bad Content-Type: %v", err)
	}
	return mt
}
