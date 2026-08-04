package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"pet-study/internal/api"
	"pet-study/internal/config"
	"pet-study/internal/db"
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
	"pet-study/internal/telemetry"
	"pet-study/internal/transport/grpcclient"
	"pet-study/internal/transport/grpcserver"
	"pet-study/internal/transport/pb"
	"pet-study/internal/workerpool"
	"runtime/debug"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.Load()
	logger := fallbackLogger()
	if err == nil {
		logger, err = config.NewLogger(os.Stderr, cfg.Logging)
	}
	if err != nil {
		logger.Error("application exited", config.LogFieldComponent, config.LogComponentMain, config.LogFieldError, err)
		os.Exit(1)
	}
	slog.SetDefault(logger)

	if err := run(cfg, logger); err != nil {
		logger.With(config.LogFieldComponent, config.LogComponentMain).
			Error("application exited", config.LogFieldError, err)
		os.Exit(1)
	}
}

func fallbackLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func run(cfg config.Config, rootLogger *slog.Logger) error {
	logger := rootLogger.With(config.LogFieldComponent, config.LogComponentMain)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	telemetryLogger := rootLogger.With(config.LogFieldComponent, telemetry.LogComponent)
	telemetryRuntime, err := telemetry.NewFailOpen(ctx, cfg.Telemetry, telemetryBuildInfo(cfg), telemetryLogger)
	if err != nil {
		return err
	}
	telemetryRuntime.InstallGlobalsWithLogger(telemetryLogger)
	// Registered before business-component defers so telemetry is flushed after
	// DB, outbound HTTP and gRPC client cleanup have had a chance to emit final signals.
	defer shutdownTelemetry(telemetryRuntime, cfg.Telemetry.ShutdownTimeout, telemetryLogger)

	var (
		userRepository service.UserRepository
		jobRepository  service.JobRepository
		sqlDB          *db.DB
		repoReady      func(context.Context) error
	)

	proxyAPI, err := middleware.NewProxyAPI(cfg.Proxy)
	if err != nil {
		return err
	}

	// Metrics registry
	m := metrics.DefaultHTTP()

	eventHub := stream.NewHub(cfg.Streaming.SubscriberBuffer)

	switch cfg.DB.StorageBackend {
	case "memory":
		memUserRepo := userrepo.NewMemoryUserRepository()
		memJobRepo := jobrepo.NewMemoryJobRepository()

		userRepository = memUserRepo
		jobRepository = memJobRepo
		repoReady = memUserRepo.Ping

	case "postgres":
		sqlDB, err = db.Open(cfg.DB)
		if err != nil {
			return err
		}
		defer db.Close(sqlDB)

		db.PublishStats(sqlDB.DB)

		userRepository = userrepo.NewSQLX(sqlDB, cfg.DB.QueryTimeout)
		jobRepository = jobrepo.NewSQLX(sqlDB, cfg.DB.QueryTimeout)

		repoReady = func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, cfg.DB.PingTimeout)
			defer cancel()
			return sqlDB.PingContext(ctx)
		}

	default:
		return fmt.Errorf("unsupported storage backend: %s", cfg.DB.StorageBackend)
	}

	// Dependencies
	userService := service.NewUserService(userRepository)
	jobService := service.NewJobService(jobRepository)

	//Async
	q := queue.New(cfg.Pool.QueueSize)
	pool := workerpool.NewWorkerPool(q, jobService, userService, m, eventHub)

	//Client-Transport
	httpClient, tr := httpclient.New(cfg.Outbound)
	defer tr.CloseIdleConnections()

	rawProfileClient := outbound.NewClientImpl(cfg.Outbound.Profile.Base, httpClient)
	instrumented := outbound.NewInstrumentedProfileClient(
		cfg.Outbound.Profile.Base,
		rawProfileClient,
		rootLogger.With(config.LogFieldComponent, config.LogComponentOutboundProfile),
	)

	profileClient := outbound.NewRetryingProfileClient(
		cfg.Outbound.Retry.MaxAttempts,
		cfg.Outbound.Retry.BaseDelay,
		cfg.Outbound.Retry.MaxDelay,
		instrumented,
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

	var (
		grpcRuntime    *grpcserver.Runtime
		jobsGRPCClient pb.JobsServiceClient
		grpcConn       *grpc.ClientConn
		checkSlice     []health.Check
	)

	if cfg.GRPC.Enable {
		grpcLogger := rootLogger.With(config.LogFieldComponent, config.LogComponentGRPCServer)
		grpcRuntime, err = grpcserver.NewRuntimeWithConfig(grpcserver.Config{
			Addr:              cfg.GRPC.Addr,
			ReflectionEnabled: cfg.GRPC.ReflectionEnable,
			Auth: grpcserver.AuthConfig{
				Verifier: verifier,
			},
			TLS: grpcserver.TLSConfig{
				Enable:       cfg.GRPC.TLS.Enable,
				CertFile:     cfg.GRPC.TLS.CertFile,
				KeyFile:      cfg.GRPC.TLS.KeyFile,
				ClientCAFile: cfg.GRPC.TLS.ClientCAFile,
			},
		}, jobService, grpcLogger)
		if err != nil {
			return err
		}

		jobsGRPCClient, grpcConn, err = grpcclient.NewJobsClientWithConfig(grpcclient.Config{
			Addr: cfg.GRPC.Addr,
			TLS: grpcclient.TLSConfig{
				Enable:     cfg.GRPC.TLS.Enable,
				CertFile:   cfg.GRPC.TLS.ClientCertFile,
				KeyFile:    cfg.GRPC.TLS.ClientKeyFile,
				CAFile:     cfg.GRPC.TLS.ServerCAFile,
				ServerName: cfg.GRPC.TLS.ServerName,
			},
		})
		if err != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
			defer cancel()
			_ = grpcRuntime.Shutdown(shutdownCtx)
			return err
		}

		defer func() {
			if grpcConn != nil {
				if err := grpcConn.Close(); err != nil {
					logger.Warn("grpc client connection close", config.LogFieldError, err)
				}
			}
		}()
		checkSlice = append(checkSlice, health.Check{Name: "grpc", Fn: grpcRuntime.Ready})
	}
	checkSlice = append(checkSlice, health.Check{Name: "db", Fn: repoReady})
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

		dbg = rbacAPI.Authorize(dbg)
		dbg = authAPI.Authenticate(dbg)

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
	handler = middleware.RecoverWithLogger(
		handler,
		rootLogger.With(config.LogFieldComponent, config.LogComponentHTTPRecover),
	) // inner: чтобы Logger/Metrics увидели 500 при panic в Router
	handler = middleware.LoggerWithLogger(
		handler,
		rootLogger.With(config.LogFieldComponent, config.LogComponentHTTPAccess),
	)
	handler = middleware.Metrics(m)(handler)
	handler = proxyAPI.TrustProxy(handler)
	handler = middleware.RecoverWithLogger(
		handler,
		rootLogger.With(config.LogFieldComponent, config.LogComponentHTTPRecover),
	) // outer: ловит panic в Logger/Metrics
	handler = requestid.RequestIDMiddleware(handler)
	handler = proxyAPI.SanitizeRequestIDHeader(handler)

	server := api.NewAPIServer(cfg, handler, readiness, pool, q, grpcRuntime, eventHub)
	logger.Info("application configured", "addr", cfg.HTTP.Addr, "debug", cfg.HTTP.Debug)

	return server.Run(ctx)
}

func telemetryBuildInfo(cfg config.Config) telemetry.BuildInfo {
	version := telemetry.DefaultServiceVersion
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}

	return telemetry.BuildInfo{
		ServiceName:    telemetry.DefaultServiceName,
		ServiceVersion: version,
		Environment:    cfg.Runtime.Environment,
	}
}

func shutdownTelemetry(rt *telemetry.Runtime, timeout time.Duration, logger *slog.Logger) {
	if rt == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := rt.Shutdown(ctx); err != nil {
		logger.Warn("telemetry shutdown", config.LogFieldError, err)
	}
}
