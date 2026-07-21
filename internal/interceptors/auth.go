package interceptors

import (
	"context"
	"errors"
	"strings"

	"pet-study/internal/security"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func UnaryAuthenticate(verifier security.Verifier) (grpc.UnaryServerInterceptor, error) {
	if verifier == nil {
		return nil, errors.New("grpc auth: verifier is nil")
	}

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		token, err := bearerTokenFromIncomingMetadata(ctx)
		if err != nil {
			return nil, grpcUnauthenticated(err)
		}

		principal, err := verifier.Verify(token)
		if err != nil {
			return nil, grpcUnauthenticated(security.UnauthorizedFromVerifyErr(err))
		}

		return handler(security.WithPrincipal(ctx, principal), req)
	}, nil
}

func bearerTokenFromIncomingMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", security.NewUnauthorized(security.AuthNMissing, nil)
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", security.NewUnauthorized(security.AuthNMissing, nil)
	}
	if len(values) != 1 {
		return "", security.NewUnauthorized(security.AuthNInvalid, nil)
	}

	return parseBearerToken(values[0])
}

func parseBearerToken(header string) (string, error) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", security.NewUnauthorized(security.AuthNMissing, nil)
	}

	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", security.NewUnauthorized(security.AuthNInvalid, nil)
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", security.NewUnauthorized(security.AuthNInvalid, nil)
	}
	return token, nil
}

func grpcUnauthenticated(err error) error {
	if kind, ok := security.AuthNKind(err); ok && kind == security.AuthNMissing {
		return status.Error(codes.Unauthenticated, "authentication required")
	}
	return status.Error(codes.Unauthenticated, "invalid bearer token")
}
