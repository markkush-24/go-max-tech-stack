package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	unsetEnv(t,
		"HTTP_TLS_ENABLE",
		"HTTP_TLS_ADDR",
		"HTTP_TLS_CERT_FILE",
		"HTTP_TLS_KEY_FILE",
		"GRPC_ENABLE",
		"GRPC_ADDR",
		"STREAMING_SSE_HEARTBEAT",
		"STREAMING_SUBSCRIBER_BUFFER",
		"STREAMING_WRITE_TIMEOUT",
	)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTP.TLS.Enable {
		t.Fatalf("HTTP.TLS.Enable=true want=false")
	}
}

func TestLoad_TLSEnabled_MissingCertOrKey_Error(t *testing.T) {
	unsetEnv(t,
		"HTTP_TLS_CERT_FILE",
		"HTTP_TLS_KEY_FILE",
		"HTTP_TLS_ADDR",
	)

	t.Setenv("HTTP_TLS_ENABLE", "true")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoad_TLSEnabled_WithCertAndKey_OK(t *testing.T) {
	t.Setenv("HTTP_TLS_ENABLE", "true")
	t.Setenv("HTTP_TLS_ADDR", ":8443")
	t.Setenv("HTTP_TLS_CERT_FILE", "certs/localhost.pem")
	t.Setenv("HTTP_TLS_KEY_FILE", "certs/localhost-key.pem")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.HTTP.TLS.Enable {
		t.Fatalf("HTTP.TLS.Enable=false want=true")
	}
}

func TestLoad_GRPCEnabled_EmptyAddr_Error(t *testing.T) {
	t.Setenv("GRPC_ENABLE", "true")
	t.Setenv("GRPC_ADDR", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "GRPC_ADDR") {
		t.Fatalf("error=%q want mention of GRPC_ADDR", err)
	}
}

func TestLoad_GRPCEnabled_WithAddr_OK(t *testing.T) {
	t.Setenv("GRPC_ENABLE", "true")
	t.Setenv("GRPC_ADDR", ":9191")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.GRPC.Enable {
		t.Fatalf("GRPC.Enable=false want=true")
	}
	if cfg.GRPC.Addr != ":9191" {
		t.Fatalf("GRPC.Addr=%q want=%q", cfg.GRPC.Addr, ":9191")
	}
}

func TestLoad_StreamingHeartbeat_Invalid_Error(t *testing.T) {
	t.Setenv("STREAMING_SSE_HEARTBEAT", "0s")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "STREAMING_SSE_HEARTBEAT") {
		t.Fatalf("error=%q want mention of STREAMING_SSE_HEARTBEAT", err)
	}
}

func TestLoad_StreamingSubscriberBuffer_Invalid_Error(t *testing.T) {
	t.Setenv("STREAMING_SUBSCRIBER_BUFFER", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "STREAMING_SUBSCRIBER_BUFFER") {
		t.Fatalf("error=%q want mention of STREAMING_SUBSCRIBER_BUFFER", err)
	}
}

func TestLoad_StreamingWriteTimeout_Invalid_Error(t *testing.T) {
	t.Setenv("STREAMING_WRITE_TIMEOUT", "0s")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "STREAMING_WRITE_TIMEOUT") {
		t.Fatalf("error=%q want mention of STREAMING_WRITE_TIMEOUT", err)
	}
}

func TestLoad_InvalidBoolFlags_Error(t *testing.T) {
	t.Run("HTTP_TLS_ENABLE", func(t *testing.T) {
		t.Setenv("HTTP_TLS_ENABLE", "not-bool")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "HTTP_TLS_ENABLE") {
			t.Fatalf("error=%q want mention of HTTP_TLS_ENABLE", err)
		}
	})

	t.Run("GRPC_ENABLE", func(t *testing.T) {
		t.Setenv("GRPC_ENABLE", "not-bool")

		_, err := Load()
		if err == nil {
			t.Fatal("Load() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "GRPC_ENABLE") {
			t.Fatalf("error=%q want mention of GRPC_ENABLE", err)
		}
	})
}

func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()

	type oldValue struct {
		value string
		ok    bool
	}

	old := make(map[string]oldValue, len(keys))
	for _, key := range keys {
		v, ok := os.LookupEnv(key)
		old[key] = oldValue{value: v, ok: ok}
		_ = os.Unsetenv(key)
	}

	t.Cleanup(func() {
		for _, key := range keys {
			if v, ok := old[key]; ok && v.ok {
				_ = os.Setenv(key, v.value)
			} else {
				_ = os.Unsetenv(key)
			}
		}
	})
}
