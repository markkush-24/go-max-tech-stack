package telemetry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"pet-study/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	metricapi "go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	traceapi "go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

const (
	DefaultServiceName    = "pet-study"
	DefaultServiceVersion = "dev"
	DefaultEnvironment    = config.EnvironmentDevelopment
	LogComponent          = "telemetry"
)

const (
	defaultTraceMaxQueueSize       = 2048
	defaultTraceMaxExportBatchSize = 512
	defaultTraceBatchTimeout       = 5 * time.Second
	defaultTelemetryExportTimeout  = 5 * time.Second
	defaultMetricExportInterval    = time.Minute
	defaultRetryInitialInterval    = time.Second
	defaultRetryMaxInterval        = 5 * time.Second
	defaultRetryMaxElapsedTime     = 30 * time.Second
	defaultDiagnosticsLogInterval  = time.Minute
)

type BuildInfo struct {
	ServiceName       string
	ServiceVersion    string
	ServiceInstanceID string
	Environment       string
}

type Runtime struct {
	enabled        bool
	resource       *resource.Resource
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *metric.MeterProvider
	propagator     propagation.TextMapPropagator
	diagnostics    *diagnostics
	errorHandler   *otelErrorHandler

	globalsMu            sync.Mutex
	globalsInstalled     bool
	previousErrorHandler otel.ErrorHandler

	shutdownOnce sync.Once
	shutdownErr  error
}

type runtimeOptions struct {
	traceExporter          sdktrace.SpanExporter
	useSimpleSpanProcessor bool
	metricReader           metric.Reader
	resourceFromEnv        bool
	diagnosticsLogInterval time.Duration
}

type DiagnosticsSnapshot struct {
	Enabled                bool
	StartupFailures        uint64
	ExportFailures         uint64
	SuppressedExportLogs   uint64
	ForceFlushes           uint64
	Shutdowns              uint64
	LastStartupFailureKind string
	LastExportFailureKind  string
}

func New(ctx context.Context, cfg config.TelemetryConfig, build BuildInfo) (*Runtime, error) {
	return newRuntime(ctx, cfg, build, runtimeOptions{
		resourceFromEnv: cfg.Enabled,
	})
}

func NewFailOpen(ctx context.Context, cfg config.TelemetryConfig, build BuildInfo, logger *slog.Logger) (*Runtime, error) {
	rt, err := New(ctx, cfg, build)
	if err == nil {
		return rt, nil
	}
	if !cfg.Enabled {
		return nil, err
	}

	fallbackCfg := cfg
	fallbackCfg.Enabled = false
	fallback, fallbackErr := New(ctx, fallbackCfg, build)
	if fallbackErr != nil {
		return nil, errors.Join(err, fallbackErr)
	}
	fallback.diagnostics.recordStartupFailure(err)
	if logger != nil {
		logger.Warn("telemetry bootstrap failed; continuing with telemetry disabled",
			"event", "telemetry.bootstrap.failed",
			"outcome", "disabled",
			"error_kind", telemetryErrorKind(err),
		)
	}
	return fallback, nil
}

func newRuntime(ctx context.Context, cfg config.TelemetryConfig, build BuildInfo, opts runtimeOptions) (*Runtime, error) {
	res, err := newResource(ctx, build, opts.resourceFromEnv)
	if err != nil {
		return nil, err
	}

	diagnostics := newDiagnostics()
	logInterval := opts.diagnosticsLogInterval
	if logInterval <= 0 {
		logInterval = defaultDiagnosticsLogInterval
	}

	traceOptions := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}
	metricOptions := []metric.Option{metric.WithResource(res)}

	var traceExporter sdktrace.SpanExporter
	if cfg.Enabled {
		traceExporter = opts.traceExporter
		if traceExporter == nil {
			traceExporter, err = otlptracegrpc.New(ctx,
				otlptracegrpc.WithTimeout(defaultTelemetryExportTimeout),
				otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{
					Enabled:         true,
					InitialInterval: defaultRetryInitialInterval,
					MaxInterval:     defaultRetryMaxInterval,
					MaxElapsedTime:  defaultRetryMaxElapsedTime,
				}),
			)
			if err != nil {
				return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
			}
		}

		if opts.useSimpleSpanProcessor {
			traceOptions = append(traceOptions, sdktrace.WithSyncer(traceExporter))
		} else {
			traceOptions = append(traceOptions, sdktrace.WithBatcher(traceExporter,
				sdktrace.WithMaxQueueSize(defaultTraceMaxQueueSize),
				sdktrace.WithMaxExportBatchSize(defaultTraceMaxExportBatchSize),
				sdktrace.WithBatchTimeout(defaultTraceBatchTimeout),
				sdktrace.WithExportTimeout(defaultTelemetryExportTimeout),
			))
		}

		reader := opts.metricReader
		if reader == nil {
			reader, err = newOTLPMetricReader(ctx)
			if err != nil {
				shutdownTraceExporter(ctx, traceExporter)
				return nil, err
			}
		}
		metricOptions = append(metricOptions, metric.WithReader(reader))
	}

	return &Runtime{
		enabled:        cfg.Enabled,
		resource:       res,
		tracerProvider: sdktrace.NewTracerProvider(traceOptions...),
		meterProvider:  metric.NewMeterProvider(metricOptions...),
		propagator:     propagation.TraceContext{},
		diagnostics:    diagnostics,
		errorHandler:   newOTelErrorHandler(diagnostics, logInterval),
	}, nil
}

