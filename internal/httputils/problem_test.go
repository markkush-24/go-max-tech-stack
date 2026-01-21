package httputils_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"pet-study/internal/httputils"
	"pet-study/internal/requestid"
	"testing"
)

func TestWriteProblem_PopulatesRequestID_FromContextAndHeader(t *testing.T) {
	rid := "test-rid-1"

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/users", nil)
	req = req.WithContext(requestid.WithRequestID(req.Context(), rid))

	rec := httptest.NewRecorder()

	p := httputils.Problem{Status: http.StatusBadRequest, Detail: "x"}
	if err := httputils.WriteProblem(rec, req, p); err != nil {
		t.Fatalf("WriteProblem: %v", err)
	}

	if got := rec.Header().Get(requestid.HeaderName); got != rid {
		t.Fatalf("%s=%q want=%q", requestid.HeaderName, got, rid)
	}

	var out struct {
		RequestID string `json:"request_id"`
		Status    int    `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal body: %v body=%q", err, rec.Body.String())
	}
	if out.Status != http.StatusBadRequest {
		t.Fatalf("body.status=%d want=%d", out.Status, http.StatusBadRequest)
	}
	if out.RequestID != rid {
		t.Fatalf("body.request_id=%q want=%q", out.RequestID, rid)
	}
}
