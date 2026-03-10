package api

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"pet-study/internal/config"
	"pet-study/internal/health"
	"pet-study/internal/queue"
	"pet-study/internal/workerpool"
)

type APIServer struct {
	config    config.Config
	router    http.Handler
	readiness *health.Readiness
	pool      *workerpool.WorkerPool
	queue     *queue.Queue
}

func NewAPIServer(
	config config.Config,
	router http.Handler,
	readiness *health.Readiness,
	pool *workerpool.WorkerPool,
	queue *queue.Queue,
) *APIServer {
	return &APIServer{
		config:    config,
		router:    router,
		readiness: readiness,
		pool:      pool,
		queue:     queue,
	}
}

func (s *APIServer) Run(ctx context.Context) error {
	logger := slog.Default().With("component", "api_server")
	srv := &http.Server{
		Addr:              s.config.HTTP.Addr,
		Handler:           s.router,
		ReadHeaderTimeout: s.config.HTTP.ReadHeaderTimeout,
		ReadTimeout:       s.config.HTTP.ReadTimeout,
		WriteTimeout:      s.config.HTTP.WriteTimeout,
		IdleTimeout:       s.config.HTTP.IdleTimeout,
		MaxHeaderBytes:    s.config.HTTP.MaxHeaderBytes,
	}

	ln, err := net.Listen("tcp", s.config.HTTP.Addr)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)

	go func() {
		logger.Info("http server listening", "addr", ln.Addr().String())

		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	s.readiness.SetReady()
	logger.Info("readiness changed", "ready", true)

	defer func() {
		s.queue.StopAccepting()

		stopCtx, cancel := context.WithTimeout(context.Background(), s.config.HTTP.ShutdownTimeout)
		defer cancel()

		if err := s.pool.Stop(stopCtx); err != nil {
			logger.Error("worker pool stop failed", "err", err)
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown started")
		s.readiness.SetNotReady()
		logger.Info("readiness changed", "ready", false)

		sdCtx, cancel := context.WithTimeout(context.Background(), s.config.HTTP.ShutdownTimeout)
		defer cancel()

		s.queue.StopAccepting()

		if err := srv.Shutdown(sdCtx); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				logger.Warn("shutdown timeout, forcing close", "timeout", s.config.HTTP.ShutdownTimeout.String())

				// fallback: жёстко закрываем
				if closeErr := srv.Close(); closeErr != nil {
					return errors.Join(err, closeErr)
				}

				<-errCh
				return nil
			}

			_ = srv.Close()
			<-errCh

			return err
		}

		<-errCh
		return nil

	case err := <-errCh:
		if err == nil {
			logger.Info("server stopped")
			return nil
		}

		logger.Error("http server error", "err", err)
		return err
	}
}
