package config

import (
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type HTTPConfig struct {
	Addr              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	MaxHeaderBytes    int
	Debug             bool
}

type DBConfig struct {
	DSN string
}

type WorkerPoolConfig struct {
	Workers   int
	QueueSize int
}

type RateLimiterConfig struct {
	RPS   int
	Burst int
}

type BulkheadConfig struct {
	MaxParallel int
}

type OutboundTransportConfig struct {
	IdleConnTimeout       time.Duration
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	MaxConnsPerHost       int
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
}

type OutboundRetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

type OutboundProfileConfig struct {
	Base    *url.URL
	Timeout time.Duration
}

type OutboundConfig struct {
	Profile   OutboundProfileConfig
	Transport OutboundTransportConfig
	Retry     OutboundRetryConfig
}

type AuthConfig struct {
	JWT JWTConfig
}

type JWTKey struct {
	KID    string
	Secret string
}

type JWTConfig struct {
	AllowedAlg string
	Issuer     string
	Audience   string
	ClockSkew  time.Duration
	Keys       []JWTKey
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

type ProxyConfig struct {
	TrustedProxies []netip.Prefix
	TrustXFF       bool
	TrustXFP       bool
}

type HSTSConfig struct {
	Enable bool
	MaxAge time.Duration
}

type SecurityHeadersConfig struct {
	Enable         bool
	ReferrerPolicy string
	FrameMode      string
	HSTS           HSTSConfig
}

type Config struct {
	HTTP            HTTPConfig
	DB              DBConfig
	Pool            WorkerPoolConfig
	Limiter         RateLimiterConfig
	Bulkhead        BulkheadConfig
	Outbound        OutboundConfig
	Auth            AuthConfig
	CORS            CORSConfig
	Proxy           ProxyConfig
	SecurityHeaders SecurityHeadersConfig
}

func defaultConfig() Config {
	return Config{
		HTTP: HTTPConfig{
			Addr:              ":8080",
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       120 * time.Second,
			ShutdownTimeout:   10 * time.Second,
			MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
			Debug:             false,
		},
		DB: DBConfig{
			DSN: "",
		},
		Pool: WorkerPoolConfig{
			Workers:   10,
			QueueSize: 10,
		},
		Limiter: RateLimiterConfig{
			RPS:   5,
			Burst: 10,
		},
		Bulkhead: BulkheadConfig{
			MaxParallel: 1,
		},
		Outbound: OutboundConfig{
			Profile: OutboundProfileConfig{
				Base:    &url.URL{Scheme: "http", Host: "localhost:8090"},
				Timeout: 1 * time.Second,
			},
			Transport: OutboundTransportConfig{
				IdleConnTimeout:       90 * time.Second,
				MaxIdleConns:          1000,
				MaxIdleConnsPerHost:   1000,
				MaxConnsPerHost:       0,
				TLSHandshakeTimeout:   5 * time.Second,
				ResponseHeaderTimeout: 1 * time.Second,
			},
			Retry: OutboundRetryConfig{
				MaxAttempts: 1,
				BaseDelay:   50 * time.Millisecond,
				MaxDelay:    500 * time.Millisecond,
			},
		},
		Auth: AuthConfig{
			JWT: JWTConfig{
				AllowedAlg: "HS256",
				Issuer:     "", // пусто = проверку issuer можно будет отключать в middleware
				Audience:   "", // пусто = проверку audience можно будет отключать в middleware
				ClockSkew:  30 * time.Second,
				Keys: []JWTKey{
					{KID: "dev", Secret: "dev-secret"},
				},
			},
		},

		CORS: CORSConfig{
			AllowedOrigins:   nil,
			AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
			AllowedHeaders:   []string{"Authorization", "Content-Type", "If-None-Match", "X-Request-Id"},
			AllowCredentials: false,
			MaxAge:           5 * time.Minute,
		},

		Proxy: ProxyConfig{
			TrustedProxies: nil,
			TrustXFF:       false,
			TrustXFP:       false,
		},

		SecurityHeaders: SecurityHeadersConfig{
			Enable:         true,
			ReferrerPolicy: "no-referrer",
			FrameMode:      "xfo_deny",
			HSTS: HSTSConfig{
				Enable: false,
				MaxAge: 0,
			},
		},
	}
}

func Load() (Config, error) {
	cfg := defaultConfig()

	var err error

	cfg.HTTP.Addr, err = lookupStringNonEmpty("HTTP_ADDR", cfg.HTTP.Addr)
	if err != nil {
		return Config{}, err
	}

	cfg.HTTP.ReadHeaderTimeout, err = lookupDurationPositive("HTTP_READ_HEADER_TIMEOUT", cfg.HTTP.ReadHeaderTimeout)
	if err != nil {
		return Config{}, err
	}
	cfg.HTTP.ReadTimeout, err = lookupDurationPositive("HTTP_READ_TIMEOUT", cfg.HTTP.ReadTimeout)
	if err != nil {
		return Config{}, err
	}
	cfg.HTTP.WriteTimeout, err = lookupDurationPositive("HTTP_WRITE_TIMEOUT", cfg.HTTP.WriteTimeout)
	if err != nil {
		return Config{}, err
	}
	cfg.HTTP.IdleTimeout, err = lookupDurationPositive("HTTP_IDLE_TIMEOUT", cfg.HTTP.IdleTimeout)
	if err != nil {
		return Config{}, err
	}
	cfg.HTTP.ShutdownTimeout, err = lookupDurationPositive("HTTP_SHUTDOWN_TIMEOUT", cfg.HTTP.ShutdownTimeout)
	if err != nil {
		return Config{}, err
	}

	cfg.HTTP.Debug, err = lookupBool("HTTP_DEBUG", cfg.HTTP.Debug)
	if err != nil {
		return Config{}, err
	}

	cfg.HTTP.MaxHeaderBytes, err = lookupIntPositive("HTTP_MAX_HEADER_BYTES", cfg.HTTP.MaxHeaderBytes)
	if err != nil {
		return Config{}, err
	}

	cfg.Pool.Workers, err = lookupIntPositive("WORKERS_COUNT", cfg.Pool.Workers)
	if err != nil {
		return Config{}, err
	}

	cfg.Pool.QueueSize, err = lookupIntPositive("QUEUE_SIZE", cfg.Pool.QueueSize)
	if err != nil {
		return Config{}, err
	}

	cfg.Limiter.RPS, err = lookupIntPositive("RATE_LIMIT_RPS", cfg.Limiter.RPS)
	if err != nil {
		return Config{}, err
	}

	cfg.Limiter.Burst, err = lookupIntPositive("RATE_LIMIT_BURST", cfg.Limiter.Burst)
	if err != nil {
		return Config{}, err
	}

	cfg.Bulkhead.MaxParallel, err = lookupIntPositive("BULKHEAD_MAX_PARALLEL", cfg.Bulkhead.MaxParallel)
	if err != nil {
		return Config{}, err
	}

	cfg.DB.DSN, err = lookupStringNonEmpty("DB_DSN", cfg.DB.DSN)
	if err != nil {
		return Config{}, err
	}

	cfg.Outbound.Transport.IdleConnTimeout, err = lookupDurationPositive(
		"OUTBOUND_TRANSPORT_IDLE_CONN_TIMEOUT", cfg.Outbound.Transport.IdleConnTimeout)
	if err != nil {
		return Config{}, err
	}

	cfg.Outbound.Transport.MaxIdleConns, err = lookupIntNonNegative(
		"OUTBOUND_TRANSPORT_MAX_IDLE_CONNS", cfg.Outbound.Transport.MaxIdleConns)
	if err != nil {
		return Config{}, err
	}

	cfg.Outbound.Transport.MaxIdleConnsPerHost, err = lookupIntPositive(
		"OUTBOUND_TRANSPORT_MAX_IDLE_CONNS_PER_HOST", cfg.Outbound.Transport.MaxIdleConnsPerHost)
	if err != nil {
		return Config{}, err
	}

	cfg.Outbound.Transport.MaxConnsPerHost, err = lookupIntNonNegative(
		"OUTBOUND_TRANSPORT_MAX_CONNS_PER_HOST", cfg.Outbound.Transport.MaxConnsPerHost)
	if err != nil {
		return Config{}, err
	}

	cfg.Outbound.Transport.TLSHandshakeTimeout, err = lookupDurationPositive(
		"OUTBOUND_TRANSPORT_TLS_HANDSHAKE_TIMEOUT", cfg.Outbound.Transport.TLSHandshakeTimeout)
	if err != nil {
		return Config{}, err
	}

	cfg.Outbound.Transport.ResponseHeaderTimeout, err = lookupDurationPositive(
		"OUTBOUND_TRANSPORT_RESPONSE_HEADER_TIMEOUT", cfg.Outbound.Transport.ResponseHeaderTimeout)
	if err != nil {
		return Config{}, err
	}

	cfg.Outbound.Retry.MaxAttempts, err = lookupIntPositive(
		"OUTBOUND_RETRY_MAX_ATTEMPTS", cfg.Outbound.Retry.MaxAttempts)
	if err != nil {
		return Config{}, err
	}

	cfg.Outbound.Retry.BaseDelay, err = lookupDurationPositive(
		"OUTBOUND_RETRY_BASE_DELAY", cfg.Outbound.Retry.BaseDelay)
	if err != nil {
		return Config{}, err
	}

	cfg.Outbound.Retry.MaxDelay, err = lookupDurationPositive(
		"OUTBOUND_RETRY_MAX_DELAY", cfg.Outbound.Retry.MaxDelay)
	if err != nil {
		return Config{}, err
	}

	baseURL, err := lookupURLAbsoluteParsed(
		"OUTBOUND_PROFILE_BASE_URL", cfg.Outbound.Profile.Base.String(),
	)
	if err != nil {
		return Config{}, err
	}
	cfg.Outbound.Profile.Base = baseURL

	cfg.Outbound.Profile.Timeout, err = lookupDurationPositive(
		"OUTBOUND_PROFILE_TIMEOUT", cfg.Outbound.Profile.Timeout)
	if err != nil {
		return Config{}, err
	}

	cfg.Auth.JWT.AllowedAlg, err = lookupStringNonEmpty(
		"AUTH_JWT_ALLOWED_ALG", cfg.Auth.JWT.AllowedAlg)
	if err != nil {
		return Config{}, err
	}
	if err := validateJWTAlg(cfg.Auth.JWT.AllowedAlg); err != nil {
		return Config{}, err
	}

	cfg.Auth.JWT.ClockSkew, err = lookupDurationNonNegative(
		"AUTH_JWT_CLOCK_SKEW", cfg.Auth.JWT.ClockSkew)
	if err != nil {
		return Config{}, err
	}

	cfg.Auth.JWT.Issuer, err = lookupStringNonEmpty(
		"AUTH_JWT_ISSUER", cfg.Auth.JWT.Issuer)
	if err != nil {
		return Config{}, err
	}
	cfg.Auth.JWT.Audience, err = lookupStringNonEmpty(
		"AUTH_JWT_AUDIENCE", cfg.Auth.JWT.Audience)
	if err != nil {
		return Config{}, err
	}

	keysRaw, err := lookupStringNonEmpty(
		"AUTH_JWT_KEYS", joinJWTKeys(cfg.Auth.JWT.Keys))
	if err != nil {
		return Config{}, err
	}
	cfg.Auth.JWT.Keys, err = parseJWTKeys("AUTH_JWT_KEYS", keysRaw)
	if err != nil {
		return Config{}, err
	}

	originsRaw, ok := os.LookupEnv("CORS_ALLOWED_ORIGINS")
	if ok {
		originsRaw = strings.TrimSpace(originsRaw)
		if originsRaw == "" {
			return Config{}, fmt.Errorf("CORS_ALLOWED_ORIGINS but empty")
		}
		cfg.CORS.AllowedOrigins, err = parseCORSOrigins("CORS_ALLOWED_ORIGINS", originsRaw)
		if err != nil {
			return Config{}, err
		}
	}

	methodsRaw, ok := os.LookupEnv("CORS_ALLOWED_METHODS")
	if ok {
		methodsRaw = strings.TrimSpace(methodsRaw)
		if methodsRaw == "" {
			return Config{}, fmt.Errorf("CORS_ALLOWED_METHODS but empty")
		}
		cfg.CORS.AllowedMethods, err = parseHTTPMethodsCSV("CORS_ALLOWED_METHODS", methodsRaw)
		if err != nil {
			return Config{}, err
		}
	}

	headersRaw, ok := os.LookupEnv("CORS_ALLOWED_HEADERS")
	if ok {
		headersRaw = strings.TrimSpace(headersRaw)
		if headersRaw == "" {
			return Config{}, fmt.Errorf("CORS_ALLOWED_HEADERS but empty")
		}
		cfg.CORS.AllowedHeaders, err = parseHTTPHeadersCSV("CORS_ALLOWED_HEADERS", headersRaw)
		if err != nil {
			return Config{}, err
		}
	}

	cfg.CORS.AllowCredentials, err = lookupBool("CORS_ALLOW_CREDENTIALS", cfg.CORS.AllowCredentials)
	if err != nil {
		return Config{}, err
	}

	cfg.CORS.MaxAge, err = lookupDurationNonNegative("CORS_MAX_AGE", cfg.CORS.MaxAge)
	if err != nil {
		return Config{}, err
	}

	if cfg.CORS.AllowCredentials && containsString(cfg.CORS.AllowedOrigins, "*") {
		return Config{}, fmt.Errorf("CORS_ALLOW_CREDENTIALS=true запрещает origin=\"*\" (CORS_ALLOWED_ORIGINS)")
	}

	proxiesRaw, ok := os.LookupEnv("PROXY_TRUSTED_PROXIES")
	if ok {
		proxiesRaw = strings.TrimSpace(proxiesRaw)
		if proxiesRaw == "" {
			return Config{}, fmt.Errorf("PROXY_TRUSTED_PROXIES but empty")
		}
		cfg.Proxy.TrustedProxies, err = parseTrustedProxies("PROXY_TRUSTED_PROXIES", proxiesRaw)
		if err != nil {
			return Config{}, err
		}
	}

	cfg.Proxy.TrustXFF, err = lookupBool("PROXY_TRUST_XFF", cfg.Proxy.TrustXFF)
	if err != nil {
		return Config{}, err
	}
	cfg.Proxy.TrustXFP, err = lookupBool("PROXY_TRUST_XFP", cfg.Proxy.TrustXFP)
	if err != nil {
		return Config{}, err
	}

	cfg.SecurityHeaders.Enable, err = lookupBool("SECURITY_HEADERS_ENABLE", cfg.SecurityHeaders.Enable)
	if err != nil {
		return Config{}, err
	}

	cfg.SecurityHeaders.ReferrerPolicy, err = lookupStringNonEmpty(
		"SECURITY_HEADERS_REFERRER_POLICY", cfg.SecurityHeaders.ReferrerPolicy)
	if err != nil {
		return Config{}, err
	}

	cfg.SecurityHeaders.FrameMode, err = lookupStringNonEmpty(
		"SECURITY_HEADERS_FRAME_MODE", cfg.SecurityHeaders.FrameMode)
	if err != nil {
		return Config{}, err
	}
	if err := validateFrameMode(cfg.SecurityHeaders.FrameMode); err != nil {
		return Config{}, err
	}

	cfg.SecurityHeaders.HSTS.Enable, err = lookupBool(
		"SECURITY_HEADERS_HSTS_ENABLE", cfg.SecurityHeaders.HSTS.Enable)
	if err != nil {
		return Config{}, err
	}

	cfg.SecurityHeaders.HSTS.MaxAge, err = lookupDurationNonNegative(
		"SECURITY_HEADERS_HSTS_MAX_AGE", cfg.SecurityHeaders.HSTS.MaxAge)
	if err != nil {
		return Config{}, err
	}
	if cfg.SecurityHeaders.HSTS.Enable && cfg.SecurityHeaders.HSTS.MaxAge <= 0 {
		return Config{}, fmt.Errorf("SECURITY_HEADERS_HSTS_ENABLE=true требует SECURITY_HEADERS_HSTS_MAX_AGE > 0")
	}

	return cfg, nil
}

func lookupStringNonEmpty(key, def string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def, nil
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "", fmt.Errorf("%s but empty", key)
	}
	return v, nil
}

func lookupIntPositive(key string, def int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def, nil
	}
	v = strings.TrimSpace(v)
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: parse int: %w", key, v, err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("%s=%q: must be > 0", key, v)
	}
	return n, nil
}

