package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"pet-study/internal/metrics"
	"pet-study/internal/middleware"
	"syscall"

	"pet-study/internal/api"
	"pet-study/internal/config"
	"pet-study/internal/health"
	"pet-study/internal/requestid"
	"pet-study/internal/router"
	"pet-study/internal/routes"
	"pet-study/internal/service"
	"pet-study/internal/store/userrepo"
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
	userService := service.NewUserService(userRepository)

	readiness := health.NewReadiness(health.Check{
		Name: "repo",
		Fn:   userRepository.Ping,
	})

	var debugRouter http.Handler
	if cfg.HTTP.Debug {
		debugRouter = router.NewDebugRouter()
	}

	// Routers
	userHandler := routes.NewUserHandler(userService)
	userHandlerV2 := routes.NewUserV2Handler(userService)
	userRouter := router.NewRouter(userHandler, userHandlerV2)

	healthRouter := router.NewHealthRouter(readiness)
	rootRouter := router.NewRoot(userRouter, healthRouter, debugRouter)

	// Metrics registry
	m := metrics.DefaultHTTP()

	// Middleware chain (outer -> inner):
	// RequestID -> Metrics -> Logger -> Recover -> Router
	handler := rootRouter
	handler = middleware.Recover(handler)
	handler = middleware.Logger(handler)
	handler = middleware.Metrics(m)(handler)
	handler = requestid.RequestIDMiddleware(handler)

	server := api.NewAPIServer(cfg, handler, readiness)
	log.Printf("config: addr=%s debug=%v", cfg.HTTP.Addr, cfg.HTTP.Debug)
	return server.Run(ctx)
}