func newOTLPMetricReader(ctx context.Context) (metric.Reader, error) {
	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithTimeout(defaultTelemetryExportTimeout),
		otlpmetricgrpc.WithRetry(otlpmetricgrpc.RetryConfig{
			Enabled:         true,
			InitialInterval: defaultRetryInitialInterval,
			MaxInterval:     defaultRetryMaxInterval,
			MaxElapsedTime:  defaultRetryMaxElapsedTime,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTLP metric exporter: %w", err)
	}
	return metric.NewPeriodicReader(exporter,
		metric.WithInterval(defaultMetricExportInterval),
		metric.WithTimeout(defaultTelemetryExportTimeout),
	), nil
}

func shutdownTraceExporter(ctx context.Context, exporter sdktrace.SpanExporter) {
	if exporter == nil {
		return
	}
	_ = exporter.Shutdown(ctx)
}

func (r *Runtime) Enabled() bool {
	return r != nil && r.enabled
}

func (r *Runtime) Resource() *resource.Resource {
	if r == nil {
		return resource.Empty()
	}
	return r.resource
}

func (r *Runtime) TracerProvider() traceapi.TracerProvider {
	if r == nil {
		return tracenoop.NewTracerProvider()
	}
	return r.tracerProvider
}

func (r *Runtime) MeterProvider() metricapi.MeterProvider {
	if r == nil {
		return metricnoop.NewMeterProvider()
	}
	return r.meterProvider
}

func (r *Runtime) Propagator() propagation.TextMapPropagator {
	if r == nil {
		return propagation.TraceContext{}
	}
	return r.propagator
}

func (r *Runtime) Diagnostics() DiagnosticsSnapshot {
	if r == nil {
		return DiagnosticsSnapshot{}
	}
	return r.diagnostics.snapshot(r.Enabled())
}

func (r *Runtime) InstallGlobals() {
	r.InstallGlobalsWithLogger(slog.Default().With(config.LogFieldComponent, LogComponent))
}

func (r *Runtime) InstallGlobalsWithLogger(logger *slog.Logger) {
	if r == nil {
		return
	}
	otel.SetTracerProvider(r.TracerProvider())
	otel.SetMeterProvider(r.MeterProvider())
	otel.SetTextMapPropagator(r.Propagator())
	r.installErrorHandler(logger)
}

func (r *Runtime) installErrorHandler(logger *slog.Logger) {
	r.globalsMu.Lock()
	defer r.globalsMu.Unlock()

	if !r.globalsInstalled {
		r.previousErrorHandler = otel.GetErrorHandler()
		r.globalsInstalled = true
	}
	r.errorHandler.setLogger(logger)
	otel.SetErrorHandler(r.errorHandler)
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}

	r.shutdownOnce.Do(func() {
		flushErr := r.ForceFlush(ctx)
		shutdownErr := errors.Join(
			r.tracerProvider.Shutdown(ctx),
			r.meterProvider.Shutdown(ctx),
		)
		r.diagnostics.recordShutdown()
		r.restoreErrorHandler()
		r.shutdownErr = errors.Join(
			flushErr,
			shutdownErr,
		)
	})

	return r.shutdownErr
}

func (r *Runtime) ForceFlush(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.diagnostics.recordForceFlush()
	return errors.Join(
		r.tracerProvider.ForceFlush(ctx),
		r.meterProvider.ForceFlush(ctx),
	)
}

func (r *Runtime) restoreErrorHandler() {
	r.globalsMu.Lock()
	defer r.globalsMu.Unlock()

	if r.globalsInstalled {
		otel.SetErrorHandler(r.previousErrorHandler)
		r.globalsInstalled = false
	}
}

func newResource(ctx context.Context, build BuildInfo, includeEnv bool) (*resource.Resource, error) {
	normalized, err := normalizeBuildInfo(build)
	if err != nil {
		return nil, err
	}

	res := resource.NewSchemaless(
		semconv.ServiceName(normalized.ServiceName),
		semconv.ServiceVersion(normalized.ServiceVersion),
		semconv.ServiceInstanceID(normalized.ServiceInstanceID),
		semconv.DeploymentEnvironmentName(normalized.Environment),
	)

	sdkResource, err := resource.New(ctx, resource.WithTelemetrySDK())
	if err != nil {
		return nil, fmt.Errorf("create telemetry SDK resource: %w", err)
	}
	res, err = resource.Merge(res, sdkResource)
	if err != nil {
		return nil, fmt.Errorf("merge telemetry SDK resource: %w", err)
	}

	if includeEnv {
		envResource, err := resource.New(ctx, resource.WithFromEnv())
		if err != nil {
			return nil, fmt.Errorf("create environment telemetry resource: %w", err)
		}
		res, err = resource.Merge(res, envResource)
		if err != nil {
			return nil, fmt.Errorf("merge environment telemetry resource: %w", err)
		}
	}

	return res, nil
}

