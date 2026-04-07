package testkit

import (
	"context"
	"mime"
	"net/http"
	"net/http/httptest"
	"pet-study/internal/config"
	"pet-study/internal/entity"
	"pet-study/internal/security"
	"pet-study/internal/stream"
	"testing"
	"time"

	"pet-study/internal/httpapi"
	"pet-study/internal/metrics"
	"pet-study/internal/middleware"
	"pet-study/internal/queue"
	"pet-study/internal/requestid"
	apirouter "pet-study/internal/router"
	routes "pet-study/internal/routes"
	"pet-study/internal/service"
	"pet-study/internal/store/jobrepo"
	"pet-study/internal/store/userrepo"
)

type App struct {
	UserRepo *userrepo.MemoryUserRepository
	JobRepo  *jobrepo.MemoryJobRepository

	UserSvc *service.UserService
	JobSvc  *service.JobService

	Q        *queue.Queue
	M        *metrics.HTTPMetrics
	EventHub *stream.Hub

	Limiter  *middleware.RateLimitedAPI
	Bulkhead *middleware.BulkheadAPI

	V1 httpapi.UsersAPI
	V2 httpapi.UsersAPI
	JH httpapi.JobsAPI
}
type StubVerifier struct {
	P     security.Principal
	Token string
}

type options struct {
	queueSize    int
	rps          float64
	burst        int
	bulkhead     int
	usersProfile httpapi.UsersProfileAPI
	health       http.Handler
	debug        http.Handler
	principal    security.Principal
	authToken    string
	policy       []security.RouteRule
	injectAuth   bool
	corsConfig   *config.CORSConfig
}

type Option func(*options)

func WithQueueSize(n int) Option {
	return func(o *options) { o.queueSize = n }
}
func WithRateLimit(rps float64, burst int) Option {
	return func(o *options) { o.rps, o.burst = rps, burst }
}
func WithBulkhead(maxParallel int) Option {
	return func(o *options) { o.bulkhead = maxParallel }
}
func WithUsersProfile(ph httpapi.UsersProfileAPI) Option {
	return func(o *options) { o.usersProfile = ph }
}
func WithHealth(h http.Handler) Option {
	return func(o *options) { o.health = h }
}
func WithDebug(h http.Handler) Option {
	return func(o *options) { o.debug = h }
}

func WithoutAuthInjection() Option {
	return func(o *options) {
		o.injectAuth = false
	}
}

func WithPrincipalUser(id int64) Option {
	return func(o *options) {
		o.principal = security.Principal{
			UserID: id,
			Role:   security.RoleUser,
		}
	}
}

func WithPrincipalAdmin(id int64) Option {
	return func(o *options) {
		o.principal = security.Principal{
			UserID: id,
			Role:   security.RoleAdmin,
		}
	}
}

func WithCORSAllowlist(origins ...string) Option {
	return func(o *options) {
		o.corsConfig = &config.CORSConfig{
			AllowedOrigins:   origins,
			AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
			AllowedHeaders:   []string{"Authorization", "Content-Type", "If-None-Match", "X-Request-Id"},
			AllowCredentials: false,
			MaxAge:           5 * time.Minute,
		}
	}
}

func defaultOptions() options {
	return options{
		queueSize: 10,
		// ВАЖНО: дефолт широкий, чтобы tests не флейкали на 429.
		rps:      1000,
		burst:    1000,
		bulkhead: 1000,

		health:     http.NewServeMux(),
		debug:      nil,
		principal:  security.Principal{UserID: 1, Role: security.RoleAdmin},
		injectAuth: true,
		authToken:  "test",
		policy:     security.DefaultPolicy,
		corsConfig: nil,
	}
}

func newApp(t *testing.T, opts ...Option) (*App, options) {
	t.Helper()

	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	userRepo := userrepo.NewMemoryUserRepository()
	jobRepo := jobrepo.NewMemoryJobRepository()

	userSvc := service.NewUserService(userRepo)
	jobSvc := service.NewJobService(jobRepo)

	q := queue.New(o.queueSize)
	m := metrics.DefaultHTTP()

	lim := middleware.NewRateLimitedAPI(o.rps, o.burst)
	bh := middleware.NewBulkhead(o.bulkhead)

	hub := stream.NewHub(16)
	v1 := routes.NewUserHandler(userSvc, jobSvc, q, m, hub)
	v2 := routes.NewUserV2Handler(userSvc, jobSvc, q, m, hub)
	eventHub := stream.NewHub(16)
	jh := routes.NewJobHandler(jobSvc, eventHub, 5*time.Second, 0*time.Second, nil)

	app := &App{
		UserRepo: userRepo,
		JobRepo:  jobRepo,
		UserSvc:  userSvc,
		JobSvc:   jobSvc,
		Q:        q,
		M:        m,
		EventHub: eventHub,
		Limiter:  lim,
		Bulkhead: bh,
		V1:       v1,
		V2:       v2,
		JH:       jh,
	}
	return app, o
}

