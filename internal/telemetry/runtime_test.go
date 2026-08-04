package telemetry

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"pet-study/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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

func TestNewFailOpenDisablesTelemetryOnBootstrapFailure(t *testing.T) {
	suppressOTelErrors(t)
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "invalid-resource-attribute")

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	rt, err := NewFailOpen(context.Background(), config.TelemetryConfig{
		Enabled:         true,
		ShutdownTimeout: time.Second,
	}, BuildInfo{
		ServiceName:       "pet-study-test",
		ServiceVersion:    "test-version",
		ServiceInstanceID: "fail-open-instance",
		Environment:       config.EnvironmentTest,
	}, logger)
	if err != nil {
		t.Fatalf("NewFailOpen() error = %v", err)
	}
	t.Cleanup(func() {
		if err := rt.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if rt.Enabled() {
		t.Fatal("Runtime.Enabled()=true want false after fail-open fallback")
	}
	snapshot := rt.Diagnostics()
	if snapshot.StartupFailures != 1 {
		t.Fatalf("StartupFailures=%d want=1", snapshot.StartupFailures)
	}
	if snapshot.LastStartupFailureKind != "telemetry" {
		t.Fatalf("LastStartupFailureKind=%q want telemetry", snapshot.LastStartupFailureKind)
	}
	if got := logs.String(); !strings.Contains(got, `"event":"telemetry.bootstrap.failed"`) {
		t.Fatalf("logs=%q want telemetry.bootstrap.failed event", got)
	}
	if got := logs.String(); strings.Contains(got, "invalid-resource-attribute") {
		t.Fatalf("logs must not include raw telemetry bootstrap error, got %q", got)
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

func TestInstallGlobalsSurfacesOTelErrorsThroughDiagnostics(t *testing.T) {
	previousTracerProvider := otel.GetTracerProvider()
	previousMeterProvider := otel.GetMeterProvider()
	previousPropagator := otel.GetTextMapPropagator()
	previousErrorHandler := otel.GetErrorHandler()
	t.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		otel.SetMeterProvider(previousMeterProvider)
		otel.SetTextMapPropagator(previousPropagator)
		otel.SetErrorHandler(previousErrorHandler)
	})

	rt, err := newRuntime(context.Background(), config.TelemetryConfig{
		Enabled:         false,
		ShutdownTimeout: time.Second,
	}, BuildInfo{
		ServiceName:       "pet-study-test",
		ServiceVersion:    "test-version",
		ServiceInstanceID: "diagnostics-instance",
		Environment:       config.EnvironmentTest,
	}, runtimeOptions{
		diagnosticsLogInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("newRuntime() error = %v", err)
	}
	t.Cleanup(func() {
		if err := rt.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	var logs bytes.Buffer
	rt.InstallGlobalsWithLogger(slog.New(slog.NewJSONHandler(&logs, nil)))

	otel.Handle(errors.New("collector token secret"))
	otel.Handle(context.DeadlineExceeded)

	snapshot := rt.Diagnostics()
	if snapshot.ExportFailures != 2 {
		t.Fatalf("ExportFailures=%d want=2", snapshot.ExportFailures)
	}
	if snapshot.SuppressedExportLogs != 1 {
		t.Fatalf("SuppressedExportLogs=%d want=1", snapshot.SuppressedExportLogs)
	}
	if snapshot.LastExportFailureKind != "deadline_exceeded" {
		t.Fatalf("LastExportFailureKind=%q want deadline_exceeded", snapshot.LastExportFailureKind)
	}
	if got := logs.String(); !strings.Contains(got, `"event":"telemetry.export.failed"`) {
		t.Fatalf("logs=%q want telemetry.export.failed event", got)
	}
	if got := logs.String(); strings.Contains(got, "collector token secret") {
		t.Fatalf("logs must not include raw telemetry export error, got %q", got)
	}
}

func TestShutdownForceFlushesAndShutsDownOnce(t *testing.T) {
	ctx := context.Background()
	exporter := &countingSpanExporter{}
	rt, err := newRuntime(ctx, config.TelemetryConfig{
		Enabled:         true,
		ShutdownTimeout: time.Second,
	}, BuildInfo{
		ServiceName:       "pet-study-test",
		ServiceVersion:    "test-version",
		ServiceInstanceID: "shutdown-instance",
		Environment:       config.EnvironmentTest,
	}, runtimeOptions{
		traceExporter:   exporter,
		metricReader:    metric.NewManualReader(),
		resourceFromEnv: false,
	})
	if err != nil {
		t.Fatalf("newRuntime() error = %v", err)
	}

	_, span := rt.TracerProvider().Tracer("telemetry-test").Start(ctx, "final-span")
	span.End()

	if err := rt.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := rt.Shutdown(ctx); err != nil {
		t.Fatalf("second Shutdown() error = %v", err)
	}
	if got := exporter.exportCalls.Load(); got != 1 {
		t.Fatalf("export calls=%d want=1", got)
	}
	if got := exporter.shutdownCalls.Load(); got != 1 {
		t.Fatalf("shutdown calls=%d want=1", got)
	}
	snapshot := rt.Diagnostics()
	if snapshot.ForceFlushes != 1 {
		t.Fatalf("ForceFlushes=%d want=1", snapshot.ForceFlushes)
	}
	if snapshot.Shutdowns != 1 {
		t.Fatalf("Shutdowns=%d want=1", snapshot.Shutdowns)
	}
}

func TestShutdownHonorsContextBudget(t *testing.T) {
	ctx := context.Background()
	exporter := &blockingSpanExporter{}
	rt, err := newRuntime(ctx, config.TelemetryConfig{
		Enabled:         true,
		ShutdownTimeout: 20 * time.Millisecond,
	}, BuildInfo{
		ServiceName:       "pet-study-test",
		ServiceVersion:    "test-version",
		ServiceInstanceID: "shutdown-budget-instance",
		Environment:       config.EnvironmentTest,
	}, runtimeOptions{
		traceExporter:   exporter,
		metricReader:    metric.NewManualReader(),
		resourceFromEnv: false,
	})
	if err != nil {
		t.Fatalf("newRuntime() error = %v", err)
	}

	_, span := rt.TracerProvider().Tracer("telemetry-test").Start(ctx, "final-span")
	span.End()

	shutdownCtx, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	started := time.Now()
	err = rt.Shutdown(shutdownCtx)
	elapsed := time.Since(started)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown() error=%v want context deadline exceeded", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("Shutdown() elapsed=%s want <=500ms", elapsed)
	}
	if exporter.exportCalls.Load() != 1 {
		t.Fatalf("export calls=%d want=1", exporter.exportCalls.Load())
	}
	snapshot := rt.Diagnostics()
	if snapshot.ForceFlushes != 1 {
		t.Fatalf("ForceFlushes=%d want=1", snapshot.ForceFlushes)
	}
	if snapshot.Shutdowns != 1 {
		t.Fatalf("Shutdowns=%d want=1", snapshot.Shutdowns)
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

type countingSpanExporter struct {
	exportCalls   atomic.Uint64
	shutdownCalls atomic.Uint64
}

func (e *countingSpanExporter) ExportSpans(_ context.Context, _ []sdktrace.ReadOnlySpan) error {
	e.exportCalls.Add(1)
	return nil
}

func (e *countingSpanExporter) Shutdown(_ context.Context) error {
	e.shutdownCalls.Add(1)
	return nil
}

type blockingSpanExporter struct {
	exportCalls atomic.Uint64
}

func (e *blockingSpanExporter) ExportSpans(ctx context.Context, _ []sdktrace.ReadOnlySpan) error {
	e.exportCalls.Add(1)
	<-ctx.Done()
	return ctx.Err()
}

func (e *blockingSpanExporter) Shutdown(_ context.Context) error {
	return nil
}