func normalizeBuildInfo(build BuildInfo) (BuildInfo, error) {
	serviceName := defaultString(build.ServiceName, DefaultServiceName)
	serviceVersion := defaultString(build.ServiceVersion, DefaultServiceVersion)
	environment := defaultString(build.Environment, DefaultEnvironment)
	instanceID := strings.TrimSpace(build.ServiceInstanceID)
	if instanceID == "" {
		generated, err := generateInstanceID()
		if err != nil {
			return BuildInfo{}, err
		}
		instanceID = generated
	}

	return BuildInfo{
		ServiceName:       serviceName,
		ServiceVersion:    serviceVersion,
		ServiceInstanceID: instanceID,
		Environment:       environment,
	}, nil
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func generateInstanceID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate telemetry service.instance.id: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

type diagnostics struct {
	startupFailures        atomic.Uint64
	exportFailures         atomic.Uint64
	suppressedExportLogs   atomic.Uint64
	forceFlushes           atomic.Uint64
	shutdowns              atomic.Uint64
	lastStartupFailureKind atomic.Value
	lastExportFailureKind  atomic.Value
}

func newDiagnostics() *diagnostics {
	return &diagnostics{}
}

func (d *diagnostics) snapshot(enabled bool) DiagnosticsSnapshot {
	if d == nil {
		return DiagnosticsSnapshot{Enabled: enabled}
	}
	return DiagnosticsSnapshot{
		Enabled:                enabled,
		StartupFailures:        d.startupFailures.Load(),
		ExportFailures:         d.exportFailures.Load(),
		SuppressedExportLogs:   d.suppressedExportLogs.Load(),
		ForceFlushes:           d.forceFlushes.Load(),
		Shutdowns:              d.shutdowns.Load(),
		LastStartupFailureKind: loadAtomicString(&d.lastStartupFailureKind),
		LastExportFailureKind:  loadAtomicString(&d.lastExportFailureKind),
	}
}

func (d *diagnostics) recordStartupFailure(err error) {
	if d == nil {
		return
	}
	d.startupFailures.Add(1)
	d.lastStartupFailureKind.Store(telemetryErrorKind(err))
}

func (d *diagnostics) recordExportFailure(err error) uint64 {
	if d == nil {
		return 0
	}
	d.lastExportFailureKind.Store(telemetryErrorKind(err))
	return d.exportFailures.Add(1)
}

func (d *diagnostics) recordSuppressedExportLog() {
	if d == nil {
		return
	}
	d.suppressedExportLogs.Add(1)
}

func (d *diagnostics) recordForceFlush() {
	if d == nil {
		return
	}
	d.forceFlushes.Add(1)
}

func (d *diagnostics) recordShutdown() {
	if d == nil {
		return
	}
	d.shutdowns.Add(1)
}

type otelErrorHandler struct {
	diagnostics       *diagnostics
	logInterval       time.Duration
	nextLogUnixNano   atomic.Int64
	pendingSuppressed atomic.Uint64
	logger            atomic.Value
}

func newOTelErrorHandler(diagnostics *diagnostics, logInterval time.Duration) *otelErrorHandler {
	if logInterval <= 0 {
		logInterval = defaultDiagnosticsLogInterval
	}
	return &otelErrorHandler{
		diagnostics: diagnostics,
		logInterval: logInterval,
	}
}

func (h *otelErrorHandler) Handle(err error) {
	if h == nil || err == nil {
		return
	}
	total := h.diagnostics.recordExportFailure(err)
	logger := h.loadLogger()
	if logger == nil {
		return
	}

	now := time.Now()
	nowUnixNano := now.UnixNano()
	for {
		next := h.nextLogUnixNano.Load()
		if next > nowUnixNano {
			h.pendingSuppressed.Add(1)
			h.diagnostics.recordSuppressedExportLog()
			return
		}
		if h.nextLogUnixNano.CompareAndSwap(next, now.Add(h.logInterval).UnixNano()) {
			break
		}
	}

	logger.Warn("telemetry export failed",
		"event", "telemetry.export.failed",
		"error_kind", telemetryErrorKind(err),
		"suppressed_logs", h.pendingSuppressed.Swap(0),
		"total_failures", total,
	)
}

func (h *otelErrorHandler) setLogger(logger *slog.Logger) {
	if h == nil {
		return
	}
	if logger == nil {
		logger = slog.Default().With(config.LogFieldComponent, LogComponent)
	}
	h.logger.Store(logger)
}

func (h *otelErrorHandler) loadLogger() *slog.Logger {
	value := h.logger.Load()
	if logger, ok := value.(*slog.Logger); ok {
		return logger
	}
	return nil
}

func telemetryErrorKind(err error) string {
	switch {
	case err == nil:
		return "none"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	default:
		return "telemetry"
	}
}

func loadAtomicString(value *atomic.Value) string {
	raw := value.Load()
	if text, ok := raw.(string); ok {
		return text
	}
	return ""
}