// NewUserRouter — для tests через httptest.NewRecorder
func NewUserRouter(t *testing.T, opts ...Option) (http.Handler, *App) {
	t.Helper()

	app, o := newApp(t, opts...)
	ver := StubVerifier{P: o.principal, Token: o.authToken}
	auth, err := middleware.NewAuthAPI(ver)
	if err != nil {
		t.Fatalf("NewAuthAPI: %v", err)
	}
	rbac, err := middleware.NewAuthorizeAPI(o.policy)
	if err != nil {
		t.Fatalf("NewAuthorizeAPI: %v", err)
	}
	userRouter := apirouter.NewRouter(app.V1, app.V2, app.JH, o.usersProfile, app.Limiter, app.Bulkhead, auth, rbac)
	if o.corsConfig != nil {
		corsAPI := middleware.NewCORS(*o.corsConfig)
		userRouter = corsAPI.CORS(userRouter)
	}

	secAPI := middleware.NewSecurityHeaders(config.SecurityHeadersConfig{
		Enable:         true,
		ReferrerPolicy: "no-referrer",
	})

	var h http.Handler = userRouter
	h = secAPI.SecurityHeaders(h)
	h = requestid.RequestIDMiddleware(h)

	if o.injectAuth && o.authToken != "" {
		h = injectBearer(o.authToken, h)
	}
	return h, app
}

// NewServer — полный стек: root router + middleware chain + httptest.Server
func NewServer(t *testing.T, opts ...Option) (*httptest.Server, *App) {
	t.Helper()

	app, o := newApp(t, opts...)
	ver := StubVerifier{P: o.principal, Token: o.authToken}
	auth, err := middleware.NewAuthAPI(ver)
	if err != nil {
		t.Fatalf("NewAuthAPI: %v", err)
	}
	rbac, err := middleware.NewAuthorizeAPI(o.policy)
	if err != nil {
		t.Fatalf("NewAuthorizeAPI: %v", err)
	}
	userRouter := apirouter.NewRouter(app.V1, app.V2, app.JH, o.usersProfile, app.Limiter, app.Bulkhead, auth, rbac)

	if o.corsConfig != nil {
		corsAPI := middleware.NewCORS(*o.corsConfig)
		userRouter = corsAPI.CORS(userRouter)
	}

	secAPI := middleware.NewSecurityHeaders(config.SecurityHeadersConfig{
		Enable:         true,
		ReferrerPolicy: "no-referrer",
	})
	userRouter = secAPI.SecurityHeaders(userRouter)

	root := apirouter.NewRoot(userRouter, o.health, o.debug)

	// Minimal test chain (differs from main: no proxy trust/sanitize)
	var h http.Handler = root
	h = middleware.Recover(h)
	h = middleware.Logger(h)
	h = middleware.Metrics(app.M)(h)
	h = middleware.Recover(h)
	h = requestid.RequestIDMiddleware(h)

	if o.injectAuth && o.authToken != "" {
		h = injectBearer(o.authToken, h)
	}

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	return srv, app
}

func (v StubVerifier) Verify(token string) (security.Principal, error) {
	if v.Token != "" && token != v.Token {
		return security.Principal{}, &security.AuthNError{Kind: security.AuthNInvalid}
	}
	return v.P, nil
}

func injectBearer(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" && token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		next.ServeHTTP(w, r)
	})
}

func MustMediaType(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	mt, _, err := mime.ParseMediaType(rec.Header().Get("Content-Type"))
	if err != nil {
		t.Fatalf("bad Content-Type: %v", err)
	}
	return mt
}

func ReqContext() context.Context { return context.Background() }

func CreateUser(name, email string, age int) *entity.CreateUserInput {
	return &entity.CreateUserInput{Name: name, Email: email, Age: age}
}
func InjectBearerForTests(token string, next http.Handler) http.Handler {
	return injectBearer(token, next)
}

func NewMetricsForTests() *metrics.HTTPMetrics {
	return metrics.DefaultHTTP()
}
