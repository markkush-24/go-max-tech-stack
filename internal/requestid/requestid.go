package requestid

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const HeaderName = "X-Request-Id"

type ctxKeyRequestID struct{}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID{}, id)
}

func RequestID(ctx context.Context) (string, bool) {
	v := ctx.Value(ctxKeyRequestID{})
	s, ok := v.(string)
	return s, ok && s != ""
}

var ridSeq atomic.Uint64

func NewRequestID() string {
	n := ridSeq.Add(1)
	return strings.ToLower(strconv.FormatInt(time.Now().UnixNano(), 36)) + "-" + strconv.FormatInt(int64(n), 36)
}

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid, ok := sanitizeRequestID(r.Header.Get(HeaderName))
		if !ok {
			rid = NewRequestID()
		}

		w.Header().Set(HeaderName, rid)
		ctx := WithRequestID(r.Context(), rid)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func sanitizeRequestID(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 128 {
		return "", false
	}
	for i := 0; i < len(s); i++ {
		b := s[i]
		// запрещаем control chars, пробелы и non-ASCII
		if b < 0x21 || b > 0x7e {
			return "", false
		}
	}
	return s, true
}
