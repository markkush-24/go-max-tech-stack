package main

import (
	"context"
	"log"
	"os/signal"
	"pet-study/internal/middleware"
	"syscall"

	"github.com/go-playground/validator/v10"

	"pet-study/internal/api"
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
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	userRepository := userrepo.NewMemoryUserRepository()
	userService := service.NewUserService(userRepository)
	v := validator.New()

	readiness := health.NewReadiness()

	userHandler := routes.NewUserHandler(userService, v)
	userRouter := router.NewRouter(userHandler)
	userRouter = middleware.MiddleWareLogger(middleware.MiddleWareRecover(userRouter))
	healthRouter := router.NewHealthRouter(readiness)
	rootRouter := router.NewRoot(userRouter, healthRouter)

	server := api.NewAPIServer(":8080", rootRouter, readiness)

	return server.Run(ctx)
}
