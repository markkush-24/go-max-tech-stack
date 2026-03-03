package security

import "context"

type RequestInfo struct {
	ClientIP string // "real" client ip (from XFF if trusted, else RemoteAddr)
	Scheme   string // "http" or "https" (from TLS or trusted XFP)
}

type requestInfoKey struct{}

func WithRequestInfo(ctx context.Context, ri RequestInfo) context.Context {
	return context.WithValue(ctx, requestInfoKey{}, ri)
}

func RequestInfoFromContext(ctx context.Context) (RequestInfo, bool) {
	ri, ok := ctx.Value(requestInfoKey{}).(RequestInfo)
	return ri, ok
}
