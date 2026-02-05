package api

import (
	"context"
	"errors"
	"log"
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

// Run запускает HTTP-сервер и останавливает его по ctx.
// Контекст сигналов и общий lifecycle приходят "сверху".
func (s *APIServer) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:              s.config.HTTP.Addr,
		Handler:           s.router,
		ReadHeaderTimeout: s.config.HTTP.ReadHeaderTimeout, // caps time to read request headers (slowloris mitigation)
		ReadTimeout:       s.config.HTTP.ReadTimeout,
		WriteTimeout:      s.config.HTTP.WriteTimeout,
		IdleTimeout:       s.config.HTTP.IdleTimeout,
		MaxHeaderBytes:    s.config.HTTP.MaxHeaderBytes,
	}

	errCh := make(chan error, 1)

	// Стартуем сервер в отдельной горутине.
	go func() {
		log.Printf("HTTP server listening on %s", s.config.HTTP.Addr)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Ошибку отдаём наверх через канал — без os.Exit/log.Fatalf
			errCh <- err
		}
		close(errCh)
	}()

	s.readiness.SetReady()
	log.Println("Set ready true")

	// Ждём либо остановки по контексту, либо ошибки сервера.
	select {
	case <-ctx.Done():
		log.Println("shutting down...")
		s.readiness.SetNotReady()
		log.Println("Set ready false")

		sdCtx, cancel := context.WithTimeout(context.Background(), s.config.HTTP.ShutdownTimeout)
		defer cancel()

		s.queue.StopAccepting()

		if err := srv.Shutdown(sdCtx); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				log.Printf("shutdown timeout (%s), forcing close", s.config.HTTP.ShutdownTimeout)

				// fallback: жёстко закрываем
				if closeErr := srv.Close(); closeErr != nil {
					return errors.Join(err, closeErr)
				}

				<-errCh
				return nil // политика: таймаут — не “ошибка запуска”, мы уже остановились
			}

			// best-effort hard close, чтобы не оставлять хвосты
			_ = srv.Close()
			<-errCh

			return err
		}

		poolCtx, cancelPool := context.WithTimeout(context.Background(), s.config.HTTP.ShutdownTimeout)
		defer cancelPool()

		if s.pool.IsRunning() {
			if err := s.pool.Stop(poolCtx); err != nil {
				log.Printf("worker pool stop: %v", err)
			}
		}

		<-errCh
		return nil

	case err := <-errCh:
		if err == nil {
			log.Println("server stopped")
			return nil
		}

		log.Printf("http server error: %v", err)
		return err
	}
}