func lookupDurationPositive(key string, def time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def, nil
	}
	v = strings.TrimSpace(v)
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: parse duration: %w", key, v, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s=%q: must be > 0", key, v)
	}
	return d, nil
}

func lookupBool(key string, def bool) (bool, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s=%q: parse bool: %w", key, v, err)
	}
	return b, nil
}

func lookupURLAbsoluteParsed(key, def string) (*url.URL, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		v = def
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, fmt.Errorf("%s is set but empty", key)
	}

	u, err := url.Parse(v)
	if err != nil {
		return nil, fmt.Errorf("%s=%q: parse url: %w", key, v, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%s=%q: scheme=%q: expected http or https", key, v, u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("%s=%q: host is empty", key, v)
	}
	if u.Fragment != "" {
		return nil, fmt.Errorf("%s=%q: fragment is not allowed", key, v)
	}

	normalized := strings.TrimRight(v, "/")

	uu, err := url.Parse(normalized)
	if err != nil {
		return nil, fmt.Errorf("%s=%q: parse normalized url: %w", key, normalized, err)
	}
	return uu, nil
}
func lookupIntNonNegative(key string, def int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def, nil
	}
	v = strings.TrimSpace(v)
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: parse int: %w", key, v, err)
	}
	if n < 0 {
		return 0, fmt.Errorf("%s=%q: must be >= 0", key, v)
	}
	return n, nil
}

