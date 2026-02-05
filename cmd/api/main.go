package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"pet-study/internal/metrics"
	"pet-study/internal/middleware"
	"pet-study/internal/queue"
	"pet-study/internal/store/jobrepo"
	"syscall"

	"pet-study/internal/api"
	"pet-study/internal/config"
	"pet-study/internal/health"
	"pet-study/internal/requestid"
	"pet-study/internal/router"
	"pet-study/internal/routes"
	"pet-study/internal/service"
	"pet-study/internal/store/userrepo"
	"pet-study/internal/workerpool"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Dependencies
	userRepository := userrepo.NewMemoryUserRepository()
	jobRepository := jobrepo.NewMemoryJobRepository()
	userService := service.NewUserService(userRepository)
	jobService := service.NewJobService(jobRepository)

	//Async
	q := queue.New(10)
	pool := workerpool.NewWorkerPool(q, jobService, userService)
	poolErr := pool.Start(cfg.Pool.Workers)
	if poolErr != nil {
		return poolErr
	}

	readiness := health.NewReadiness(
		health.Check{Name: "repo", Fn: userRepository.Ping},
		health.Check{Name: "workerpool", Fn: pool.CheckRunning})

	var debugRouter http.Handler
	if cfg.HTTP.Debug {
		debugRouter = router.NewDebugRouter()
	}

	limitedAPI := middleware.NewRateLimitedAPI(float64(cfg.Limiter.RPS), cfg.Limiter.BURST)

	// Routers
	userHandler := routes.NewUserHandler(userService, jobService, q)
	userHandlerV2 := routes.NewUserV2Handler(userService, jobService, q)

	jobsHandler := routes.NewJobHandler(jobService)

	userRouter := router.NewRouter(userHandler, userHandlerV2, jobsHandler, limitedAPI)

	healthRouter := router.NewHealthRouter(readiness)
	rootRouter := router.NewRoot(userRouter, healthRouter, debugRouter)

	// Metrics registry
	m := metrics.DefaultHTTP()

	// Middleware chain (outer -> inner):
	// RequestID -> Metrics -> Logger -> Recover -> Router
	handler := rootRouter
	handler = middleware.Recover(handler) // inner: чтобы Logger/Metrics увидели 500 при panic в Router
	handler = middleware.Logger(handler)
	handler = middleware.Metrics(m)(handler)
	handler = middleware.Recover(handler) // outer: ловит panic в Logger/Metrics
	handler = requestid.RequestIDMiddleware(handler)

	server := api.NewAPIServer(cfg, handler, readiness, pool, q)
	log.Printf("config: addr=%s debug=%v", cfg.HTTP.Addr, cfg.HTTP.Debug)
	return server.Run(ctx)
}
