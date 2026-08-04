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

type shutdownOutcome string

const (
	shutdownOutcomeGraceful shutdownOutcome = "graceful"
	shutdownOutcomeForced   shutdownOutcome = "forced"
	shutdownOutcomeTimedOut shutdownOutcome = "timed_out"
	shutdownOutcomeFailed   shutdownOutcome = "failed"
)

const (
	logComponentAPIServer = "api_server"
	logFieldEvent         = "event"
	logFieldTrigger       = "trigger"
	logFieldOutcome       = "outcome"
)

type componentShutdownError struct {
	Component string
	Outcome   shutdownOutcome
	Err       error
}

func (e *componentShutdownError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s shutdown %s", e.Component, e.Outcome)
	}
	return fmt.Sprintf("%s shutdown %s: %v", e.Component, e.Outcome, e.Err)
}

func (e *componentShutdownError) Unwrap() error {
	return e.Err
}

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
	logger := slog.Default().With(config.LogFieldComponent, logComponentAPIServer)
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
	shutdownTrigger := "context"
	select {
	case <-ctx.Done():
		shutdownTrigger = "context"
	case err := <-errCh:
		shutdownTrigger = err.Name + "_server_error"
		logger.Error("server error", "server", err.Name, config.LogFieldError, err.Err)
		runErr = fmt.Errorf("%s server failed: %w", err.Name, err.Err)
	case <-grpcDone:
		if err := s.grpcRuntime.Err(); err != nil {
			shutdownTrigger = "grpc_server_error"
			logger.Error("server error", "server", "grpc", config.LogFieldError, err)
			runErr = fmt.Errorf("grpc server failed: %w", err)
		} else {
			shutdownTrigger = "grpc_stopped"
		}
	}

	shutdownStarted := time.Now()
	logger.Info(
		"shutdown started",
		logFieldEvent, "shutdown.started",
		logFieldTrigger, shutdownTrigger,
		"shutdown_timeout_ms", s.config.HTTP.ShutdownTimeout.Milliseconds(),
	)
	cleanupErr := s.cleanup(logger, srv, httpsSrv, &wg)
	s.logShutdownSummary(logger, shutdownStarted, shutdownTrigger, runErr, cleanupErr)
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.config.HTTP.ShutdownTimeout)
	defer cancel()

	s.readiness.SetNotReady()
	logger.Info("readiness changed", "ready", false)

	s.queue.StopAccepting()

	var shutdownErrs []error

	shutdownErrs = append(shutdownErrs, shutdownHTTPServers(shutdownCtx, logger, httpSrv, httpsSrv)...)

	if s.grpcRuntime != nil {
		if err := shutdownGRPC(shutdownCtx, logger, s.grpcRuntime); err != nil {
			shutdownErrs = append(shutdownErrs, err)
		}
	}

	if err := shutdownWorkerPool(shutdownCtx, logger, s.pool); err != nil {
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

func (s *APIServer) logShutdownSummary(
	logger *slog.Logger,
	start time.Time,
	trigger string,
	runErr error,
	cleanupErr error,
) {
	outcome := overallShutdownOutcome(runErr, cleanupErr)
	level := slog.LevelInfo
	if outcome != string(shutdownOutcomeGraceful) {
		level = slog.LevelWarn
	}

	attrs := []slog.Attr{
		slog.String(logFieldEvent, "shutdown.completed"),
		slog.String(logFieldTrigger, trigger),
		slog.String(logFieldOutcome, outcome),
		slog.Int64(config.LogFieldDurationMS, time.Since(start).Milliseconds()),
	}

	if s.queue != nil {
		attrs = append(attrs, slog.Int("queue_depth", len(s.queue.Chan())))
	}
	if s.eventHub != nil {
		attrs = append(attrs,
			slog.Int64("sse_subscribers", s.eventHub.Subscribers()),
			slog.Int64("sse_events_total", s.eventHub.EventsTotal()),
			slog.Int64("sse_drops_total", s.eventHub.DropsTotal()),
		)
	}
	if s.pool != nil {
		if stopOutcome, ok := s.pool.LastStopOutcome(); ok {
			attrs = append(attrs, slog.Int("repaired_active_jobs", stopOutcome.RepairedActiveJobs))
		}
	}

	logger.LogAttrs(context.Background(), level, "shutdown completed", attrs...)
}

func overallShutdownOutcome(runErr error, cleanupErr error) string {
	switch {
	case runErr == nil && cleanupErr == nil:
		return string(shutdownOutcomeGraceful)
	case hasShutdownOutcome(cleanupErr, shutdownOutcomeForced):
		return string(shutdownOutcomeForced)
	case hasShutdownOutcome(cleanupErr, shutdownOutcomeTimedOut):
		return string(shutdownOutcomeTimedOut)
	default:
		return string(shutdownOutcomeFailed)
	}
}

func hasShutdownOutcome(err error, outcome shutdownOutcome) bool {
	if err == nil {
		return false
	}

	var componentErr *componentShutdownError
	if errors.As(err, &componentErr) && componentErr.Outcome == outcome {
		return true
	}

	type multiUnwrapper interface {
		Unwrap() []error
	}
	if unwrapped, ok := err.(multiUnwrapper); ok {
		for _, child := range unwrapped.Unwrap() {
			if hasShutdownOutcome(child, outcome) {
				return true
			}
		}
		return false
	}

	type singleUnwrapper interface {
		Unwrap() error
	}
	if unwrapped, ok := err.(singleUnwrapper); ok {
		return hasShutdownOutcome(unwrapped.Unwrap(), outcome)
	}

	return false
}

type namedHTTPServer struct {
	name string
	srv  *http.Server
}

func shutdownHTTPServers(ctx context.Context, logger *slog.Logger, httpSrv, httpsSrv *http.Server) []error {
	servers := []namedHTTPServer{
		{name: "http", srv: httpSrv},
	}
	if httpsSrv != nil {
		servers = append(servers, namedHTTPServer{name: "https", srv: httpsSrv})
	}

	errCh := make(chan error, len(servers))
	var wg sync.WaitGroup
	for _, server := range servers {
		server := server
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := shutdownServer(ctx, logger, server.name, server.srv); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	return errs
}

func shutdownServer(ctx context.Context, logger *slog.Logger, name string, srv *http.Server) error {
	if srv == nil {
		return nil
	}

	sdCtx, cancel := componentShutdownContext(ctx)
	defer cancel()

	if err := srv.Shutdown(sdCtx); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		if errors.Is(err, context.DeadlineExceeded) {
			logger.Warn("shutdown timeout, forcing close", "server", name)
			if closeErr := srv.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
				return errors.Join(
					newComponentShutdownError(name, shutdownOutcomeForced, err),
					newComponentShutdownError(name, shutdownOutcomeFailed, closeErr),
				)
			}
			return newComponentShutdownError(name, shutdownOutcomeForced, err)
		}

		if closeErr := srv.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
			return errors.Join(
				newComponentShutdownError(name, shutdownOutcomeFailed, err),
				newComponentShutdownError(name, shutdownOutcomeFailed, closeErr),
			)
		}
		return newComponentShutdownError(name, shutdownOutcomeFailed, err)
	}

	logger.Info("component shutdown complete", "target", name, "outcome", string(shutdownOutcomeGraceful))
	return nil
}

