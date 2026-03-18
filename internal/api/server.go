package api

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"pet-study/internal/config"
	"pet-study/internal/health"
	"pet-study/internal/queue"
	"pet-study/internal/workerpool"
	"sync"
	"time"
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
	srv := s.newHTTPServer(s.config.HTTP.Addr, nil)

	ln, err := net.Listen("tcp", s.config.HTTP.Addr)
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

		lnHTTPS, err = net.Listen("tcp", s.config.HTTP.TLS.Addr)
		if err != nil {
			_ = ln.Close()
			return err
		}
	}

	if httpsSrv == nil {
		logger.Info("https server disabled")
	}

	errCh := make(chan error, 2)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		logger.Info("http server listening", "addr", ln.Addr().String())

		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	if httpsSrv != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()

			logger.Info("https server listening", "addr", lnHTTPS.Addr().String())

			if err := httpsSrv.ServeTLS(lnHTTPS, "", ""); err != nil && err != http.ErrServerClosed {
				errCh <- err
			}
		}()
	}

	s.readiness.SetReady()
	logger.Info("readiness changed", "ready", true)

	defer func() {
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

		s.queue.StopAccepting()

		var shutdownErrs []error

		if httpsSrv != nil {
			if err := shutdownServer(logger, "https", httpsSrv, s.config.HTTP.ShutdownTimeout); err != nil {
				shutdownErrs = append(shutdownErrs, err)
			}
		}

		if err := shutdownServer(logger, "http", srv, s.config.HTTP.ShutdownTimeout); err != nil {
			shutdownErrs = append(shutdownErrs, err)
		}

		wg.Wait()

		if len(shutdownErrs) > 0 {
			return errors.Join(shutdownErrs...)
		}

		return nil

	case err := <-errCh:
		s.readiness.SetNotReady()
		logger.Info("readiness changed", "ready", false)

		s.queue.StopAccepting()

		_ = srv.Close()
		if httpsSrv != nil {
			_ = httpsSrv.Close()
		}
		wg.Wait()

		logger.Error("server error", "err", err)
		return err
	}
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
