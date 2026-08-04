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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		sr, ok := w.(*statusRecorder)
		if !ok {
			sr = newStatusRecorder(w)
			w = sr
		}

		next.ServeHTTP(w, r)

		rid, _ := requestid.RequestID(r.Context())

		clientIP := "-"
		scheme := "-"
		if ri, ok := security.RequestInfoFromContext(r.Context()); ok {
			if ri.ClientIP != "" {
				clientIP = ri.ClientIP
			}
			if ri.Scheme != "" {
				scheme = ri.Scheme
			}
		}

		logger.Info(
			"http request completed",
			config.LogFieldMethod, r.Method,
			"path", r.URL.Path,
			config.LogFieldRoute, canonicalHTTPRoute(r.Pattern),
			config.LogFieldStatus, sr.Status(),
			"bytes", sr.Bytes(),
			config.LogFieldDurationMS, time.Since(start).Milliseconds(),
			config.LogFieldRequestID, rid,
			"client_ip", clientIP,
			"scheme", scheme,
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

func Recover(next http.Handler) http.Handler {
	return RecoverWithLogger(next, slog.Default().With(config.LogFieldComponent, config.LogComponentHTTPRecover))
}

func RecoverWithLogger(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				rid, _ := requestid.RequestID(r.Context())
				logger.Error(
					"panic recovered",
					config.LogFieldRequestID, rid,
					config.LogFieldError, err,
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
