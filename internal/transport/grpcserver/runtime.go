package grpcserver

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"pet-study/internal/interceptors"
	"pet-study/internal/service"
	"pet-study/internal/transport/pb"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var ErrGRPCServerNotRunning = errors.New("grpc server not running")

type Runtime struct {
	addr     string
	logger   *slog.Logger
	listener net.Listener
	server   *grpc.Server
	stopping atomic.Bool
	running  atomic.Bool
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
		addr:     addr,
		logger:   logger,
		listener: listener,
		server:   grpcServer,
	}, nil
}

func (r *Runtime) Start(stop context.CancelFunc) {
	go func() {
		r.logger.Info("gRPC server listening", "addr", r.addr)

		r.running.Store(true)
		err := r.server.Serve(r.listener)
		r.running.Store(false)

		if err != nil && !r.stopping.Load() {
			r.logger.Error("gRPC server failed", "addr", r.addr, "err", err)
			stop()
		}
	}()
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}

	r.stopping.Store(true)

	done := make(chan struct{})
	go func() {
		r.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		r.logger.Warn("gRPC graceful shutdown timed out, forcing stop")
		r.server.Stop()
		<-done
		return ctx.Err()
	}
}

func (r *Runtime) Ready(ctx context.Context) error {
	if r == nil {
		return ErrGRPCServerNotRunning
	}

	if !r.running.Load() || r.stopping.Load() {
		return ErrGRPCServerNotRunning
	}

	return nil
}
