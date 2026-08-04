package interceptors

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"pet-study/internal/config"
)

func TestUnaryRequestIDAndLogging_UsesCanonicalLogFieldsAndSingleComponent(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil)).
		With(config.LogFieldComponent, config.LogComponentGRPCServer)

	interceptor := UnaryRequestIDAndLogging(logger)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(requestIDMetadataKey, "rid-1"))

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/pb.JobsService/GetJob"}, func(context.Context, any) (any, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("interceptor() error = %v", err)
	}

	line := strings.TrimSpace(logs.String())
	if got := strings.Count(line, `"`+config.LogFieldComponent+`"`); got != 1 {
		t.Fatalf("log=%q component field count=%d want 1", line, got)
	}
	if strings.Contains(line, `"duration"`) {
		t.Fatalf("log=%q must not include legacy duration field", line)
	}

	var record map[string]any
	if err := json.Unmarshal([]byte(line), &record); err != nil {
		t.Fatalf("unmarshal log line: %v line=%q", err, line)
	}

	if record[config.LogFieldComponent] != config.LogComponentGRPCServer {
		t.Fatalf("component=%v want %q", record[config.LogFieldComponent], config.LogComponentGRPCServer)
	}
	if record[config.LogFieldRequestID] != "rid-1" {
		t.Fatalf("request_id=%v want rid-1", record[config.LogFieldRequestID])
	}
	if record[config.LogFieldMethod] != "/pb.JobsService/GetJob" {
		t.Fatalf("method=%v want /pb.JobsService/GetJob", record[config.LogFieldMethod])
	}
	if _, ok := record[config.LogFieldDurationMS]; !ok {
		t.Fatalf("duration_ms missing from log record: %#v", record)
	}
}
