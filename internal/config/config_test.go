package config

import (
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

type envKV struct {
	key   string
	value string
}

var validConfigEnv = []envKV{
	{"HTTP_ADDR", ":18080"},
	{"HTTP_TLS_ENABLE", "false"},
	{"HTTP_TLS_ADDR", ":18443"},
	{"HTTP_TLS_CERT_FILE", "certs/localhost.pem"},
	{"HTTP_TLS_KEY_FILE", "certs/localhost-key.pem"},
	{"GRPC_ENABLE", "false"},
	{"GRPC_ADDR", ":19090"},
	{"STREAMING_SSE_HEARTBEAT", "20s"},
	{"STREAMING_SUBSCRIBER_BUFFER", "32"},
	{"STREAMING_WRITE_TIMEOUT", "11s"},
	{"HTTP_READ_HEADER_TIMEOUT", "6s"},
	{"HTTP_READ_TIMEOUT", "12s"},
	{"HTTP_WRITE_TIMEOUT", "16s"},
	{"HTTP_IDLE_TIMEOUT", "130s"},
	{"HTTP_SHUTDOWN_TIMEOUT", "12s"},
	{"HTTP_DEBUG", "true"},
	{"HTTP_MAX_HEADER_BYTES", "2048"},
	{"WORKERS_COUNT", "7"},
	{"QUEUE_SIZE", "17"},
	{"RATE_LIMIT_RPS", "11"},
	{"RATE_LIMIT_BURST", "22"},
	{"BULKHEAD_MAX_PARALLEL", "3"},
	{"DB_DSN", "postgres://test:test@localhost:5433/test?sslmode=disable"},
	{"STORAGE_BACKEND", "postgres"},
	{"DB_MAX_OPEN_CONNS", "15"},
	{"DB_MAX_IDLE_CONNS", "8"},
	{"DB_CONN_MAX_LIFETIME", "40m"},
	{"DB_CONN_MAX_IDLE_TIME", "6m"},
	{"DB_QUERY_TIMEOUT", "4s"},
	{"DB_PING_TIMEOUT", "2s"},
	{"OUTBOUND_TRANSPORT_IDLE_CONN_TIMEOUT", "91s"},
	{"OUTBOUND_TRANSPORT_MAX_IDLE_CONNS", "101"},
	{"OUTBOUND_TRANSPORT_MAX_IDLE_CONNS_PER_HOST", "51"},
	{"OUTBOUND_TRANSPORT_MAX_CONNS_PER_HOST", "9"},
	{"OUTBOUND_TRANSPORT_TLS_HANDSHAKE_TIMEOUT", "6s"},
	{"OUTBOUND_TRANSPORT_RESPONSE_HEADER_TIMEOUT", "2s"},
	{"OUTBOUND_RETRY_MAX_ATTEMPTS", "2"},
	{"OUTBOUND_RETRY_BASE_DELAY", "60ms"},
	{"OUTBOUND_RETRY_MAX_DELAY", "600ms"},
	{"OUTBOUND_PROFILE_BASE_URL", "https://profile.example.test/base/"},
	{"OUTBOUND_PROFILE_TIMEOUT", "2s"},
	{"AUTH_JWT_ALLOWED_ALG", "HS256"},
	{"AUTH_JWT_CLOCK_SKEW", "45s"},
	{"AUTH_JWT_ISSUER", "issuer"},
	{"AUTH_JWT_AUDIENCE", "audience"},
	{"AUTH_JWT_KEYS", "dev:dev-secret,next:next-secret"},
	{"CORS_ALLOWED_ORIGINS", "HTTP://LOCALHOST:3000,https://example.test"},
	{"CORS_ALLOWED_METHODS", "get,post,options"},
	{"CORS_ALLOWED_HEADERS", "authorization,content-type,x-request-id"},
	{"CORS_ALLOW_CREDENTIALS", "false"},
	{"CORS_MAX_AGE", "10m"},
	{"PROXY_TRUSTED_PROXIES", "127.0.0.1/32,::1/128"},
	{"PROXY_TRUST_XFF", "true"},
	{"PROXY_TRUST_XFP", "true"},
	{"SECURITY_HEADERS_ENABLE", "true"},
	{"SECURITY_HEADERS_REFERRER_POLICY", "same-origin"},
	{"SECURITY_HEADERS_HSTS_ENABLE", "true"},
	{"SECURITY_HEADERS_HSTS_MAX_AGE", "1h"},
}

func TestLoad_Defaults(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("HTTP.Addr=%q want=:8080", cfg.HTTP.Addr)
	}
	if cfg.HTTP.ReadHeaderTimeout != 5*time.Second ||
		cfg.HTTP.ReadTimeout != 10*time.Second ||
		cfg.HTTP.WriteTimeout != 15*time.Second ||
		cfg.HTTP.IdleTimeout != 120*time.Second ||
		cfg.HTTP.ShutdownTimeout != 10*time.Second {
		t.Fatalf("HTTP timeouts = %+v", cfg.HTTP)
	}
	if cfg.HTTP.MaxHeaderBytes != http.DefaultMaxHeaderBytes {
		t.Fatalf("HTTP.MaxHeaderBytes=%d want=%d", cfg.HTTP.MaxHeaderBytes, http.DefaultMaxHeaderBytes)
	}
	if cfg.HTTP.Debug {
		t.Fatalf("HTTP.Debug=true want=false")
	}
	if cfg.HTTP.TLS.Enable || cfg.HTTP.TLS.Addr != ":8443" || cfg.HTTP.TLS.CertFile != "" || cfg.HTTP.TLS.KeyFile != "" {
		t.Fatalf("HTTP.TLS=%+v", cfg.HTTP.TLS)
	}
	if cfg.GRPC.Enable || cfg.GRPC.Addr != ":9090" {
		t.Fatalf("GRPC=%+v", cfg.GRPC)
	}
	if cfg.DB.StorageBackend != "postgres" {
		t.Fatalf("DB.StorageBackend=%q want=%q", cfg.DB.StorageBackend, "postgres")
	}
	const wantDSN = "postgres://petstudy:petstudy@localhost:5432/petstudy?sslmode=disable"
	if cfg.DB.DSN != wantDSN {
		t.Fatalf("DB.DSN=%q want=%q", cfg.DB.DSN, wantDSN)
	}
	if cfg.DB.MaxOpenConns != 10 ||
		cfg.DB.MaxIdleConns != 10 ||
		cfg.DB.ConnMaxLifetime != 30*time.Minute ||
		cfg.DB.ConnMaxIdleTime != 5*time.Minute ||
		cfg.DB.QueryTimeout != 3*time.Second ||
		cfg.DB.PingTimeout != time.Second {
		t.Fatalf("DB defaults=%+v", cfg.DB)
	}
	if cfg.Pool.Workers != 10 || cfg.Pool.QueueSize != 10 {
		t.Fatalf("Pool=%+v", cfg.Pool)
	}
	if cfg.Limiter.RPS != 5 || cfg.Limiter.Burst != 10 {
		t.Fatalf("Limiter=%+v", cfg.Limiter)
	}
	if cfg.Bulkhead.MaxParallel != 1 {
		t.Fatalf("Bulkhead=%+v", cfg.Bulkhead)
	}
	if cfg.Outbound.Profile.Base.String() != "http://localhost:8090" || cfg.Outbound.Profile.Timeout != time.Second {
		t.Fatalf("Outbound.Profile=%+v", cfg.Outbound.Profile)
	}
	if cfg.Outbound.Transport.IdleConnTimeout != 90*time.Second ||
		cfg.Outbound.Transport.MaxIdleConns != 1000 ||
		cfg.Outbound.Transport.MaxIdleConnsPerHost != 1000 ||
		cfg.Outbound.Transport.MaxConnsPerHost != 0 ||
		cfg.Outbound.Transport.TLSHandshakeTimeout != 5*time.Second ||
		cfg.Outbound.Transport.ResponseHeaderTimeout != time.Second {
		t.Fatalf("Outbound.Transport=%+v", cfg.Outbound.Transport)
	}
	if cfg.Outbound.Retry.MaxAttempts != 1 ||
		cfg.Outbound.Retry.BaseDelay != 50*time.Millisecond ||
		cfg.Outbound.Retry.MaxDelay != 500*time.Millisecond {
		t.Fatalf("Outbound.Retry=%+v", cfg.Outbound.Retry)
	}
	if cfg.Auth.JWT.AllowedAlg != "HS256" ||
		cfg.Auth.JWT.Issuer != "" ||
		cfg.Auth.JWT.Audience != "" ||
		cfg.Auth.JWT.ClockSkew != 30*time.Second ||
		!reflect.DeepEqual(cfg.Auth.JWT.Keys, []JWTKey{{KID: "dev", Secret: "dev-secret"}}) {
		t.Fatalf("Auth.JWT=%+v", cfg.Auth.JWT)
	}
	if cfg.CORS.AllowedOrigins != nil ||
		!reflect.DeepEqual(cfg.CORS.AllowedMethods, []string{"GET", "POST", "OPTIONS"}) ||
		!reflect.DeepEqual(cfg.CORS.AllowedHeaders, []string{"Authorization", "Content-Type", "If-None-Match", "X-Request-Id"}) ||
		cfg.CORS.AllowCredentials ||
		cfg.CORS.MaxAge != 5*time.Minute {
		t.Fatalf("CORS=%+v", cfg.CORS)
	}
	if len(cfg.Proxy.TrustedProxies) != 0 || cfg.Proxy.TrustXFF || cfg.Proxy.TrustXFP {
		t.Fatalf("Proxy=%+v", cfg.Proxy)
	}
	if !cfg.SecurityHeaders.Enable ||
		cfg.SecurityHeaders.ReferrerPolicy != "no-referrer" ||
		cfg.SecurityHeaders.HSTS.Enable ||
		cfg.SecurityHeaders.HSTS.MaxAge != 0 {
		t.Fatalf("SecurityHeaders=%+v", cfg.SecurityHeaders)
	}
	if cfg.Streaming.SSEHeartbeat != 15*time.Second ||
		cfg.Streaming.SubscriberBuffer != 16 ||
		cfg.Streaming.WriteTimeout != 10*time.Second {
		t.Fatalf("Streaming=%+v", cfg.Streaming)
	}
}

