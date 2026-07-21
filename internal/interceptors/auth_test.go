package interceptors

import (
	"context"
	"errors"
	"testing"

	"pet-study/internal/security"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnaryAuthenticateRejectsMissingBearer(t *testing.T) {
	interceptor, err := UnaryAuthenticate(testVerifier{
		token:     "valid",
		principal: security.Principal{UserID: 1, Role: security.RoleUser},
	})
	if err != nil {
		t.Fatalf("UnaryAuthenticate: %v", err)
	}

	called := false
	_, err = interceptor(context.Background(), nil, unaryInfo(), func(context.Context, any) (any, error) {
		called = true
		return nil, nil
	})

	if called {
		t.Fatal("handler was called")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("code=%v want=%v err=%v", status.Code(err), codes.Unauthenticated, err)
	}
}

func TestUnaryAuthenticateRejectsInvalidBearer(t *testing.T) {
	interceptor, err := UnaryAuthenticate(testVerifier{
		token:     "valid",
		principal: security.Principal{UserID: 1, Role: security.RoleUser},
	})
	if err != nil {
		t.Fatalf("UnaryAuthenticate: %v", err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Basic valid"))
	_, err = interceptor(ctx, nil, unaryInfo(), func(context.Context, any) (any, error) {
		t.Fatal("handler was called")
		return nil, nil
	})

	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("code=%v want=%v err=%v", status.Code(err), codes.Unauthenticated, err)
	}
}

func TestUnaryAuthenticateRejectsVerifierFailure(t *testing.T) {
	interceptor, err := UnaryAuthenticate(testVerifier{
		err: &security.AuthNError{Kind: security.AuthNInvalid, Cause: errors.New("bad token")},
	})
	if err != nil {
		t.Fatalf("UnaryAuthenticate: %v", err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer invalid"))
	_, err = interceptor(ctx, nil, unaryInfo(), func(context.Context, any) (any, error) {
		t.Fatal("handler was called")
		return nil, nil
	})

	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("code=%v want=%v err=%v", status.Code(err), codes.Unauthenticated, err)
	}
}

func TestUnaryAuthenticateMapsPrincipalIntoContext(t *testing.T) {
	want := security.Principal{UserID: 42, Role: security.RoleAdmin}
	interceptor, err := UnaryAuthenticate(testVerifier{
		token:     "valid",
		principal: want,
	})
	if err != nil {
		t.Fatalf("UnaryAuthenticate: %v", err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer valid"))
	_, err = interceptor(ctx, nil, unaryInfo(), func(ctx context.Context, _ any) (any, error) {
		got, ok := security.FromContext(ctx)
		if !ok {
			t.Fatal("missing principal")
		}
		if got != want {
			t.Fatalf("principal=%+v want=%+v", got, want)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
}

func TestUnaryAuthenticateRequiresVerifier(t *testing.T) {
	if _, err := UnaryAuthenticate(nil); err == nil {
		t.Fatal("UnaryAuthenticate(nil) succeeded")
	}
}

func unaryInfo() *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{FullMethod: "/pb.JobsService/GetJob"}
}

type testVerifier struct {
	token     string
	principal security.Principal
	err       error
}

func (v testVerifier) Verify(token string) (security.Principal, error) {
	if v.err != nil {
		return security.Principal{}, v.err
	}
	if token != v.token {
		return security.Principal{}, &security.AuthNError{Kind: security.AuthNInvalid}
	}
	return v.principal, nil
}
