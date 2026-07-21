package grpcserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"pet-study/internal/service"
	"pet-study/internal/store/jobrepo"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestRuntimeNewRuntimeRejectsBindFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	jobSvc := service.NewJobService(jobrepo.NewMemoryJobRepository())
	_, err = NewRuntime(listener.Addr().String(), jobSvc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("NewRuntime succeeded on an already-bound address")
	}
}

func TestRuntimeStartExposesFatalServeError(t *testing.T) {
	serveErr := errors.New("serve failed")
	server := newFakeRuntimeServer()
	server.serveErr = serveErr
	runtime := newTestRuntime(server)

	stopCalled := make(chan struct{})
	if err := runtime.Start(func() { close(stopCalled) }); err != nil {
		t.Fatalf("Start: %v", err)
	}

	select {
	case <-runtime.Done():
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for runtime Done")
	}

	if !errors.Is(runtime.Err(), serveErr) {
		t.Fatalf("Err=%v want %v", runtime.Err(), serveErr)
	}

	select {
	case <-stopCalled:
	case <-time.After(time.Second):
		t.Fatal("fatal Serve error did not notify owner")
	}
}

func TestRuntimeStartRejectsRepeatedStart(t *testing.T) {
	server := newFakeRuntimeServer()
	server.blockServe = true
	runtime := newTestRuntime(server)

	if err := runtime.Start(nil); err != nil {
		t.Fatalf("Start #1: %v", err)
	}
	waitClosed(t, server.serveStarted, "Serve start")

	if err := runtime.Start(nil); !errors.Is(err, ErrGRPCServerAlreadyStarted) {
		t.Fatalf("Start #2 error=%v want ErrGRPCServerAlreadyStarted", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runtime.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	if got := server.serveCalls(); got != 1 {
		t.Fatalf("Serve calls=%d want=1", got)
	}
}

func TestRuntimeShutdownForcedFallbackIsIdempotent(t *testing.T) {
	server := newFakeRuntimeServer()
	server.blockServe = true
	server.blockGraceful = true
	runtime := newTestRuntime(server)

	if err := runtime.Start(nil); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitClosed(t, server.serveStarted, "Serve start")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := runtime.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown error=%v want DeadlineExceeded", err)
	}

	if got := server.gracefulCalls(); got != 1 {
		t.Fatalf("GracefulStop calls=%d want=1", got)
	}
	if got := server.stopCalls(); got != 1 {
		t.Fatalf("Stop calls=%d want=1", got)
	}

	if err := runtime.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown #2: %v", err)
	}
	if got := server.gracefulCalls(); got != 1 {
		t.Fatalf("GracefulStop calls after repeat=%d want=1", got)
	}
	if got := server.stopCalls(); got != 1 {
		t.Fatalf("Stop calls after repeat=%d want=1", got)
	}

	if runtime.Err() != nil {
		t.Fatalf("Err=%v want nil after forced shutdown", runtime.Err())
	}
}

func newTestRuntime(server *fakeRuntimeServer) *Runtime {
	return &Runtime{
		addr:         "127.0.0.1:0",
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		listener:     fakeListener{},
		server:       server,
		state:        runtimeStateNew,
		done:         make(chan struct{}),
		gracefulDone: make(chan struct{}),
	}
}

type fakeRuntimeServer struct {
	serveStarted chan struct{}
	stopped      chan struct{}
	startOnce    sync.Once
	stopOnce     sync.Once

	mu            sync.Mutex
	serveCount    int
	gracefulCount int
	stopCount     int

	serveErr      error
	blockServe    bool
	blockGraceful bool
}

func newFakeRuntimeServer() *fakeRuntimeServer {
	return &fakeRuntimeServer{
		serveStarted: make(chan struct{}),
		stopped:      make(chan struct{}),
	}
}

func (s *fakeRuntimeServer) Serve(net.Listener) error {
	s.mu.Lock()
	s.serveCount++
	s.mu.Unlock()
	s.startOnce.Do(func() { close(s.serveStarted) })

	if s.blockServe {
		<-s.stopped
		if s.serveErr == nil {
			return grpc.ErrServerStopped
		}
	}
	return s.serveErr
}

func (s *fakeRuntimeServer) GracefulStop() {
	s.mu.Lock()
	s.gracefulCount++
	s.mu.Unlock()

	if s.blockGraceful {
		<-s.stopped
		return
	}
	s.stopOnce.Do(func() { close(s.stopped) })
}

func (s *fakeRuntimeServer) Stop() {
	s.mu.Lock()
	s.stopCount++
	s.mu.Unlock()
	s.stopOnce.Do(func() { close(s.stopped) })
}

func (s *fakeRuntimeServer) serveCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.serveCount
}

func (s *fakeRuntimeServer) gracefulCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gracefulCount
}

func (s *fakeRuntimeServer) stopCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopCount
}

func waitClosed(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for %s", name)
	}
}

type fakeListener struct{}

func (fakeListener) Accept() (net.Conn, error) { return nil, errors.New("not used") }
func (fakeListener) Close() error              { return nil }
func (fakeListener) Addr() net.Addr            { return fakeAddr("127.0.0.1:0") }

type fakeAddr string

func (a fakeAddr) Network() string { return "tcp" }
func (a fakeAddr) String() string  { return string(a) }
