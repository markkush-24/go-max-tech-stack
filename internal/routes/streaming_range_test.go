package routes_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"pet-study/internal/entity"
	"pet-study/internal/stream"
	"pet-study/internal/testkit"
	"strings"
	"testing"
	"time"
)

type exportedUser struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

type ssePayload struct {
	Step string `json:"step"`
}

func TestJobEventsStream_OwnerReceivesEvent(t *testing.T) {
	srv, app := testkit.NewServer(t, testkit.WithPrincipalUser(1))

	job := entity.Job{Status: entity.JobQueued, OwnerUserID: 1}
	if err := app.JobSvc.Save(context.Background(), &job); err != nil {
		t.Fatalf("save job: %v", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/api/v1/jobs/1/events", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := srv.Client().Do(req)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	time.Sleep(50 * time.Millisecond)
	app.EventHub.Publish(job.ID, stream.Event{Type: "queued", JobID: job.ID, Data: ssePayload{Step: "queued"}})

	var resp *http.Response
	select {
	case err := <-errCh:
		t.Fatalf("GET /events: %v", err)
	case resp = <-respCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for SSE response")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d want=200 body=%s", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type=%q want=%q", ct, "text/event-stream")
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("Cache-Control=%q want=%q", cc, "no-cache")
	}

	reader := bufio.NewReader(resp.Body)
	var lines []string
	deadline := time.After(2 * time.Second)
	for len(lines) < 3 {
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for SSE payload; got=%q", strings.Join(lines, ""))
		default:
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read SSE line: %v", err)
		}
		lines = append(lines, line)
		if line == "\n" {
			break
		}
	}

	joined := strings.Join(lines, "")
	if !strings.Contains(joined, "event: queued\n") {
		t.Fatalf("SSE body missing event line: %q", joined)
	}
	if !strings.Contains(joined, `data: {"step":"queued"}`) {
		t.Fatalf("SSE body missing data line: %q", joined)
	}
}

func TestJobEventsStream_AdminAllowed(t *testing.T) {
	srv, app := testkit.NewServer(t, testkit.WithPrincipalAdmin(999))

	job := entity.Job{Status: entity.JobQueued, OwnerUserID: 1}
	if err := app.JobSvc.Save(context.Background(), &job); err != nil {
		t.Fatalf("save job: %v", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/api/v1/jobs/1/events", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 1)
	go func() {
		resp, err := srv.Client().Do(req)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	time.Sleep(50 * time.Millisecond)
	app.EventHub.Publish(job.ID, stream.Event{Type: "queued", JobID: job.ID, Data: ssePayload{Step: "queued"}})

	select {
	case err := <-errCh:
		t.Fatalf("GET /events as admin: %v", err)
	case resp := <-respCh:
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status=%d want=200 body=%s", resp.StatusCode, body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for admin SSE response")
	}
}

func TestUsersExport_FullDownloadReturnsJSON(t *testing.T) {
	srv, app := testkit.NewServer(t, testkit.WithPrincipalUser(1))

	user, err := app.UserSvc.CreateUser(context.Background(), testkit.CreateUser("Bob", "bob@example.com", 21))
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	resp, err := srv.Client().Get(srv.URL + "/api/v1/users/1/export")
	if err != nil {
		t.Fatalf("GET /export: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d want=200 body=%s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Disposition"); got != `attachment; filename="user-1-export.json"` {
		t.Fatalf("Content-Disposition=%q", got)
	}

	var out exportedUser
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode export body: %v", err)
	}
	if out.ID != int64(user.ID) || out.Name != user.Name || out.Email != user.Email {
		t.Fatalf("unexpected export body: %+v", out)
	}
}

func TestUsersExport_RangeRequestReturnsPartialContent(t *testing.T) {
	srv, app := testkit.NewServer(t, testkit.WithPrincipalUser(1))

	_, err := app.UserSvc.CreateUser(context.Background(), testkit.CreateUser("Bob", "bob@example.com", 21))
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/users/1/export", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Range", "bytes=0-9")

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /export with range: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d want=206 body=%s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Range"); got != "bytes 0-9/56" {
		t.Fatalf("Content-Range=%q want=%q", got, "bytes 0-9/56")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read partial body: %v", err)
	}
	if string(body) != `{"id":1,"n` {
		t.Fatalf("partial body=%q want=%q", string(body), `{"id":1,"n`)
	}
}
