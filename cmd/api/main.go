package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"pet-study/internal/middleware"
	"syscall"

	"pet-study/internal/api"
	"pet-study/internal/config"
	"pet-study/internal/health"
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

	userRepository := userrepo.NewMemoryUserRepository()
	userService := service.NewUserService(userRepository)

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	readiness := health.NewReadiness()

	userHandler := routes.NewUserHandler(userService)
	userHandlerV2 := routes.NewUserV2Handler(userService)
	userRouter := router.NewRouter(userHandler, userHandlerV2)
	userRouter = middleware.MiddleWareLogger(middleware.MiddleWareRecover(userRouter))
	healthRouter := router.NewHealthRouter(readiness)
	rootRouter := router.NewRoot(userRouter, healthRouter)

	server := api.NewAPIServer(cfg, rootRouter, readiness)

	return server.Run(ctx)
}
