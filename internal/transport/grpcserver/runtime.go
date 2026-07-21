package grpcserver

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"pet-study/internal/interceptors"
	"pet-study/internal/service"
	"pet-study/internal/transport/pb"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	ErrGRPCServerNotRunning      = errors.New("grpc server not running")
	ErrGRPCServerAlreadyStarted  = errors.New("grpc server already started")
	ErrGRPCServerRuntimeRequired = errors.New("grpc server runtime is nil")
)

type runtimeState uint8

const (
	runtimeStateNew runtimeState = iota
	runtimeStateRunning
	runtimeStateStopping
	runtimeStateStopped
)

type grpcRuntimeServer interface {
	Serve(net.Listener) error
	GracefulStop()
	Stop()
}

type Runtime struct {
	addr     string
	logger   *slog.Logger
	listener net.Listener
	server   grpcRuntimeServer

	mu            sync.Mutex
	state         runtimeState
	err           error
	done          chan struct{}
	doneOnce      sync.Once
	shutdownOnce  sync.Once
	forceStopOnce sync.Once
	gracefulDone  chan struct{}
}

func NewRuntime(addr string, jobService *service.JobService, logger *slog.Logger) (*Runtime, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	grpcJobService := NewJobServer(jobService)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.UnaryRequestIDAndLogging(logger),
		),
	)

	pb.RegisterJobsServiceServer(grpcServer, grpcJobService)
	reflection.Register(grpcServer)

	return &Runtime{
		addr:         addr,
		logger:       logger,
		listener:     listener,
		server:       grpcServer,
		state:        runtimeStateNew,
		done:         make(chan struct{}),
		gracefulDone: make(chan struct{}),
	}, nil
}

func (r *Runtime) Start(stop context.CancelFunc) error {
	if r == nil {
		return ErrGRPCServerRuntimeRequired
	}

	r.mu.Lock()
	if r.state != runtimeStateNew {
		r.mu.Unlock()
		return ErrGRPCServerAlreadyStarted
	}
	r.state = runtimeStateRunning
	r.mu.Unlock()

	go func() {
		r.logger.Info("gRPC server listening", "addr", r.addr)

		err := r.server.Serve(r.listener)
		fatal := r.finishServe(err)

		if fatal {
			r.logger.Error("gRPC server failed", "addr", r.addr, "err", err)
			if stop != nil {
				stop()
			}
		}
	}()

	return nil
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}

	needsGracefulStop := r.beginShutdown()
	if !needsGracefulStop {
		return nil
	}

	r.shutdownOnce.Do(func() {
		go func() {
			r.server.GracefulStop()
			close(r.gracefulDone)
		}()
	})

	select {
	case <-r.gracefulDone:
		<-r.done
		return nil
	case <-ctx.Done():
		r.logger.Warn("gRPC graceful shutdown timed out, forcing stop")
		r.forceStopOnce.Do(r.server.Stop)
		<-r.gracefulDone
		<-r.done
		return ctx.Err()
	}
}

func (r *Runtime) Ready(ctx context.Context) error {
	if r == nil {
		return ErrGRPCServerNotRunning
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state != runtimeStateRunning {
		return ErrGRPCServerNotRunning
	}

	return nil
}

func (r *Runtime) Done() <-chan struct{} {
	if r == nil {
		done := make(chan struct{})
		close(done)
		return done
	}

	return r.done
}

func (r *Runtime) Err() error {
	if r == nil {
		return ErrGRPCServerRuntimeRequired
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

func (r *Runtime) beginShutdown() bool {
	r.mu.Lock()
	switch r.state {
	case runtimeStateNew:
		r.state = runtimeStateStopped
		r.mu.Unlock()
		r.server.Stop()
		_ = r.listener.Close()
		r.doneOnce.Do(func() { close(r.done) })
		return false
	case runtimeStateRunning:
		r.state = runtimeStateStopping
		r.mu.Unlock()
		return true
	case runtimeStateStopping:
		r.mu.Unlock()
		return true
	case runtimeStateStopped:
		r.mu.Unlock()
		return false
	default:
		r.mu.Unlock()
		return false
	}
}

func (r *Runtime) finishServe(err error) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	fatal := err != nil && r.state == runtimeStateRunning && !errors.Is(err, grpc.ErrServerStopped)
	if fatal {
		r.err = err
	}
	r.state = runtimeStateStopped

	r.doneOnce.Do(func() { close(r.done) })
	return fatal
}