func lookupDurationNonNegative(key string, def time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def, nil
	}
	v = strings.TrimSpace(v)
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: parse duration: %w", key, v, err)
	}
	if d < 0 {
		return 0, fmt.Errorf("%s=%q: must be >= 0", key, v)
	}
	return d, nil
}

func splitCSVStrict(key, v string) ([]string, error) {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, fmt.Errorf("%s=%q: empty CSV item", key, v)
		}
		out = append(out, p)
	}
	return out, nil
}

func dedupePreserveOrder(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, x := range in {
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

func containsString(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func validateJWTAlg(alg string) error {
	switch strings.TrimSpace(alg) {
	case "HS256":
		return nil
	default:
		return fmt.Errorf("AUTH_JWT_ALLOWED_ALG=%q: unsupported alg (allowed: HS256)", alg)
	}
}

func joinJWTKeys(keys []JWTKey) string {
	if len(keys) == 0 {
		return ""
	}
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(strings.TrimSpace(k.KID))
		b.WriteString(":")
		b.WriteString(k.Secret)
	}
	return b.String()
}

func parseJWTKeys(key, v string) ([]JWTKey, error) {
	items, err := splitCSVStrict(key, v)
	if err != nil {
		return nil, err
	}
	out := make([]JWTKey, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, it := range items {
		i := strings.IndexByte(it, ':')
		if i <= 0 || i >= len(it)-1 {
			return nil, fmt.Errorf("%s=%q: expected item format kid:secret", key, v)
		}
		kid := strings.TrimSpace(it[:i])
		sec := it[i+1:]
		if kid == "" {
			return nil, fmt.Errorf("%s=%q: kid is empty", key, v)
		}
		if strings.TrimSpace(sec) == "" {
			return nil, fmt.Errorf("%s=%q: secret for kid=%q is empty", key, v, kid)
		}
		if _, ok := seen[kid]; ok {
			return nil, fmt.Errorf("%s=%q: duplicate kid=%q", key, v, kid)
		}
		seen[kid] = struct{}{}
		out = append(out, JWTKey{KID: kid, Secret: sec})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s=%q: must contain at least 1 key", key, v)
	}
	return out, nil
}

func parseCORSOrigins(key, v string) ([]string, error) {
	items, err := splitCSVStrict(key, v)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		if it == "*" {
			out = append(out, "*")
			continue
		}
		origin, err := normalizeOrigin(key, it)
		if err != nil {
			return nil, err
		}
		out = append(out, origin)
	}
	return dedupePreserveOrder(out), nil
}

func normalizeOrigin(key, raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("%s=%q: origin is empty", key, raw)
	}

	s = strings.TrimRight(s, "/")

	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("%s=%q: parse origin: %w", key, raw, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("%s=%q: scheme=%q: expected http or https", key, raw, u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("%s=%q: host is empty", key, raw)
	}
	if u.User != nil {
		return "", fmt.Errorf("%s=%q: userinfo is not allowed in origin", key, raw)
	}
	if u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("%s=%q: path/query/fragment are not allowed in origin", key, raw)
	}

	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host), nil
}