func shutdownGRPC(ctx context.Context, logger *slog.Logger, runtime grpcRuntime) error {
	grpcCtx, cancel := componentShutdownContext(ctx)
	defer cancel()

	if err := runtime.Shutdown(grpcCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			logger.Warn("gRPC shutdown timeout, forced stop attempted")
			return newComponentShutdownError("grpc", shutdownOutcomeForced, err)
		}
		return newComponentShutdownError("grpc", shutdownOutcomeFailed, err)
	}
	logger.Info("component shutdown complete", "target", "grpc", "outcome", string(shutdownOutcomeGraceful))
	return nil
}

func shutdownWorkerPool(ctx context.Context, logger *slog.Logger, pool *workerpool.WorkerPool) error {
	stopCtx, cancel := componentShutdownContext(ctx)
	defer cancel()

	if err := pool.Stop(stopCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			logger.Warn("worker pool shutdown timed out")
			return newComponentShutdownError("workerpool", shutdownOutcomeTimedOut, err)
		}
		return newComponentShutdownError("workerpool", shutdownOutcomeFailed, err)
	}
	logger.Info("component shutdown complete", "target", "workerpool", "outcome", string(shutdownOutcomeGraceful))
	return nil
}

func componentShutdownContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if deadline, ok := ctx.Deadline(); ok {
		return context.WithDeadline(ctx, deadline)
	}
	return context.WithCancel(ctx)
}

func newComponentShutdownError(component string, outcome shutdownOutcome, err error) error {
	return &componentShutdownError{
		Component: component,
		Outcome:   outcome,
		Err:       err,
	}
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
