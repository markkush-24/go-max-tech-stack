package api

import (
	"context"
	"log"
	"net/http"
	"pet-study/internal/health"
	"time"
)

type APIServer struct {
	addr      string
	router    http.Handler
	readiness *health.Readiness
}

func NewAPIServer(addr string, router http.Handler, readiness *health.Readiness) *APIServer {
	return &APIServer{
		addr:      addr,
		router:    router,
		readiness: readiness,
	}
}

// Run запускает HTTP-сервер и останавливает его по ctx.
// Контекст сигналов и общий lifecycle приходят "сверху".
func (s *APIServer) Run(ctx context.Context) error {
	srv := &http.Server{
		Addr:              s.addr,
		Handler:           s.router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)

	// Стартуем сервер в отдельной горутине.
	go func() {
		log.Printf("HTTP server listening on %s", s.addr)

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

		// ВАЖНО: таймаут для Shutdown — на отдельном контексте,
		// а не на уже отменённом ctx.
		sdCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(sdCtx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
			return err
		}

		log.Println("server stopped cleanly")
		return nil

	case err := <-errCh:
		// Если канал закрылся без ошибки — просто выходим.
		if err == nil {
			log.Println("server stopped")
			return nil
		}

		log.Printf("http server error: %v", err)
		return err
	}
}