func TestLoad_ExplicitValidEnvironment(t *testing.T) {
	setValidConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTP.Addr != ":18080" || !cfg.HTTP.Debug || cfg.HTTP.MaxHeaderBytes != 2048 {
		t.Fatalf("HTTP=%+v", cfg.HTTP)
	}
	if cfg.Streaming.SSEHeartbeat != 20*time.Second ||
		cfg.Streaming.SubscriberBuffer != 32 ||
		cfg.Streaming.WriteTimeout != 11*time.Second {
		t.Fatalf("Streaming=%+v", cfg.Streaming)
	}
	if cfg.DB.DSN != "postgres://test:test@localhost:5433/test?sslmode=disable" ||
		cfg.DB.StorageBackend != "postgres" ||
		cfg.DB.MaxOpenConns != 15 ||
		cfg.DB.MaxIdleConns != 8 ||
		cfg.DB.ConnMaxLifetime != 40*time.Minute ||
		cfg.DB.ConnMaxIdleTime != 6*time.Minute ||
		cfg.DB.QueryTimeout != 4*time.Second ||
		cfg.DB.PingTimeout != 2*time.Second {
		t.Fatalf("DB=%+v", cfg.DB)
	}
	if cfg.Outbound.Profile.Base.String() != "https://profile.example.test/base" ||
		cfg.Outbound.Profile.Timeout != 2*time.Second {
		t.Fatalf("Outbound.Profile=%+v", cfg.Outbound.Profile)
	}
	if cfg.Auth.JWT.Issuer != "issuer" ||
		cfg.Auth.JWT.Audience != "audience" ||
		cfg.Auth.JWT.ClockSkew != 45*time.Second ||
		!reflect.DeepEqual(cfg.Auth.JWT.Keys, []JWTKey{
			{KID: "dev", Secret: "dev-secret"},
			{KID: "next", Secret: "next-secret"},
		}) {
		t.Fatalf("Auth.JWT=%+v", cfg.Auth.JWT)
	}
	if !reflect.DeepEqual(cfg.CORS.AllowedOrigins, []string{"http://localhost:3000", "https://example.test"}) ||
		!reflect.DeepEqual(cfg.CORS.AllowedMethods, []string{"GET", "POST", "OPTIONS"}) ||
		!reflect.DeepEqual(cfg.CORS.AllowedHeaders, []string{"Authorization", "Content-Type", "X-Request-Id"}) {
		t.Fatalf("CORS=%+v", cfg.CORS)
	}
	if len(cfg.Proxy.TrustedProxies) != 2 || !cfg.Proxy.TrustXFF || !cfg.Proxy.TrustXFP {
		t.Fatalf("Proxy=%+v", cfg.Proxy)
	}
	if !cfg.SecurityHeaders.HSTS.Enable || cfg.SecurityHeaders.HSTS.MaxAge != time.Hour {
		t.Fatalf("SecurityHeaders=%+v", cfg.SecurityHeaders)
	}
}

