package config

import (
	"fmt"
	"net/http"
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
	Workers int
}

type RateLimiterConfig struct {
	RPS   int
	BURST int
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
	BaseURL string
	Timeout time.Duration
}

type OutboundConfig struct {
	Profile   OutboundProfileConfig
	Transport OutboundTransportConfig
	Retry     OutboundRetryConfig
}

type Config struct {
	HTTP     HTTPConfig
	DB       DBConfig
	Pool     WorkerPoolConfig
	Limiter  RateLimiterConfig
	Bulkhead BulkheadConfig
	Outbound OutboundConfig
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
			DSN: "", // optional: if DB_DSN is set, it must be non-empty
		},
		Pool: WorkerPoolConfig{
			Workers: 10, // optional: if DB_DSN is set, it must be non-empty
		},
		Limiter: RateLimiterConfig{
			RPS:   5,
			BURST: 10,
		},
		Bulkhead: BulkheadConfig{
			MaxParallel: 1,
		},
		Outbound: OutboundConfig{
			Profile: OutboundProfileConfig{
				BaseURL: "http://localhost:8090",
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

	cfg.Limiter.RPS, err = lookupIntPositive("RATE_LIMIT_RPS", cfg.Limiter.RPS)
	if err != nil {
		return Config{}, err
	}

	cfg.Limiter.BURST, err = lookupIntPositive("RATE_LIMIT_BURST", cfg.Limiter.BURST)
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

	cfg.Outbound.Profile.BaseURL, err = lookupURLAbsolute(
		"OUTBOUND_PROFILE_BASE_URL", cfg.Outbound.Profile.BaseURL)
	if err != nil {
		return Config{}, err
	}

	cfg.Outbound.Profile.Timeout, err = lookupDurationPositive(
		"OUTBOUND_PROFILE_TIMEOUT", cfg.Outbound.Profile.Timeout)
	if err != nil {
		return Config{}, err
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

func lookupURLAbsolute(key, def string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		v = def
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "", fmt.Errorf("%s is set but empty", key)
	}

	u, err := url.Parse(v)
	if err != nil {
		return "", fmt.Errorf("%s=%q: parse url: %w", key, v, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("%s=%q: scheme=%q: expected http or https", key, v, u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("%s=%q: host is empty", key, v)
	}

	if u.Fragment != "" {
		return "", fmt.Errorf("%s=%q: fragment is not allowed", key, v)
	}

	return strings.TrimRight(v, "/"), nil
}
func lookupIntNonNegative(key string, def int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: parse int: %w", key, v, err)
	}
	if n < 0 {
		return 0, fmt.Errorf("%s=%q: must be >= 0", key, v)
	}
	return n, nil
}
