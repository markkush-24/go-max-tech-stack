package config

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type LogFormat string

const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

type LoggingConfig struct {
	Level     LogLevel
	Format    LogFormat
	AddSource bool
}

const (
	LogFieldComponent  = "component"
	LogFieldRequestID  = "request_id"
	LogFieldMethod     = "method"
	LogFieldRoute      = "route"
	LogFieldDurationMS = "duration_ms"
	LogFieldStatus     = "status"
	LogFieldError      = "err"
)

const (
	LogComponentMain            = "main"
	LogComponentHTTPAccess      = "http_access"
	LogComponentHTTPRecover     = "http_recover"
	LogComponentOutboundProfile = "outbound_profile"
	LogComponentGRPCServer      = "grpc_server"
)

func NewLogger(w io.Writer, cfg LoggingConfig) (*slog.Logger, error) {
	if w == nil {
		return nil, fmt.Errorf("log writer is required")
	}

	level, err := cfg.Level.slogLevel()
	if err != nil {
		return nil, err
	}

	opts := &slog.HandlerOptions{
		AddSource: cfg.AddSource,
		Level:     level,
	}

	switch normalizeLogFormat(cfg.Format) {
	case LogFormatText:
		return slog.New(slog.NewTextHandler(w, opts)), nil
	case LogFormatJSON:
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	default:
		return nil, fmt.Errorf("log format %q: unsupported log format (allowed: text, json)", cfg.Format)
	}
}

func lookupLogLevel(key string, def LogLevel) (LogLevel, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def, nil
	}

	v = strings.TrimSpace(v)
	if v == "" {
		return "", fmt.Errorf("%s but empty", key)
	}

	level := LogLevel(strings.ToLower(v))
	if _, err := level.slogLevel(); err != nil {
		return "", fmt.Errorf("%s=%q: unsupported log level (allowed: debug, info, warn, error)", key, v)
	}
	return level, nil
}

func lookupLogFormat(key string, def LogFormat) (LogFormat, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def, nil
	}

	v = strings.TrimSpace(v)
	if v == "" {
		return "", fmt.Errorf("%s but empty", key)
	}

	format := LogFormat(strings.ToLower(v))
	if !isValidLogFormat(format) {
		return "", fmt.Errorf("%s=%q: unsupported log format (allowed: text, json)", key, v)
	}
	return format, nil
}

func (l LogLevel) slogLevel() (slog.Level, error) {
	switch normalizeLogLevel(l) {
	case LogLevelDebug:
		return slog.LevelDebug, nil
	case LogLevelInfo:
		return slog.LevelInfo, nil
	case LogLevelWarn:
		return slog.LevelWarn, nil
	case LogLevelError:
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("log level %q: unsupported log level (allowed: debug, info, warn, error)", l)
	}
}

func normalizeLogLevel(level LogLevel) LogLevel {
	return LogLevel(strings.ToLower(strings.TrimSpace(string(level))))
}

func normalizeLogFormat(format LogFormat) LogFormat {
	return LogFormat(strings.ToLower(strings.TrimSpace(string(format))))
}

func isValidLogFormat(format LogFormat) bool {
	switch normalizeLogFormat(format) {
	case LogFormatText, LogFormatJSON:
		return true
	default:
		return false
	}
}