func TestLoad_OptionalTransports(t *testing.T) {
	tests := []struct {
		name   string
		env    map[string]string
		assert func(t *testing.T, cfg Config)
	}{
		{
			name: "TLS enabled",
			env: map[string]string{
				"HTTP_TLS_ENABLE":    "true",
				"HTTP_TLS_ADDR":      ":18443",
				"HTTP_TLS_CERT_FILE": "certs/localhost.pem",
				"HTTP_TLS_KEY_FILE":  "certs/localhost-key.pem",
			},
			assert: func(t *testing.T, cfg Config) {
				t.Helper()
				if !cfg.HTTP.TLS.Enable ||
					cfg.HTTP.TLS.Addr != ":18443" ||
					cfg.HTTP.TLS.CertFile != "certs/localhost.pem" ||
					cfg.HTTP.TLS.KeyFile != "certs/localhost-key.pem" {
					t.Fatalf("HTTP.TLS=%+v", cfg.HTTP.TLS)
				}
			},
		},
		{
			name: "gRPC enabled",
			env: map[string]string{
				"GRPC_ENABLE": "true",
				"GRPC_ADDR":   ":19090",
			},
			assert: func(t *testing.T, cfg Config) {
				t.Helper()
				if !cfg.GRPC.Enable || cfg.GRPC.Addr != ":19090" {
					t.Fatalf("GRPC=%+v", cfg.GRPC)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidConfigEnv(t)
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			tt.assert(t, cfg)
		})
	}
}

func TestLoad_InvalidEnvironment(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{
			name:    "invalid bool",
			env:     map[string]string{"HTTP_DEBUG": "not-bool"},
			wantErr: "HTTP_DEBUG",
		},
		{
			name:    "TLS enabled missing cert",
			env:     map[string]string{"HTTP_TLS_ENABLE": "true", "HTTP_TLS_CERT_FILE": ""},
			wantErr: "HTTP_TLS_CERT_FILE",
		},
		{
			name:    "gRPC enabled empty address",
			env:     map[string]string{"GRPC_ENABLE": "true", "GRPC_ADDR": ""},
			wantErr: "GRPC_ADDR",
		},
		{
			name:    "invalid positive duration",
			env:     map[string]string{"HTTP_READ_TIMEOUT": "not-duration"},
			wantErr: "HTTP_READ_TIMEOUT",
		},
		{
			name:    "zero positive duration",
			env:     map[string]string{"DB_QUERY_TIMEOUT": "0s"},
			wantErr: "DB_QUERY_TIMEOUT",
		},
		{
			name:    "negative nonnegative duration",
			env:     map[string]string{"AUTH_JWT_CLOCK_SKEW": "-1s"},
			wantErr: "AUTH_JWT_CLOCK_SKEW",
		},
		{
			name:    "invalid positive integer",
			env:     map[string]string{"WORKERS_COUNT": "not-int"},
			wantErr: "WORKERS_COUNT",
		},
		{
			name:    "zero positive integer",
			env:     map[string]string{"QUEUE_SIZE": "0"},
			wantErr: "QUEUE_SIZE",
		},
		{
			name:    "negative nonnegative integer",
			env:     map[string]string{"DB_MAX_OPEN_CONNS": "-1"},
			wantErr: "DB_MAX_OPEN_CONNS",
		},
		{
			name:    "empty DB DSN",
			env:     map[string]string{"DB_DSN": ""},
			wantErr: "DB_DSN",
		},
		{
			name:    "empty storage backend",
			env:     map[string]string{"STORAGE_BACKEND": ""},
			wantErr: "STORAGE_BACKEND",
		},
		{
			name:    "invalid outbound URL scheme",
			env:     map[string]string{"OUTBOUND_PROFILE_BASE_URL": "ftp://profile.example.test"},
			wantErr: "OUTBOUND_PROFILE_BASE_URL",
		},
		{
			name:    "invalid outbound URL host",
			env:     map[string]string{"OUTBOUND_PROFILE_BASE_URL": "http:///profiles"},
			wantErr: "OUTBOUND_PROFILE_BASE_URL",
		},
		{
			name:    "invalid outbound URL fragment",
			env:     map[string]string{"OUTBOUND_PROFILE_BASE_URL": "https://profile.example.test/#fragment"},
			wantErr: "OUTBOUND_PROFILE_BASE_URL",
		},
		{
			name:    "unsupported JWT alg",
			env:     map[string]string{"AUTH_JWT_ALLOWED_ALG": "RS256"},
			wantErr: "AUTH_JWT_ALLOWED_ALG",
		},
		{
			name:    "invalid JWT key format",
			env:     map[string]string{"AUTH_JWT_KEYS": "dev"},
			wantErr: "AUTH_JWT_KEYS",
		},
		{
			name:    "empty JWT key secret",
			env:     map[string]string{"AUTH_JWT_KEYS": "dev:   "},
			wantErr: "AUTH_JWT_KEYS",
		},
		{
			name:    "duplicate JWT key id",
			env:     map[string]string{"AUTH_JWT_KEYS": "dev:one,dev:two"},
			wantErr: "AUTH_JWT_KEYS",
		},
		{
			name:    "empty CORS origins",
			env:     map[string]string{"CORS_ALLOWED_ORIGINS": ""},
			wantErr: "CORS_ALLOWED_ORIGINS",
		},
		{
			name:    "invalid CORS origin path",
			env:     map[string]string{"CORS_ALLOWED_ORIGINS": "https://example.test/path"},
			wantErr: "CORS_ALLOWED_ORIGINS",
		},
		{
			name: "CORS credentials with wildcard",
			env: map[string]string{
				"CORS_ALLOWED_ORIGINS":   "*",
				"CORS_ALLOW_CREDENTIALS": "true",
			},
			wantErr: "CORS_ALLOW_CREDENTIALS",
		},
		{
			name:    "invalid CORS method",
			env:     map[string]string{"CORS_ALLOWED_METHODS": "GET,POST BAD"},
			wantErr: "CORS_ALLOWED_METHODS",
		},
		{
			name:    "invalid CORS header",
			env:     map[string]string{"CORS_ALLOWED_HEADERS": "X Bad"},
			wantErr: "CORS_ALLOWED_HEADERS",
		},
		{
			name:    "invalid trusted proxy",
			env:     map[string]string{"PROXY_TRUSTED_PROXIES": "not-an-ip"},
			wantErr: "PROXY_TRUSTED_PROXIES",
		},
		{
			name:    "empty trusted proxy",
			env:     map[string]string{"PROXY_TRUSTED_PROXIES": ""},
			wantErr: "PROXY_TRUSTED_PROXIES",
		},
		{
			name: "HSTS enabled without max age",
			env: map[string]string{
				"SECURITY_HEADERS_HSTS_ENABLE":  "true",
				"SECURITY_HEADERS_HSTS_MAX_AGE": "0s",
			},
			wantErr: "SECURITY_HEADERS_HSTS_ENABLE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidConfigEnv(t)
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			_, err := Load()
			if err == nil {
				t.Fatal("Load() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error=%q want mention of %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_HostileUnrelatedEnvironment(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "://bad")
	t.Setenv("OTEL_SERVICE_NAME", "")
	t.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "not-duration")
	t.Setenv("GOGC", "not-a-percent")
	t.Setenv("GOMEMLIMIT", "not-a-size")
	t.Setenv("GODEBUG", "unknown=bad")

	_, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func setValidConfigEnv(t *testing.T) {
	t.Helper()

	for _, kv := range validConfigEnv {
		t.Setenv(kv.key, kv.value)
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()

	type oldValue struct {
		value string
		ok    bool
	}

	old := make(map[string]oldValue, len(validConfigEnv))
	for _, kv := range validConfigEnv {
		v, ok := os.LookupEnv(kv.key)
		old[kv.key] = oldValue{value: v, ok: ok}
		if err := os.Unsetenv(kv.key); err != nil {
			t.Fatalf("unset %s: %v", kv.key, err)
		}
	}

	t.Cleanup(func() {
		for _, kv := range validConfigEnv {
			if previous, ok := old[kv.key]; ok && previous.ok {
				_ = os.Setenv(kv.key, previous.value)
			} else {
				_ = os.Unsetenv(kv.key)
			}
		}
	})
}
