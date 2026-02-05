package config

import (
	"fmt"
	"net/http"
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

type Config struct {
	HTTP    HTTPConfig
	DB      DBConfig
	Pool    WorkerPoolConfig
	Limiter RateLimiterConfig
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

	cfg.DB.DSN, err = lookupStringNonEmpty("DB_DSN", cfg.DB.DSN)
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
		return "", fmt.Errorf("%s is set but empty", key)
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
