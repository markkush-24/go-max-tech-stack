package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pet-study/internal/config"
	"pet-study/internal/entity"
	"pet-study/internal/requestid"
	"pet-study/internal/routes"
	"pet-study/internal/security"
	"pet-study/internal/service"
	"pet-study/internal/store/jobrepo"
	"pet-study/internal/stream"
)

func TestJobEventsLogsConnectionLifecycleWithoutPayload(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil)).
		With(config.LogFieldComponent, "sse")

	jobSvc := service.NewJobService(jobrepo.NewMemoryJobRepository())
	job := entity.Job{Status: entity.JobRunning, OwnerUserID: 7}
	if err := jobSvc.Save(context.Background(), &job); err != nil {
		t.Fatalf("save job: %v", err)
	}

	hub := stream.NewHub(16)
	handler := routes.NewJobHandlerWithLogger(jobSvc, hub, time.Hour, 0, nil, logger)

	ctx, cancel := context.WithCancel(context.Background())
	ctx = requestid.WithRequestID(ctx, "rid-sse")
	ctx = security.WithPrincipal(ctx, security.Principal{UserID: 7, Role: security.RoleUser})
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/jobs/%d/events", job.ID), nil).
		WithContext(ctx)
	rec := httptest.NewRecorder()

	errCh := make(chan error, 1)
	go func() {
		errCh <- handler.Events(rec, req, int(job.ID))
	}()

	waitForSSESubscriber(t, hub)
	hub.Publish(job.ID, stream.Event{
		Type:  "queued",
		JobID: job.ID,
		Data:  map[string]string{"body": "secret-body"},
	})
	cancel()

	if err := waitSSEHandler(t, errCh); err != nil {
		t.Fatalf("Events() error = %v", err)
	}

	logText := logs.String()
	if strings.Contains(logText, "secret-body") {
		t.Fatalf("logs=%q must not include SSE payload", logText)
	}

	lines := strings.Split(strings.TrimSpace(logText), "\n")
	if len(lines) != 2 {
		t.Fatalf("log lines=%q want open and close only", logText)
	}

	opened := decodeAPILogRecord(t, lines[0])
	closed := decodeAPILogRecord(t, lines[1])

	if opened["event"] != "sse.connection.opened" {
		t.Fatalf("open event=%v want sse.connection.opened", opened["event"])
	}
	if closed["event"] != "sse.connection.closed" {
		t.Fatalf("close event=%v want sse.connection.closed", closed["event"])
	}
	if opened[config.LogFieldRequestID] != "rid-sse" || closed[config.LogFieldRequestID] != "rid-sse" {
		t.Fatalf("request_id open=%v close=%v want rid-sse", opened[config.LogFieldRequestID], closed[config.LogFieldRequestID])
	}
	if opened["job_id"] != float64(job.ID) || closed["job_id"] != float64(job.ID) {
		t.Fatalf("job_id open=%v close=%v want %d", opened["job_id"], closed["job_id"], job.ID)
	}
	if closed["reason"] != "client_closed" {
		t.Fatalf("close reason=%v want client_closed", closed["reason"])
	}
	if _, ok := closed[config.LogFieldDurationMS]; !ok {
		t.Fatalf("close record missing duration_ms: %#v", closed)
	}
}

func waitForSSESubscriber(t *testing.T, hub *stream.Hub) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for hub.Subscribers() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("timeout waiting for SSE subscriber")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitSSEHandler(t *testing.T, errCh <-chan error) error {
	t.Helper()

	select {
	case err := <-errCh:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for SSE handler")
		return nil
	}
}

func decodeAPILogRecord(t *testing.T, line string) map[string]any {
	t.Helper()

	var record map[string]any
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		t.Fatalf("unmarshal log line: %v line=%q", err, line)
	}
	return record
}
