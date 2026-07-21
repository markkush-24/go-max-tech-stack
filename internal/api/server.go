package api

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"pet-study/internal/config"
	"pet-study/internal/health"
	"pet-study/internal/queue"
	"pet-study/internal/stream"
	"pet-study/internal/workerpool"
	"sync"
	"time"
)

type grpcRuntime interface {
	Start(stop context.CancelFunc) error
	Shutdown(ctx context.Context) error
	Done() <-chan struct{}
	Err() error
}

type listenFunc func(network, addr string) (net.Listener, error)

type APIServer struct {
	config      config.Config
	router      http.Handler
	readiness   *health.Readiness
	pool        *workerpool.WorkerPool
	queue       *queue.Queue
	grpcRuntime grpcRuntime
	eventHub    *stream.Hub
	listen      listenFunc
}

func NewAPIServer(
	config config.Config,
	router http.Handler,
	readiness *health.Readiness,
	pool *workerpool.WorkerPool,
	queue *queue.Queue,
	grpcRuntime grpcRuntime,
	eventHub *stream.Hub,
) *APIServer {
	return &APIServer{
		config:      config,
		router:      router,
		readiness:   readiness,
		pool:        pool,
		queue:       queue,
		grpcRuntime: grpcRuntime,
		eventHub:    eventHub,
		listen:      net.Listen,
	}
}

func (s *APIServer) Run(ctx context.Context) error {
	logger := slog.Default().With("component", "api_server")
	srv := s.newHTTPServer(s.config.HTTP.Addr, nil)

	ln, err := s.listen("tcp", s.config.HTTP.Addr)
	if err != nil {
		return err
	}

	var httpsSrv *http.Server
	var lnHTTPS net.Listener
	if s.config.HTTP.TLS.Enable {
		cert, err := tls.LoadX509KeyPair(s.config.HTTP.TLS.CertFile, s.config.HTTP.TLS.KeyFile)
		if err != nil {
			_ = ln.Close()
			return err
		}

		httpsSrv = s.newHTTPServer(
			s.config.HTTP.TLS.Addr,
			&tls.Config{
				Certificates: []tls.Certificate{cert},
			},
		)

		lnHTTPS, err = s.listen("tcp", s.config.HTTP.TLS.Addr)
		if err != nil {
			_ = ln.Close()
			return err
		}
	}

	if httpsSrv == nil {
		logger.Info("https server disabled")
	}

	if err := s.pool.Start(ctx, s.config.Pool.Workers); err != nil {
		_ = ln.Close()
		if lnHTTPS != nil {
			_ = lnHTTPS.Close()
		}
		return err
	}

	if s.grpcRuntime != nil {
		if err := s.grpcRuntime.Start(nil); err != nil {
			stopCtx, cancel := context.WithTimeout(context.Background(), s.config.HTTP.ShutdownTimeout)
			defer cancel()
			_ = s.pool.Stop(stopCtx)
			_ = ln.Close()
			if lnHTTPS != nil {
				_ = lnHTTPS.Close()
			}
			return err
		}
	}

	errCh := make(chan componentError, 2)
	grpcDone := doneChan(s.grpcRuntime)
	var wg sync.WaitGroup

	wg.Add(1)

	/*
		Запускаем сервер из отдельной горутины в данном случае запускается сервер http
	*/
	go func() {
		defer wg.Done()

		logger.Info("http server listening", "addr", ln.Addr().String())

		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- componentError{Name: "http", Err: err}
		}
	}()

	/*
		Запускаем сервер из отдельной горутины в данном случае запускается сервер https
	*/
	if httpsSrv != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()

			logger.Info("https server listening", "addr", lnHTTPS.Addr().String())

			if err := httpsSrv.ServeTLS(lnHTTPS, "", ""); err != nil && err != http.ErrServerClosed {
				errCh <- componentError{Name: "https", Err: err}
			}
		}()
	}

	s.readiness.SetReady()
	logger.Info("readiness changed", "ready", true)

	var runErr error
	select {
	case <-ctx.Done():
		logger.Info("shutdown started")
	case err := <-errCh:
		logger.Error("server error", "server", err.Name, "err", err.Err)
		runErr = fmt.Errorf("%s server failed: %w", err.Name, err.Err)
	case <-grpcDone:
		if err := s.grpcRuntime.Err(); err != nil {
			logger.Error("server error", "server", "grpc", "err", err)
			runErr = fmt.Errorf("grpc server failed: %w", err)
		}
	}

	cleanupErr := s.cleanup(logger, srv, httpsSrv, &wg)
	return errors.Join(runErr, cleanupErr)
}

type componentError struct {
	Name string
	Err  error
}

func doneChan(runtime grpcRuntime) <-chan struct{} {
	if runtime == nil {
		return nil
	}
	return runtime.Done()
}

func (s *APIServer) cleanup(logger *slog.Logger, httpSrv, httpsSrv *http.Server, wg *sync.WaitGroup) error {
	s.readiness.SetNotReady()
	logger.Info("readiness changed", "ready", false)

	s.queue.StopAccepting()

	var shutdownErrs []error

	if httpsSrv != nil {
		if err := shutdownServer(logger, "https", httpsSrv, s.config.HTTP.ShutdownTimeout); err != nil {
			shutdownErrs = append(shutdownErrs, err)
		}
	}

	if err := shutdownServer(logger, "http", httpSrv, s.config.HTTP.ShutdownTimeout); err != nil {
		shutdownErrs = append(shutdownErrs, err)
	}

	if s.grpcRuntime != nil {
		grpcCtx, cancel := context.WithTimeout(context.Background(), s.config.HTTP.ShutdownTimeout)
		defer cancel()

		if err := s.grpcRuntime.Shutdown(grpcCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
			shutdownErrs = append(shutdownErrs, err)
		}
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), s.config.HTTP.ShutdownTimeout)
	defer cancel()

	if err := s.pool.Stop(stopCtx); err != nil {
		logger.Error("worker pool stop failed", "err", err)
		shutdownErrs = append(shutdownErrs, err)
	}

	if s.eventHub != nil {
		if err := s.eventHub.Close(); err != nil {
			shutdownErrs = append(shutdownErrs, err)
		}
	}

	wg.Wait()

	if len(shutdownErrs) > 0 {
		return errors.Join(shutdownErrs...)
	}
	return nil
}

func shutdownServer(logger *slog.Logger, name string, srv *http.Server, timeout time.Duration) error {
	if srv == nil {
		return nil
	}

	sdCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(sdCtx); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		if errors.Is(err, context.DeadlineExceeded) {
			logger.Warn("shutdown timeout, forcing close", "server", name, "timeout", timeout.String())
			if closeErr := srv.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
				return errors.Join(err, closeErr)
			}
			return nil
		}

		if closeErr := srv.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
			return errors.Join(err, closeErr)
		}
		return err
	}

	return nil
}

func (s *APIServer) newHTTPServer(addr string, tlsCfg *tls.Config) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           s.router,
		ReadHeaderTimeout: s.config.HTTP.ReadHeaderTimeout,
		ReadTimeout:       s.config.HTTP.ReadTimeout,
		WriteTimeout:      s.config.HTTP.WriteTimeout,
		IdleTimeout:       s.config.HTTP.IdleTimeout,
		MaxHeaderBytes:    s.config.HTTP.MaxHeaderBytes,
		TLSConfig:         tlsCfg,
	}
}
