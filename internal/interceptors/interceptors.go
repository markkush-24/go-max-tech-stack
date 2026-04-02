package interceptors

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"pet-study/internal/requestid"
)

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
			values := md.Get("request-id")
			if len(values) > 0 && values[0] != "" {
				rid = values[0]
			}
		}
		if rid == "" {
			rid = requestid.NewRequestID()
		}

		ctx = requestid.WithRequestID(ctx, rid)

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
