package api

import (
	"context"
	"errors"
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
	"sync"
	"testing"
	"time"
)

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

type fakeSupervisorGRPC struct {
	done chan struct{}
	once sync.Once

	mu        sync.Mutex
	err       error
	starts    int
	shutdowns int
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

func (g *fakeSupervisorGRPC) Shutdown(context.Context) error {
	g.mu.Lock()
	g.shutdowns++
	g.mu.Unlock()
	g.once.Do(func() { close(g.done) })
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
