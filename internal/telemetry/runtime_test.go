package telemetry

import (
	"context"
	"strings"
	"testing"
	"time"

	"pet-study/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewDisabledIgnoresOTelEnvironmentAndBuildsDeterministicRuntime(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "ignored-service")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "invalid-resource-attribute")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "://bad")

	rt, err := New(context.Background(), config.TelemetryConfig{
		Enabled:         false,
		ShutdownTimeout: time.Second,
	}, BuildInfo{
		ServiceName:       "pet-study-test",
		ServiceVersion:    "test-version",
		ServiceInstanceID: "test-instance",
		Environment:       config.EnvironmentTest,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if rt.Enabled() {
		t.Fatal("Runtime.Enabled()=true want false")
	}
	if rt.TracerProvider() == nil {
		t.Fatal("TracerProvider() is nil")
	}
	if rt.MeterProvider() == nil {
		t.Fatal("MeterProvider() is nil")
	}
	assertResourceAttribute(t, rt.Resource(), "service.name", "pet-study-test")
	assertResourceAttribute(t, rt.Resource(), "service.version", "test-version")
	assertResourceAttribute(t, rt.Resource(), "service.instance.id", "test-instance")
	assertResourceAttribute(t, rt.Resource(), "deployment.environment.name", config.EnvironmentTest)

	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := rt.Shutdown(context.Background()); err != nil {
		t.Fatalf("second Shutdown() error = %v", err)
	}
}

func TestRuntimeProvidersShareResource(t *testing.T) {
	ctx := context.Background()
	traceExporter := tracetest.NewInMemoryExporter()
	reader := metric.NewManualReader()

	rt, err := newRuntime(ctx, config.TelemetryConfig{
		Enabled:         true,
		ShutdownTimeout: time.Second,
	}, BuildInfo{
		ServiceName:       "pet-study-test",
		ServiceVersion:    "test-version",
		ServiceInstanceID: "shared-instance",
		Environment:       config.EnvironmentTest,
	}, runtimeOptions{
		traceExporter:          traceExporter,
		useSimpleSpanProcessor: true,
		metricReader:           reader,
		resourceFromEnv:        false,
	})
	if err != nil {
		t.Fatalf("newRuntime() error = %v", err)
	}
	t.Cleanup(func() {
		if err := rt.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	_, span := rt.TracerProvider().Tracer("telemetry-test").Start(ctx, "test-span")
	span.End()

	spans := traceExporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("exported spans=%d want=1", len(spans))
	}

	counter, err := rt.MeterProvider().Meter("telemetry-test").Int64Counter("test_counter")
	if err != nil {
		t.Fatalf("Int64Counter() error = %v", err)
	}
	counter.Add(ctx, 1)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if !spans[0].Resource.Equal(rt.Resource()) {
		t.Fatalf("span resource does not match runtime resource: span=%v runtime=%v",
			spans[0].Resource.Attributes(), rt.Resource().Attributes())
	}
	if !rm.Resource.Equal(rt.Resource()) {
		t.Fatalf("metric resource does not match runtime resource: metric=%v runtime=%v",
			rm.Resource.Attributes(), rt.Resource().Attributes())
	}
	if !spans[0].Resource.Equal(rm.Resource) {
		t.Fatalf("trace and metric resources differ: trace=%v metric=%v",
			spans[0].Resource.Attributes(), rm.Resource.Attributes())
	}
}

func TestNewResourceUsesStandardOTelEnvironmentWhenEnabled(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "env-service")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", strings.Join([]string{
		"service.version=env-version",
		"service.instance.id=env-instance",
		"deployment.environment.name=staging",
	}, ","))

	rt, err := newRuntime(context.Background(), config.TelemetryConfig{
		Enabled:         true,
		ShutdownTimeout: time.Second,
	}, BuildInfo{
		ServiceName:       "build-service",
		ServiceVersion:    "build-version",
		ServiceInstanceID: "build-instance",
		Environment:       config.EnvironmentDevelopment,
	}, runtimeOptions{
		traceExporter:          tracetest.NewNoopExporter(),
		useSimpleSpanProcessor: true,
		metricReader:           metric.NewManualReader(),
		resourceFromEnv:        true,
	})
	if err != nil {
		t.Fatalf("newRuntime() error = %v", err)
	}
	t.Cleanup(func() {
		if err := rt.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	assertResourceAttribute(t, rt.Resource(), "service.name", "env-service")
	assertResourceAttribute(t, rt.Resource(), "service.version", "env-version")
	assertResourceAttribute(t, rt.Resource(), "service.instance.id", "env-instance")
	assertResourceAttribute(t, rt.Resource(), "deployment.environment.name", "staging")
}

func TestNewResourceRejectsInvalidOTelResourceEnvironmentWhenEnabled(t *testing.T) {
	suppressOTelErrors(t)
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "invalid-resource-attribute")

	_, err := newRuntime(context.Background(), config.TelemetryConfig{
		Enabled:         true,
		ShutdownTimeout: time.Second,
	}, BuildInfo{
		ServiceName:       "pet-study-test",
		ServiceVersion:    "test-version",
		ServiceInstanceID: "test-instance",
		Environment:       config.EnvironmentTest,
	}, runtimeOptions{
		traceExporter:          tracetest.NewNoopExporter(),
		useSimpleSpanProcessor: true,
		metricReader:           metric.NewManualReader(),
		resourceFromEnv:        true,
	})
	if err == nil {
		t.Fatal("newRuntime() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "environment telemetry resource") {
		t.Fatalf("error=%q want environment telemetry resource", err)
	}
}

func TestInstallGlobalsUsesRuntimeProvidersAndPropagator(t *testing.T) {
	previousTracerProvider := otel.GetTracerProvider()
	previousMeterProvider := otel.GetMeterProvider()
	previousPropagator := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		otel.SetMeterProvider(previousMeterProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})

	rt, err := New(context.Background(), config.TelemetryConfig{
		Enabled:         false,
		ShutdownTimeout: time.Second,
	}, BuildInfo{
		ServiceName:       "pet-study-test",
		ServiceVersion:    "test-version",
		ServiceInstanceID: "global-instance",
		Environment:       config.EnvironmentTest,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := rt.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	rt.InstallGlobals()

	if otel.GetTracerProvider() != rt.TracerProvider() {
		t.Fatal("global TracerProvider was not installed from runtime")
	}
	if otel.GetMeterProvider() != rt.MeterProvider() {
		t.Fatal("global MeterProvider was not installed from runtime")
	}
	if fields := otel.GetTextMapPropagator().Fields(); !sameStrings(fields, propagation.TraceContext{}.Fields()) {
		t.Fatalf("global propagator fields=%v want trace context fields", fields)
	}
}

func assertResourceAttribute(t *testing.T, res interface{ Attributes() []attribute.KeyValue }, key string, want string) {
	t.Helper()

	for _, attr := range res.Attributes() {
		if string(attr.Key) == key {
			if got := attr.Value.AsString(); got != want {
				t.Fatalf("resource attribute %s=%q want=%q", key, got, want)
			}
			return
		}
	}
	t.Fatalf("resource attribute %s missing from %v", key, res.Attributes())
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int, len(a))
	for _, item := range a {
		seen[item]++
	}
	for _, item := range b {
		if seen[item] == 0 {
			return false
		}
		seen[item]--
	}
	return true
}

func suppressOTelErrors(t *testing.T) {
	t.Helper()

	previous := otel.GetErrorHandler()
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(error) {}))
	t.Cleanup(func() {
		otel.SetErrorHandler(previous)
	})
}
