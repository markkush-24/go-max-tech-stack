package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"pet-study/internal/interceptors"
	"pet-study/internal/security"
	"pet-study/internal/service"
	"pet-study/internal/transport/pb"
	"strings"
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

type Config struct {
	Addr              string
	ReflectionEnabled bool
	Auth              AuthConfig
	TLS               TLSConfig
}

type AuthConfig struct {
	Verifier security.Verifier
}

type TLSConfig struct {
	Enable       bool
	CertFile     string
	KeyFile      string
	ClientCAFile string
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
	return NewRuntimeWithConfig(Config{Addr: addr}, jobService, logger)
}

func NewRuntimeWithConfig(cfg Config, jobService *service.JobService, logger *slog.Logger) (*Runtime, error) {
	if strings.TrimSpace(cfg.Addr) == "" {
		return nil, fmt.Errorf("grpc addr is required")
	}

	if cfg.ReflectionEnabled && !isLoopbackListenAddr(cfg.Addr) {
		return nil, fmt.Errorf("grpc reflection requires a loopback listener")
	}

	authInterceptor, err := interceptors.UnaryAuthenticate(cfg.Auth.Verifier)
	if err != nil {
		return nil, err
	}

	serverOptions := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			interceptors.UnaryRequestIDAndLogging(logger),
			authInterceptor,
		),
	}

	if cfg.TLS.Enable {
		creds, err := newServerTLSCredentials(cfg.TLS)
		if err != nil {
			return nil, err
		}
		serverOptions = append(serverOptions, grpc.Creds(creds))
	} else if !isLoopbackListenAddr(cfg.Addr) {
		return nil, fmt.Errorf("plaintext grpc requires a loopback listener")
	}

	grpcServer := grpc.NewServer(serverOptions...)

	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, err
	}

	grpcJobService := NewJobServer(jobService)

	pb.RegisterJobsServiceServer(grpcServer, grpcJobService)
	if cfg.ReflectionEnabled {
		reflection.Register(grpcServer)
	}

	return &Runtime{
		addr:         listener.Addr().String(),
		logger:       logger,
		listener:     listener,
		server:       grpcServer,
		state:        runtimeStateNew,
		done:         make(chan struct{}),
		gracefulDone: make(chan struct{}),
	}, nil
}

func (r *Runtime) Addr() string {
	if r == nil {
		return ""
	}
	return r.addr
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
