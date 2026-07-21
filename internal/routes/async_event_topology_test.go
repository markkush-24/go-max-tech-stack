package routes_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"pet-study/internal/entity"
	"pet-study/internal/testkit"
	"pet-study/internal/workerpool"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAsyncCreatePublishesMonotonicEventsToJobSSE(t *testing.T) {
	srv, app := testkit.NewServer(t, testkit.WithPrincipalAdmin(1))

	pool := workerpool.NewWorkerPool(app.Q, app.JobSvc, app.UserSvc, app.M, app.EventHub)
	if err := pool.Start(context.Background(), 1); err != nil {
		t.Fatalf("start workerpool: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := pool.Stop(ctx); err != nil {
			t.Errorf("stop workerpool: %v", err)
		}
	})

	createResp := postAsyncUser(t, srv.Client(), srv.URL+"/api/v1/users?async=1")
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(createResp.Body)
		t.Fatalf("POST async status=%d want=%d body=%s", createResp.StatusCode, http.StatusAccepted, body)
	}

	var job entity.Job
	if err := json.NewDecoder(createResp.Body).Decode(&job); err != nil {
		t.Fatalf("decode async job: %v", err)
	}
	if job.ID == 0 {
		t.Fatal("async response returned zero job id")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/v1/jobs/"+strconv.FormatInt(job.ID, 10)+"/events", nil)
	if err != nil {
		t.Fatalf("new events request: %v", err)
	}

	eventsResp, err := srv.Client().Do(eventsReq)
	if err != nil {
		t.Fatalf("GET job events: %v", err)
	}
	defer eventsResp.Body.Close()

	if eventsResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(eventsResp.Body)
		t.Fatalf("GET events status=%d want=%d body=%s", eventsResp.StatusCode, http.StatusOK, body)
	}

	got := readSSEEventTypes(t, eventsResp.Body, 3)
	want := []string{string(entity.JobQueued), string(entity.JobRunning), string(entity.JobSucceeded)}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("events=%v want=%v", got, want)
	}
	assertMonotonicJobEvents(t, got)
}

func postAsyncUser(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(
		http.MethodPost,
		url,
		bytes.NewBufferString(`{"name":"Async Bob","email":"async-bob@example.com","age":21}`),
	)
	if err != nil {
		t.Fatalf("new async request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST async user: %v", err)
	}
	return resp
}

func readSSEEventTypes(t *testing.T, body io.Reader, count int) []string {
	t.Helper()

	reader := bufio.NewReader(body)
	events := make([]string, 0, count)
	for len(events) < count {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read SSE event: %v", err)
		}
		if eventType, ok := strings.CutPrefix(line, "event: "); ok {
			events = append(events, strings.TrimSpace(eventType))
		}
	}
	return events
}

func assertMonotonicJobEvents(t *testing.T, events []string) {
	t.Helper()

	ranks := map[string]int{
		string(entity.JobQueued):    0,
		string(entity.JobRunning):   1,
		string(entity.JobSucceeded): 2,
		string(entity.JobFailed):    2,
	}

	last := -1
	for _, eventType := range events {
		rank, ok := ranks[eventType]
		if !ok {
			t.Fatalf("unknown job event %q", eventType)
		}
		if rank < last {
			t.Fatalf("events are not monotonic: %v", events)
		}
		last = rank
	}
}
