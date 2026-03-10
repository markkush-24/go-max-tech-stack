package middleware

import (
	"log/slog"
	"net/http"
	"pet-study/internal/httputils"
	"pet-study/internal/requestid"
	"pet-study/internal/security"
	"runtime/debug"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusRecorder) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := slog.Default().With("component", "http_access")
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
			"method", r.Method,
			"path", r.URL.Path,
			"pattern", r.Pattern,
			"status", sr.Status(),
			"bytes", sr.Bytes(),
			"latency_ms", time.Since(start).Milliseconds(),
			"request_id", rid,
			"client_ip", clientIP,
			"scheme", scheme,
		)
	})
}

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := slog.Default().With("component", "http_recover")
		defer func() {
			if err := recover(); err != nil {
				rid, _ := requestid.RequestID(r.Context())
				logger.Error(
					"panic recovered",
					"request_id", rid,
					"err", err,
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
