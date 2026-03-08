package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"pet-study/internal/httputils"
	"pet-study/internal/metrics"
	"pet-study/internal/middleware"
	"pet-study/internal/outbound"
	"pet-study/internal/outbound/httpclient"
	"pet-study/internal/queue"
	"pet-study/internal/security"
	"pet-study/internal/store/jobrepo"
	"syscall"

	"pet-study/internal/api"
	"pet-study/internal/config"
	"pet-study/internal/health"
	"pet-study/internal/requestid"
	apirouter "pet-study/internal/router"
	routes "pet-study/internal/routes"
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
	proxyAPI, err := middleware.NewProxyAPI(cfg.Proxy)
	if err != nil {
		return err
	}

	// Metrics registry
	m := metrics.DefaultHTTP()

	// Dependencies
	userRepository := userrepo.NewMemoryUserRepository()
	jobRepository := jobrepo.NewMemoryJobRepository()
	userService := service.NewUserService(userRepository)
	jobService := service.NewJobService(jobRepository)

	//Async
	q := queue.New(cfg.Pool.QueueSize)
	pool := workerpool.NewWorkerPool(q, jobService, userService, m)
	poolErr := pool.Start(cfg.Pool.Workers)
	if poolErr != nil {
		return poolErr
	}

	readiness := health.NewReadiness(
		health.Check{Name: "repo", Fn: userRepository.Ping},
		health.Check{Name: "workerpool", Fn: pool.CheckRunning})

	//Client-Transport
	httpClient, tr := httpclient.New(cfg.Outbound)
	defer tr.CloseIdleConnections()
	rawProfileClient := outbound.NewClientImpl(cfg.Outbound.Profile.Base, httpClient)
	instrumented := outbound.NewInstrumentedProfileClient(cfg.Outbound.Profile.Base, rawProfileClient, log.Default())

	profileClient := outbound.NewRetryingProfileClient(
		cfg.Outbound.Retry.MaxAttempts,
		cfg.Outbound.Retry.BaseDelay,
		cfg.Outbound.Retry.MaxDelay,
		instrumented,
	)

	profileService := service.NewUserProfileService(userService, profileClient, cfg.Outbound.Profile.Timeout)
	profileHandler := routes.NewUsersProfileHandler(profileService)

	limitedAPI := middleware.NewRateLimitedAPI(float64(cfg.Limiter.RPS), cfg.Limiter.Burst)
	bulkhead := middleware.NewBulkhead(cfg.Bulkhead.MaxParallel)

	// Routers
	userHandler := routes.NewUserHandler(userService, jobService, q, m)
	userHandlerV2 := routes.NewUserV2Handler(userService, jobService, q, m)

	jobsHandler := routes.NewJobHandler(jobService)

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

	authAPI := middleware.NewAuthAPI(verifier)
	rbacAPI := middleware.NewAuthorizeAPI(security.DefaultPolicy)

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

	server := api.NewAPIServer(cfg, handler, readiness, pool, q)
	log.Printf("config: addr=%s debug=%v", cfg.HTTP.Addr, cfg.HTTP.Debug)
	return server.Run(ctx)
}
