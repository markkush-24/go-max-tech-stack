package interceptors

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUnaryClientRequestIDAndTimeoutPreservesOutgoingMetadata(t *testing.T) {
	interceptor := UnaryClientRequestIDAndTimeout(2 * time.Second)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"authorization", "Bearer token",
		"traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-00",
	))

	var gotMD metadata.MD
	var deadline time.Time
	err := interceptor(ctx, "/pb.JobsService/GetJob", nil, nil, nil, func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		opts ...grpc.CallOption,
	) error {
		gotMD, _ = metadata.FromOutgoingContext(ctx)
		var ok bool
		deadline, ok = ctx.Deadline()
		if !ok {
			t.Fatal("missing deadline")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}

	if got := gotMD.Get("authorization"); len(got) != 1 || got[0] != "Bearer token" {
		t.Fatalf("authorization metadata=%v", got)
	}
	if got := gotMD.Get("traceparent"); len(got) != 1 {
		t.Fatalf("traceparent metadata=%v", got)
	}
	if got := gotMD.Get(requestIDMetadataKey); len(got) != 1 || got[0] == "" {
		t.Fatalf("request-id metadata=%v", got)
	}
	if time.Until(deadline) > 2*time.Second {
		t.Fatalf("deadline=%v exceeds configured timeout", deadline)
	}
}

func TestUnaryClientRequestIDAndTimeoutPreservesValidRequestID(t *testing.T) {
	interceptor := UnaryClientRequestIDAndTimeout(0)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(requestIDMetadataKey, "rid-ok"))

	var gotMD metadata.MD
	err := interceptor(ctx, "/pb.JobsService/GetJob", nil, nil, nil, func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		opts ...grpc.CallOption,
	) error {
		gotMD, _ = metadata.FromOutgoingContext(ctx)
		return nil
	})
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}

	if got := gotMD.Get(requestIDMetadataKey); len(got) != 1 || got[0] != "rid-ok" {
		t.Fatalf("request-id metadata=%v", got)
	}
}

func TestUnaryClientRequestIDAndTimeoutReplacesInvalidRequestID(t *testing.T) {
	interceptor := UnaryClientRequestIDAndTimeout(0)
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		requestIDMetadataKey, "bad id",
		"authorization", "Bearer token",
	))

	var gotMD metadata.MD
	err := interceptor(ctx, "/pb.JobsService/GetJob", nil, nil, nil, func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		opts ...grpc.CallOption,
	) error {
		gotMD, _ = metadata.FromOutgoingContext(ctx)
		return nil
	})
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}

	if got := gotMD.Get("authorization"); len(got) != 1 || got[0] != "Bearer token" {
		t.Fatalf("authorization metadata=%v", got)
	}
	got := gotMD.Get(requestIDMetadataKey)
	if len(got) != 1 || got[0] == "" || got[0] == "bad id" {
		t.Fatalf("request-id metadata=%v", got)
	}
}

func TestUnaryClientRequestIDAndTimeoutKeepsEarlierDeadline(t *testing.T) {
	interceptor := UnaryClientRequestIDAndTimeout(2 * time.Second)
	parent, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var deadline time.Time
	err := interceptor(parent, "/pb.JobsService/GetJob", nil, nil, nil, func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		opts ...grpc.CallOption,
	) error {
		var ok bool
		deadline, ok = ctx.Deadline()
		if !ok {
			t.Fatal("missing deadline")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}

	if time.Until(deadline) > time.Second {
		t.Fatalf("deadline=%v should keep earlier parent deadline", deadline)
	}
}
