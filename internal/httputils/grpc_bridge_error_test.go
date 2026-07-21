package httputils

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"pet-study/internal/entity"
	"pet-study/internal/security"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapGRPCBridgeError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantIs     error
		wantAsBad  bool
		wantAuth   bool
		wantForbid bool
	}{
		{name: "invalid argument", err: status.Error(codes.InvalidArgument, "bad id"), wantAsBad: true},
		{name: "not found", err: status.Error(codes.NotFound, "missing"), wantIs: entity.ErrJobNotFound},
		{name: "permission denied", err: status.Error(codes.PermissionDenied, "forbidden"), wantForbid: true},
		{name: "unauthenticated", err: status.Error(codes.Unauthenticated, "auth"), wantAuth: true},
		{name: "canceled", err: status.Error(codes.Canceled, "canceled"), wantIs: ErrGRPCBridgeCanceled},
		{name: "deadline", err: status.Error(codes.DeadlineExceeded, "timeout"), wantIs: ErrGRPCBridgeTimeout},
		{name: "unavailable", err: status.Error(codes.Unavailable, "down"), wantIs: ErrGRPCBridgeUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapGRPCBridgeError(tt.err)
			if tt.wantIs != nil && !errors.Is(got, tt.wantIs) {
				t.Fatalf("error=%v want errors.Is %v", got, tt.wantIs)
			}
			if tt.wantAsBad {
				var bad *BadRequestError
				if !errors.As(got, &bad) {
					t.Fatalf("error=%T want BadRequestError", got)
				}
			}
			if tt.wantAuth {
				var auth *security.UnauthorizedError
				if !errors.As(got, &auth) {
					t.Fatalf("error=%T want UnauthorizedError", got)
				}
			}
			if tt.wantForbid {
				var forbidden *security.ForbiddenError
				if !errors.As(got, &forbidden) {
					t.Fatalf("error=%T want ForbiddenError", got)
				}
			}
		})
	}
}

func TestMapErrorGRPCBridgeStatus(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/1/grpc", nil)

	tests := []struct {
		name   string
		err    error
		status int
	}{
		{name: "unavailable", err: ErrGRPCBridgeUnavailable, status: http.StatusServiceUnavailable},
		{name: "timeout", err: ErrGRPCBridgeTimeout, status: http.StatusGatewayTimeout},
		{name: "canceled", err: ErrGRPCBridgeCanceled, status: http.StatusRequestTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapError(req, tt.err)
			if got.Problem.Status != tt.status {
				t.Fatalf("status=%d want=%d", got.Problem.Status, tt.status)
			}
		})
	}
}
