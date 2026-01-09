package main

import (
	"context"
	"log"
	"os/signal"
	"pet-study/internal/httputils"
	"pet-study/internal/middleware"
	"syscall"

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

	readiness := health.NewReadiness()

	userHandler := routes.NewUserHandler(userService)
	userRouter := router.NewRouter(httputils.AppHandler(userHandler.Handle))
	userRouter = middleware.MiddleWareLogger(middleware.MiddleWareRecover(userRouter))
	healthRouter := router.NewHealthRouter(readiness)
	rootRouter := router.NewRoot(userRouter, healthRouter)

	server := api.NewAPIServer(":8080", rootRouter, readiness)

	return server.Run(ctx)
}
