package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"pet-study/internal/config"
	"pet-study/internal/requestid"
)

const (
	logComponentHTTPSecurity = "http_security"

	logFieldEvent      = "event"
	logFieldDecision   = "decision"
	logFieldAuthNKind  = "authn_kind"
	logFieldAuthZKind  = "authz_kind"
	logFieldCORSDenial = "cors_denial"
	logFieldErrorKind  = "error_kind"

	logDecisionDenied = "denied"
)

func defaultSecurityLogger() *slog.Logger {
	return slog.Default().With(config.LogFieldComponent, logComponentHTTPSecurity)
}

func normalizeLogger(logger *slog.Logger, component string) *slog.Logger {
	if logger != nil {
		return logger
	}
	return slog.Default().With(config.LogFieldComponent, component)
}

func requestLogAttrs(r *http.Request, status int) []slog.Attr {
	rid, _ := requestid.RequestID(r.Context())
	route := canonicalHTTPRoute(r.Pattern)
	if route == "" {
		route = "unmatched"
	}

	return []slog.Attr{
		slog.String(config.LogFieldMethod, r.Method),
		slog.String(config.LogFieldRoute, route),
		slog.Int(config.LogFieldStatus, status),
		slog.String(config.LogFieldRequestID, rid),
	}
}

func logSecurityDecision(
	ctx context.Context,
	logger *slog.Logger,
	event string,
	attrs []slog.Attr,
) {
	allAttrs := make([]slog.Attr, 0, 2+len(attrs))
	allAttrs = append(allAttrs,
		slog.String(logFieldEvent, event),
		slog.String(logFieldDecision, logDecisionDenied),
	)
	allAttrs = append(allAttrs, attrs...)

	logger.LogAttrs(ctx, slog.LevelInfo, "security decision", allAttrs...)
}
