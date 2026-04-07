package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"pet-study/internal/api"
	"pet-study/internal/config"
	"pet-study/internal/health"
	"pet-study/internal/httputils"
	"pet-study/internal/metrics"
	"pet-study/internal/middleware"
	"pet-study/internal/outbound"
	"pet-study/internal/outbound/httpclient"
	"pet-study/internal/queue"
	"pet-study/internal/requestid"
	apirouter "pet-study/internal/router"
	"pet-study/internal/routes"
	"pet-study/internal/security"
	"pet-study/internal/service"
	"pet-study/internal/store/jobrepo"
	"pet-study/internal/store/userrepo"
	"pet-study/internal/stream"
	"pet-study/internal/transport/grpcclient"
	"pet-study/internal/transport/grpcserver"
	"pet-study/internal/transport/pb"
	"pet-study/internal/workerpool"
	"syscall"

	"google.golang.org/grpc"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		logger.Error("application exited", "component", "main", "err", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.Default().With("component", "main")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	proxyAPI, err := middleware.NewProxyAPI(cfg.Proxy)
	if err != nil {
		return err
	}

	// Metrics registry
	m := metrics.DefaultHTTP()

	eventHub := stream.NewHub(cfg.Streaming.SubscriberBuffer)

	// Dependencies
	userRepository := userrepo.NewMemoryUserRepository()
	jobRepository := jobrepo.NewMemoryJobRepository()
	userService := service.NewUserService(userRepository)
	jobService := service.NewJobService(jobRepository)

	//Async
	q := queue.New(cfg.Pool.QueueSize)
	pool := workerpool.NewWorkerPool(q, jobService, userService, m, eventHub)
	poolErr := pool.Start(ctx, cfg.Pool.Workers)
	if poolErr != nil {
		return poolErr
	}

	//Client-Transport
	httpClient, tr := httpclient.New(cfg.Outbound)
	defer tr.CloseIdleConnections()

	rawProfileClient := outbound.NewClientImpl(cfg.Outbound.Profile.Base, httpClient)
	instrumented := outbound.NewInstrumentedProfileClient(cfg.Outbound.Profile.Base, rawProfileClient, logger)

	profileClient := outbound.NewRetryingProfileClient(
		cfg.Outbound.Retry.MaxAttempts,
		cfg.Outbound.Retry.BaseDelay,
		cfg.Outbound.Retry.MaxDelay,
		instrumented,
	)

	var (
		grpcRuntime    *grpcserver.Runtime
		jobsGRPCClient pb.JobsServiceClient
		grpcConn       *grpc.ClientConn
		checkSlice     []health.Check
	)

	if cfg.GRPC.Enable {
		grpcRuntime, err = grpcserver.NewRuntime(cfg.GRPC.Addr, jobService, logger)
		if err != nil {
			return err
		}
		grpcRuntime.Start(stop)

		jobsGRPCClient, grpcConn, err = grpcclient.NewJobsClient(cfg.GRPC.Addr)
		if err != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
			defer cancel()
			_ = grpcRuntime.Shutdown(shutdownCtx)
			return err
		}

		defer func() {
			if grpcConn != nil {
				if err := grpcConn.Close(); err != nil {
					logger.Warn("grpc client connection close", "err", err)
				}
			}
		}()
		checkSlice = append(checkSlice, health.Check{Name: "grpc", Fn: grpcRuntime.Ready})
	}

	checkSlice = append(checkSlice, health.Check{Name: "repo", Fn: userRepository.Ping})
	checkSlice = append(checkSlice, health.Check{Name: "workerpool", Fn: pool.CheckRunning})
	checkSlice = append(checkSlice, health.Check{Name: "streamHub", Fn: eventHub.Ready})
	readiness := health.NewReadiness(checkSlice...)

	profileService := service.NewUserProfileService(userService, profileClient, cfg.Outbound.Profile.Timeout)
	profileHandler := routes.NewUsersProfileHandler(profileService)

	limitedAPI := middleware.NewRateLimitedAPI(float64(cfg.Limiter.RPS), cfg.Limiter.Burst)
	bulkhead := middleware.NewBulkhead(cfg.Bulkhead.MaxParallel)

	// Routers
	userHandler := routes.NewUserHandler(userService, jobService, q, m, eventHub)
	userHandlerV2 := routes.NewUserV2Handler(userService, jobService, q, m, eventHub)
	jobsHandler := routes.NewJobHandler(
		jobService,
		eventHub,
		cfg.Streaming.SSEHeartbeat,
		cfg.Streaming.WriteTimeout,
		jobsGRPCClient,
	)

	keys := make([]security.HMACKey, 0, len(cfg.Auth.JWT.Keys))
	for _, k := range cfg.Auth.JWT.Keys {
		keys = append(keys, security.HMACKey{KID: k.KID, Secret: []byte(k.Secret)})
	}

	verifier, err := security.NewJWTVerifierHS256(
		cfg.Auth.JWT.AllowedAlg,
		cfg.Auth.JWT.Issuer,
		cfg.Auth.JWT.Audience,
		cfg.Auth.JWT.ClockSkew,
		keys,
	)
	if err != nil {
		return err
	}

	authAPI, err := middleware.NewAuthAPI(verifier)
	if err != nil {
		return err
	}

	rbacAPI, err := middleware.NewAuthorizeAPI(security.DefaultPolicy)
	if err != nil {
		return err
	}

	corsAPI := middleware.NewCORS(cfg.CORS)
	secAPI := middleware.NewSecurityHeaders(cfg.SecurityHeaders)

	userRouter := apirouter.NewRouter(
		userHandler,
		userHandlerV2,
		jobsHandler,
		profileHandler,
		limitedAPI,
		bulkhead,
		authAPI,
		rbacAPI,
	)
	userRouter = corsAPI.CORS(userRouter)
	userRouter = secAPI.SecurityHeaders(userRouter)

	var debugRouter http.Handler
	if cfg.HTTP.Debug {
		rawDebug := apirouter.NewDebugRouter()

		dbg := httputils.HandlerToApp(rawDebug)

		dbg = authAPI.Authenticate(dbg)
		dbg = rbacAPI.Authorize(dbg)

		debugRouter = dbg
	}

	healthRouter := apirouter.NewHealthRouter(readiness)
	rootRouter := apirouter.NewRoot(userRouter, healthRouter, debugRouter)

	// Middleware chain (execution order, outer -> inner):
	// 1) Proxy.SanitizeRequestIDHeader (trust-only X-Request-Id)
	// 2) RequestID
	// 3) Recover (outer)
	// 4) TrustProxy (RequestInfo: ClientIP/Scheme)
	// 5) Metrics
	// 6) Logger
	// 7) Recover (inner)
	// 8) RootRouter (ServeMux: API + health + debug)
	handler := rootRouter
	handler = middleware.Recover(handler) // inner: чтобы Logger/Metrics увидели 500 при panic в Router
	handler = middleware.Logger(handler)
	handler = middleware.Metrics(m)(handler)
	handler = proxyAPI.TrustProxy(handler)
	handler = middleware.Recover(handler) // outer: ловит panic в Logger/Metrics
	handler = requestid.RequestIDMiddleware(handler)
	handler = proxyAPI.SanitizeRequestIDHeader(handler)

	server := api.NewAPIServer(cfg, handler, readiness, pool, q, grpcRuntime, eventHub)
	logger.Info("application configured", "addr", cfg.HTTP.Addr, "debug", cfg.HTTP.Debug)

	return server.Run(ctx)
}
