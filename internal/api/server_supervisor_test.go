package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"pet-study/internal/config"
	"pet-study/internal/health"
	"pet-study/internal/metrics"
	"pet-study/internal/queue"
	"pet-study/internal/service"
	"pet-study/internal/store/jobrepo"
	"pet-study/internal/store/userrepo"
	"pet-study/internal/stream"
	"pet-study/internal/workerpool"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRunShutdownUsesOneGlobalBudgetAndReportsForcedOutcomes(t *testing.T) {
	var logs bytes.Buffer
	restoreLogger := replaceDefaultLogger(t, &logs)
	defer restoreLogger()

	q := queue.New(1)
	hub := stream.NewHub(16)
	pool := newSupervisorTestPool(q, hub)
	grpcRuntime := newFakeSupervisorGRPC()
	grpcRuntime.blockShutdown = true
	readiness := health.NewReadiness()

	handlerStarted := make(chan struct{})
	var handlerStartedOnce sync.Once
	mux := http.NewServeMux()
	mux.HandleFunc("/block", func(w http.ResponseWriter, r *http.Request) {
		handlerStartedOnce.Do(func() { close(handlerStarted) })
		<-r.Context().Done()
	})

	cfg := supervisorTestConfig("127.0.0.1:0", 0)
	cfg.HTTP.ShutdownTimeout = 120 * time.Millisecond

	server := NewAPIServer(cfg, mux, readiness, pool, q, grpcRuntime, hub)

	addrCh := make(chan string, 1)
	server.listen = func(network, addr string) (net.Listener, error) {
		ln, err := net.Listen(network, addr)
		if err == nil {
			addrCh <- ln.Addr().String()
		}
		return ln, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(ctx)
	}()

	addr := waitSupervisorAddr(t, addrCh)
	waitSupervisorReady(t, readiness)

	clientErrCh := make(chan error, 1)
	go func() {
		resp, err := http.Get("http://" + addr + "/block")
		if resp != nil {
			_ = resp.Body.Close()
		}
		clientErrCh <- err
	}()

	waitClosed(t, handlerStarted, "blocking handler start")

	startedShutdown := time.Now()
	cancel()
	err := waitSupervisorRun(t, errCh)
	elapsed := time.Since(startedShutdown)
	_ = waitSupervisorRun(t, clientErrCh)

	if err == nil {
		t.Fatal("Run() error = nil, want forced shutdown outcomes")
	}
	assertComponentShutdownError(t, err, "http", shutdownOutcomeForced)
	assertComponentShutdownError(t, err, "grpc", shutdownOutcomeForced)

	if !errors.Is(grpcRuntime.shutdownCtxErrAtStart(), context.DeadlineExceeded) {
		t.Fatalf("grpc shutdown ctx err at start=%v want DeadlineExceeded", grpcRuntime.shutdownCtxErrAtStart())
	}
	if elapsed > 350*time.Millisecond {
		t.Fatalf("shutdown elapsed=%s, want one global budget around %s", elapsed, cfg.HTTP.ShutdownTimeout)
	}
	summary := findLogRecordByEvent(t, logs.String(), "shutdown.completed")
	if summary[logFieldTrigger] != "context" {
		t.Fatalf("shutdown trigger=%v want context", summary[logFieldTrigger])
	}
	if summary[logFieldOutcome] != string(shutdownOutcomeForced) {
		t.Fatalf("shutdown outcome=%v want forced", summary[logFieldOutcome])
	}
	if _, ok := summary[config.LogFieldDurationMS]; !ok {
		t.Fatalf("shutdown summary missing duration_ms: %#v", summary)
	}
	if summary["queue_depth"] == nil || summary["sse_subscribers"] == nil || summary["repaired_active_jobs"] == nil {
		t.Fatalf("shutdown summary missing aggregate fields: %#v", summary)
	}
	if _, ok := summary[config.LogFieldError]; ok {
		t.Fatalf("shutdown summary must not include raw err: %#v", summary)
	}
	assertSupervisorCleanup(t, q, pool, hub, grpcRuntime)
}

