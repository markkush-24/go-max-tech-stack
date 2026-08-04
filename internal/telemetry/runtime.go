package telemetry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"

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

	shutdownOnce sync.Once
	shutdownErr  error
}

type runtimeOptions struct {
	traceExporter          sdktrace.SpanExporter
	useSimpleSpanProcessor bool
	metricReader           metric.Reader
	resourceFromEnv        bool
}

func New(ctx context.Context, cfg config.TelemetryConfig, build BuildInfo) (*Runtime, error) {
	return newRuntime(ctx, cfg, build, runtimeOptions{
		resourceFromEnv: cfg.Enabled,
	})
}

func newRuntime(ctx context.Context, cfg config.TelemetryConfig, build BuildInfo, opts runtimeOptions) (*Runtime, error) {
	res, err := newResource(ctx, build, opts.resourceFromEnv)
	if err != nil {
		return nil, err
	}

	traceOptions := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}
	metricOptions := []metric.Option{metric.WithResource(res)}

	var traceExporter sdktrace.SpanExporter
	if cfg.Enabled {
		traceExporter = opts.traceExporter
		if traceExporter == nil {
			traceExporter, err = otlptracegrpc.New(ctx)
			if err != nil {
				return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
			}
		}

		if opts.useSimpleSpanProcessor {
			traceOptions = append(traceOptions, sdktrace.WithSyncer(traceExporter))
		} else {
			traceOptions = append(traceOptions, sdktrace.WithBatcher(traceExporter))
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
	}, nil
}

func newOTLPMetricReader(ctx context.Context) (metric.Reader, error) {
	exporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create OTLP metric exporter: %w", err)
	}
	return metric.NewPeriodicReader(exporter), nil
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

func (r *Runtime) InstallGlobals() {
	if r == nil {
		return
	}
	otel.SetTracerProvider(r.TracerProvider())
	otel.SetMeterProvider(r.MeterProvider())
	otel.SetTextMapPropagator(r.Propagator())
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}

	r.shutdownOnce.Do(func() {
		r.shutdownErr = errors.Join(
			r.tracerProvider.Shutdown(ctx),
			r.meterProvider.Shutdown(ctx),
		)
	})

	return r.shutdownErr
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
