package main

import (
	"context"
	"log"
	"os/signal"
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
		// log.Fatal уже завершает процесс через os.Exit(1)
		log.Fatal(err)
	}
}

// run — composition root: создаёт все зависимости, root-context
// и передаёт их вниз.
func run() error {
	// Root-контекст приложения с отменой по SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Сборка зависимостей (DI) ---

	userRepository := userrepo.NewMemoryUserRepository()
	userService := service.NewUserService(userRepository)
	v := validator.New()

	readiness := health.NewReadiness()

	userHandler := routes.NewUserHandler(userService, v)
	userRouter := router.NewRouter(userHandler)
	healthRouter := router.NewHealthRouter(readiness)
	rootRouter := router.NewRoot(userRouter, healthRouter)

	server := api.NewAPIServer(":8080", rootRouter, readiness)

	// ---- Запуск сервера ----

	return server.Run(ctx)
}