func TestRunReturnsGRPCFatalErrorAndStopsOwnedComponents(t *testing.T) {
	q := queue.New(1)
	hub := stream.NewHub(16)
	pool := newSupervisorTestPool(q, hub)
	grpcRuntime := newFakeSupervisorGRPC()
	readiness := health.NewReadiness()

	server := NewAPIServer(
		supervisorTestConfig("127.0.0.1:0", 1),
		http.NewServeMux(),
		readiness,
		pool,
		q,
		grpcRuntime,
		hub,
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Run(context.Background())
	}()

	waitSupervisorReady(t, readiness)

	serveErr := errors.New("grpc serve failed")
	grpcRuntime.fail(serveErr)

	err := waitSupervisorRun(t, errCh)
	if !errors.Is(err, serveErr) {
		t.Fatalf("Run() error=%v want grpc serve failure", err)
	}
	assertSupervisorCleanup(t, q, pool, hub, grpcRuntime)
}

func TestRunHTTPFailureUsesUnifiedCleanup(t *testing.T) {
	q := queue.New(1)
	hub := stream.NewHub(16)
	pool := newSupervisorTestPool(q, hub)
	grpcRuntime := newFakeSupervisorGRPC()
	readiness := health.NewReadiness()
	serveErr := errors.New("http accept failed")

	server := NewAPIServer(
		supervisorTestConfig("127.0.0.1:0", 1),
		http.NewServeMux(),
		readiness,
		pool,
		q,
		grpcRuntime,
		hub,
	)
	server.listen = func(network, addr string) (net.Listener, error) {
		return &failingListener{err: serveErr}, nil
	}

	err := server.Run(context.Background())
	if !errors.Is(err, serveErr) {
		t.Fatalf("Run() error=%v want http serve failure", err)
	}
	assertSupervisorCleanup(t, q, pool, hub, grpcRuntime)
}

func newSupervisorTestPool(q *queue.Queue, hub *stream.Hub) *workerpool.WorkerPool {
	jobSvc := service.NewJobService(jobrepo.NewMemoryJobRepository())
	userSvc := service.NewUserService(userrepo.NewMemoryUserRepository())
	return workerpool.NewWorkerPool(q, jobSvc, userSvc, metrics.DefaultHTTP(), hub)
}

func supervisorTestConfig(addr string, workers int) config.Config {
	return config.Config{
		HTTP: config.HTTPConfig{
			Addr:            addr,
			ShutdownTimeout: time.Second,
		},
		Pool: config.WorkerPoolConfig{
			Workers: workers,
		},
	}
}

func waitSupervisorReady(t *testing.T, readiness *health.Readiness) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for !readiness.IsReady() {
		if time.Now().After(deadline) {
			t.Fatal("timeout waiting for readiness=true")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitSupervisorRun(t *testing.T, errCh <-chan error) error {
	t.Helper()

	select {
	case err := <-errCh:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Run")
		return nil
	}
}

func waitSupervisorAddr(t *testing.T, addrCh <-chan string) string {
	t.Helper()

	select {
	case addr := <-addrCh:
		return addr
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for listener address")
		return ""
	}
}

func waitClosed(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for %s", name)
	}
}

func replaceDefaultLogger(t *testing.T, buf *bytes.Buffer) func() {
	t.Helper()

	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))
	return func() {
		slog.SetDefault(prev)
	}
}

func findLogRecordByEvent(t *testing.T, logText string, event string) map[string]any {
	t.Helper()

	for _, line := range strings.Split(strings.TrimSpace(logText), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("unmarshal log line: %v line=%q", err, line)
		}
		if record[logFieldEvent] == event {
			return record
		}
	}
	t.Fatalf("log event %q not found in logs:\n%s", event, logText)
	return nil
}

