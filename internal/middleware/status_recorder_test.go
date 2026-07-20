package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusRecorder_FirstWriteHeaderWins(t *testing.T) {
	rec := httptest.NewRecorder()
	sr := newStatusRecorder(rec)

	sr.WriteHeader(http.StatusCreated)
	sr.WriteHeader(http.StatusInternalServerError)

	if got := sr.Status(); got != http.StatusCreated {
		t.Fatalf("recorded status=%d want=%d", got, http.StatusCreated)
	}
	if got := rec.Code; got != http.StatusCreated {
		t.Fatalf("wire status=%d want=%d", got, http.StatusCreated)
	}
}

func TestStatusRecorder_ImplicitWriteRecordsOK(t *testing.T) {
	rec := httptest.NewRecorder()
	sr := newStatusRecorder(rec)

	n, err := sr.Write([]byte("ok"))
	if err != nil {
		t.Fatalf("Write error=%v", err)
	}

	if n != 2 {
		t.Fatalf("bytes written=%d want=2", n)
	}
	if got := sr.Status(); got != http.StatusOK {
		t.Fatalf("recorded status=%d want=%d", got, http.StatusOK)
	}
	if got := sr.Bytes(); got != 2 {
		t.Fatalf("recorded bytes=%d want=2", got)
	}
	if got := rec.Code; got != http.StatusOK {
		t.Fatalf("wire status=%d want=%d", got, http.StatusOK)
	}
}

func TestStatusRecorder_ExplicitErrorStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	sr := newStatusRecorder(rec)

	sr.WriteHeader(http.StatusTeapot)

	if got := sr.Status(); got != http.StatusTeapot {
		t.Fatalf("recorded status=%d want=%d", got, http.StatusTeapot)
	}
	if got := rec.Code; got != http.StatusTeapot {
		t.Fatalf("wire status=%d want=%d", got, http.StatusTeapot)
	}
}

func TestStatusRecorder_ResponseControllerFlushRecordsImplicitOK(t *testing.T) {
	rec := httptest.NewRecorder()
	sr := newStatusRecorder(rec)

	if err := http.NewResponseController(sr).Flush(); err != nil {
		t.Fatalf("Flush error=%v", err)
	}

	if !rec.Flushed {
		t.Fatal("underlying ResponseWriter was not flushed")
	}
	if got := sr.Status(); got != http.StatusOK {
		t.Fatalf("recorded status=%d want=%d", got, http.StatusOK)
	}
	if got := rec.Code; got != http.StatusOK {
		t.Fatalf("wire status=%d want=%d", got, http.StatusOK)
	}
	if got := sr.Unwrap(); got != rec {
		t.Fatalf("Unwrap()=%T want underlying recorder", got)
	}
}
