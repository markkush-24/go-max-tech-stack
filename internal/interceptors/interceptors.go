package interceptors

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"pet-study/internal/requestid"
)

const requestIDMetadataKey = "request-id"

func UnaryRequestIDAndLogging(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		rid := ""
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if sanitized, ok := requestIDFromMetadata(md); ok {
				rid = sanitized
			}
		}
		if rid == "" {
			rid = requestid.NewRequestID()
		}

		ctx = requestid.WithRequestID(ctx, rid)
		_ = grpc.SetHeader(ctx, metadata.Pairs(requestIDMetadataKey, rid))
		grpc.SetTrailer(ctx, metadata.Pairs(requestIDMetadataKey, rid))

		resp, err := handler(ctx, req)

		logger.Info("grpc request completed",
			"request_id", rid,
			"method", info.FullMethod,
			"code", status.Code(err).String(),
			"duration", time.Since(start),
		)

		return resp, err
	}
}

func UnaryClientRequestIDAndTimeout(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx = ensureOutgoingRequestID(ctx)

		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func ensureOutgoingRequestID(ctx context.Context) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return metadata.AppendToOutgoingContext(ctx, requestIDMetadataKey, requestid.NewRequestID())
	}

	if _, ok := requestIDFromMetadata(md); ok {
		return ctx
	}

	out := md.Copy()
	out.Set(requestIDMetadataKey, requestid.NewRequestID())
	return metadata.NewOutgoingContext(ctx, out)
}

func requestIDFromMetadata(md metadata.MD) (string, bool) {
	values := md.Get(requestIDMetadataKey)
	if len(values) != 1 {
		return "", false
	}
	return sanitizeGRPCRequestID(values[0])
}

func sanitizeGRPCRequestID(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 128 {
		return "", false
	}
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b < 0x21 || b > 0x7e {
			return "", false
		}
	}
	return s, true
}
