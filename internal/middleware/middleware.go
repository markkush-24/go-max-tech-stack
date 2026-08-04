package middleware

import (
	"log/slog"
	"net/http"
	"pet-study/internal/config"
	"pet-study/internal/httputils"
	"pet-study/internal/requestid"
	"pet-study/internal/security"
	"runtime/debug"
	"strings"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (w *statusRecorder) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

func Logger(next http.Handler) http.Handler {
	return LoggerWithLogger(next, slog.Default().With(config.LogFieldComponent, config.LogComponentHTTPAccess))
}

func LoggerWithLogger(next http.Handler, logger *slog.Logger) http.Handler {
	logger = normalizeLogger(logger, config.LogComponentHTTPAccess)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		sr, ok := w.(*statusRecorder)
		if !ok {
			sr = newStatusRecorder(w)
			w = sr
		}

		next.ServeHTTP(w, r)

		rid, _ := requestid.RequestID(r.Context())

		scheme := "-"
		if ri, ok := security.RequestInfoFromContext(r.Context()); ok {
			if ri.Scheme != "" {
				scheme = ri.Scheme
			}
		}

		route := canonicalHTTPRoute(r.Pattern)
		if route == "" {
			route = "unmatched"
		}

		logger.LogAttrs(
			r.Context(),
			accessLogLevel(sr.Status(), route),
			"http request completed",
			slog.String(logFieldEvent, "http.request.completed"),
			slog.String(config.LogFieldMethod, r.Method),
			slog.String(config.LogFieldRoute, route),
			slog.Int(config.LogFieldStatus, sr.Status()),
			slog.Int("bytes", sr.Bytes()),
			slog.Int64(config.LogFieldDurationMS, time.Since(start).Milliseconds()),
			slog.String(config.LogFieldRequestID, rid),
			slog.String("scheme", scheme),
		)
	})
}

func canonicalHTTPRoute(pattern string) string {
	method, route, ok := strings.Cut(pattern, " ")
	if ok && isHTTPMethod(method) && route != "" {
		return route
	}
	return pattern
}

func isHTTPMethod(method string) bool {
	switch method {
	case http.MethodConnect,
		http.MethodDelete,
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPatch,
		http.MethodPost,
		http.MethodPut,
		http.MethodTrace:
		return true
	default:
		return false
	}
}

func accessLogLevel(status int, route string) slog.Level {
	if status >= http.StatusBadRequest {
		return slog.LevelInfo
	}
	if isNoisyAccessRoute(route) {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

func isNoisyAccessRoute(route string) bool {
	return route == "/livez" ||
		route == "/readyz" ||
		strings.HasPrefix(route, "/debug/") ||
		strings.HasSuffix(route, "/events")
}

func Recover(next http.Handler) http.Handler {
	return RecoverWithLogger(next, slog.Default().With(config.LogFieldComponent, config.LogComponentHTTPRecover))
}

func RecoverWithLogger(next http.Handler, logger *slog.Logger) http.Handler {
	logger = normalizeLogger(logger, config.LogComponentHTTPRecover)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				rid, _ := requestid.RequestID(r.Context())
				logger.Error(
					"panic recovered",
					logFieldEvent, "http.panic.recovered",
					config.LogFieldRequestID, rid,
					logFieldErrorKind, "panic",
					"stack", string(debug.Stack()),
				)
				_ = httputils.WriteProblem(w, r, httputils.Problem{
					Status:    http.StatusInternalServerError,
					Detail:    "internal server error",
					RequestID: rid,
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