func assertSupervisorCleanup(
	t *testing.T,
	q *queue.Queue,
	pool *workerpool.WorkerPool,
	hub *stream.Hub,
	grpcRuntime *fakeSupervisorGRPC,
) {
	t.Helper()

	if !errors.Is(q.Enqueue(context.Background(), queue.WorkItem{}), queue.ErrQueueStopped) {
		t.Fatal("queue still accepts work after supervisor cleanup")
	}
	if pool.IsRunning() {
		t.Fatal("worker pool is still running after supervisor cleanup")
	}
	if err := hub.Ready(context.Background()); !errors.Is(err, stream.ErrHubClosed) {
		t.Fatalf("event hub Ready error=%v want ErrHubClosed", err)
	}
	if got := grpcRuntime.shutdownCalls(); got != 1 {
		t.Fatalf("grpc Shutdown calls=%d want=1", got)
	}
}

func assertComponentShutdownError(t *testing.T, err error, component string, outcome shutdownOutcome) {
	t.Helper()

	if !hasComponentShutdownError(err, component, outcome) {
		t.Fatalf("error=%v missing %s %s shutdown outcome", err, component, outcome)
	}
}

func hasComponentShutdownError(err error, component string, outcome shutdownOutcome) bool {
	if err == nil {
		return false
	}

	var componentErr *componentShutdownError
	if errors.As(err, &componentErr) && componentErr.Component == component && componentErr.Outcome == outcome {
		return true
	}

	type multiUnwrapper interface {
		Unwrap() []error
	}
	if unwrapped, ok := err.(multiUnwrapper); ok {
		for _, child := range unwrapped.Unwrap() {
			if hasComponentShutdownError(child, component, outcome) {
				return true
			}
		}
		return false
	}

	type singleUnwrapper interface {
		Unwrap() error
	}
	if unwrapped, ok := err.(singleUnwrapper); ok {
		return hasComponentShutdownError(unwrapped.Unwrap(), component, outcome)
	}

	return false
}

type fakeSupervisorGRPC struct {
	done chan struct{}
	once sync.Once

	mu                 sync.Mutex
	err                error
	starts             int
	shutdowns          int
	blockShutdown      bool
	shutdownCtxErrSeen error
}

func newFakeSupervisorGRPC() *fakeSupervisorGRPC {
	return &fakeSupervisorGRPC{
		done: make(chan struct{}),
	}
}

func (g *fakeSupervisorGRPC) Start(context.CancelFunc) error {
	g.mu.Lock()
	g.starts++
	g.mu.Unlock()
	return nil
}

func (g *fakeSupervisorGRPC) Shutdown(ctx context.Context) error {
	g.mu.Lock()
	g.shutdowns++
	blockShutdown := g.blockShutdown
	g.shutdownCtxErrSeen = ctx.Err()
	g.mu.Unlock()
	g.once.Do(func() { close(g.done) })

	if blockShutdown {
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}

func (g *fakeSupervisorGRPC) Done() <-chan struct{} {
	return g.done
}

func (g *fakeSupervisorGRPC) Err() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.err
}

func (g *fakeSupervisorGRPC) fail(err error) {
	g.mu.Lock()
	g.err = err
	g.mu.Unlock()
	g.once.Do(func() { close(g.done) })
}

func (g *fakeSupervisorGRPC) shutdownCalls() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.shutdowns
}

func (g *fakeSupervisorGRPC) shutdownCtxErrAtStart() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.shutdownCtxErrSeen
}

type failingListener struct {
	err       error
	closeOnce sync.Once
	closed    chan struct{}
}

func (l *failingListener) Accept() (net.Conn, error) {
	return nil, l.err
}

func (l *failingListener) Close() error {
	l.closeOnce.Do(func() {
		if l.closed != nil {
			close(l.closed)
		}
	})
	return nil
}

func (l *failingListener) Addr() net.Addr {
	return fakeSupervisorAddr("127.0.0.1:0")
}

type fakeSupervisorAddr string

func (a fakeSupervisorAddr) Network() string { return "tcp" }
func (a fakeSupervisorAddr) String() string  { return string(a) }
