package httputils

import (
	"context"
	"errors"
	"pet-study/internal/entity"
	"pet-study/internal/security"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrGRPCBridgeUnavailable = errors.New("grpc bridge unavailable")
	ErrGRPCBridgeTimeout     = errors.New("grpc bridge timeout")
	ErrGRPCBridgeCanceled    = errors.New("grpc bridge canceled")
)

const GRPCBridgeCallTimeout = 2 * time.Second

func MapGRPCBridgeError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.Canceled) {
		return ErrGRPCBridgeCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrGRPCBridgeTimeout
	}

	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	switch st.Code() {
	case codes.InvalidArgument:
		return &BadRequestError{Detail: st.Message()}
	case codes.NotFound:
		return entity.ErrJobNotFound
	case codes.PermissionDenied:
		return security.NewForbidden(security.AuthZForbidden, nil)
	case codes.Unauthenticated:
		return security.NewUnauthorized(security.AuthNInvalid, nil)
	case codes.Canceled:
		return ErrGRPCBridgeCanceled
	case codes.DeadlineExceeded:
		return ErrGRPCBridgeTimeout
	case codes.Unavailable:
		return ErrGRPCBridgeUnavailable
	default:
		return err
	}
}