func parseHTTPMethodsCSV(key, v string) ([]string, error) {
	items, err := splitCSVStrict(key, v)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		m := strings.ToUpper(strings.TrimSpace(it))
		if m == "" || strings.ContainsAny(m, " \t") {
			return nil, fmt.Errorf("%s=%q: invalid method=%q", key, v, it)
		}
		out = append(out, m)
	}
	return dedupePreserveOrder(out), nil
}

func parseHTTPHeadersCSV(key, v string) ([]string, error) {
	items, err := splitCSVStrict(key, v)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		h := strings.TrimSpace(it)
		if h == "" || strings.ContainsAny(h, " \t") {
			return nil, fmt.Errorf("%s=%q: invalid header=%q", key, v, it)
		}
		out = append(out, http.CanonicalHeaderKey(h))
	}
	return dedupePreserveOrder(out), nil
}

func parseTrustedProxies(key, v string) ([]netip.Prefix, error) {
	items, err := splitCSVStrict(key, v)
	if err != nil {
		return nil, err
	}
	out := make([]netip.Prefix, 0, len(items))
	seen := make(map[string]struct{}, len(items))

	for _, it := range items {
		if p, err := netip.ParsePrefix(it); err == nil {
			n := p.Masked()
			s := n.String()
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, n)
			continue
		}

		if a, err := netip.ParseAddr(it); err == nil {
			bits := 32
			if a.Is6() {
				bits = 128
			}
			p := netip.PrefixFrom(a, bits)
			s := p.String()
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, p)
			continue
		}

		return nil, fmt.Errorf("%s=%q: invalid item=%q (expected CIDR or IP)", key, v, it)
	}

	return out, nil
}

func validateFrameMode(mode string) error {
	switch strings.TrimSpace(mode) {
	case "xfo_deny", "csp_frame_ancestors_none":
		return nil
	default:
		return fmt.Errorf("SECURITY_HEADERS_FRAME_MODE=%q: expected xfo_deny or csp_frame_ancestors_none", mode)
	}
}
