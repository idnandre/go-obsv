package lambda

import (
	"context"
	"log"
	"runtime/metrics"
	"strings"
	"sync"

	"github.com/idnandre/gobsv/internal/metadata"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var (
	meterProvider *sdkmetric.MeterProvider
	traceProvider *sdktrace.TracerProvider
	once          sync.Once
)

func New(ctx context.Context, otlpHttpTarget, serviceName string) {
	once.Do(func() {
		exp, err := newOTLPTraceExporter(ctx, otlpHttpTarget)
		if err != nil {
			log.Fatalf("failed to initialize exporter: %v", err)
		}
		traceProvider = newTraceProvider(exp, serviceName)
		otel.SetTracerProvider(traceProvider)
		otel.SetTextMapPropagator(propagation.TraceContext{})

		expM, err := newOTLPMetricExporter(ctx, otlpHttpTarget)
		if err != nil {
			log.Fatalf("failed to initialize exporter: %v", err)
		}
		meterProvider = newMeterProvider(expM)
		otel.SetMeterProvider(meterProvider)
		addMetricsToOTEL(meterProvider, serviceName)
	})
}

func ForceFlush(ctx context.Context) error {
	return traceProvider.ForceFlush(ctx)
}

func Shutdown(ctx context.Context) {
	go func(ctx context.Context) {
		traceProvider.Shutdown(ctx)
	}(ctx)
	go func(ctx context.Context) {
		meterProvider.Shutdown(ctx)
	}(ctx)
}

// OTLP Trace Exporter
func newOTLPTraceExporter(ctx context.Context, otlpHttpTarget string) (sdktrace.SpanExporter, error) {
	// Change default HTTPS -> HTTP
	insecureOpt := otlptracehttp.WithInsecure()

	// Update default OTLP reciver endpoint
	endpointOpt := otlptracehttp.WithEndpoint(otlpHttpTarget)

	return otlptracehttp.New(ctx, insecureOpt, endpointOpt)
}

// OTLP Metric Exporter
func newOTLPMetricExporter(ctx context.Context, otlpHttpTarget string) (sdkmetric.Exporter, error) {
	// Change default HTTPS -> HTTP
	insecureOpt := otlpmetrichttp.WithInsecure()

	// Update default OTLP reciver endpoint
	endpointOpt := otlpmetrichttp.WithEndpoint(otlpHttpTarget)

	return otlpmetrichttp.New(ctx, insecureOpt, endpointOpt)
}

// TracerProvider is an OpenTelemetry TracerProvider.
// It provides Tracers to instrumentation so it can trace operational flow through a system.
func newTraceProvider(exp sdktrace.SpanExporter, serviceName string) *sdktrace.TracerProvider {
	// Ensure default SDK resources and the required service name are set.
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)

	if err != nil {
		panic(err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
}

func newMeterProvider(exp sdkmetric.Exporter) *sdkmetric.MeterProvider {
	extraResources, _ := resource.New(
		context.Background(),
		resource.WithOS(),
		resource.WithProcess(),
		resource.WithContainer(),
		resource.WithHost(),
	)
	resource, _ := resource.Merge(
		resource.Default(),
		extraResources,
	)

	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)),
		sdkmetric.WithResource(resource),
	)
}

func addMetricsToOTEL(provider *sdkmetric.MeterProvider, serviceName string) {
	meter := provider.Meter(serviceName)

	// Get descriptions for all supported metrics.
	metricsMeta := metrics.All()

	// Register metrics and retrieve the values in prometheus client
	for i := range metricsMeta {
		// Get metric options
		meta := metricsMeta[i]
		opt := getMetricsOptions(metricsMeta[i])
		name := normalizeOtelName(meta.Name)

		// Register metrics per type of metric
		if meta.Cumulative {
			// Register as a counter
			counter, err := meter.Float64ObservableCounter(name, otelmetric.WithDescription(meta.Description))
			if err != nil {
				log.Fatal(err)
			}
			_, err = meter.RegisterCallback(func(_ context.Context, o otelmetric.Observer) error {
				o.ObserveFloat64(counter, metadata.GetSingleMetricFloat(meta.Name), opt)
				return nil
			}, counter)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			// Register as a gauge
			gauge, err := meter.Float64ObservableGauge(name, otelmetric.WithDescription(meta.Description))
			if err != nil {
				log.Fatal(err)
			}
			_, err = meter.RegisterCallback(func(_ context.Context, o otelmetric.Observer) error {
				o.ObserveFloat64(gauge, metadata.GetSingleMetricFloat(meta.Name), opt)
				return nil
			}, gauge)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

// getMetricsOptions function to get metric labels
func getMetricsOptions(metric metrics.Description) otelmetric.MeasurementOption {
	tokens := strings.Split(metric.Name, "/")
	if len(tokens) < 2 {
		return nil
	}

	nameTokens := strings.Split(tokens[len(tokens)-1], ":")
	subsystem := metadata.GetMetricSubsystemName(metric)

	// create a unique name for metric, that will be its primary key on the registry
	opt := otelmetric.WithAttributes(
		attribute.Key("Namespace").String(tokens[1]),
		attribute.Key("Subsystem").String(subsystem),
		attribute.Key("Units").String(nameTokens[1]),
	)
	return opt
}

// normalizePrometheusName function to normalize prometheus metric name
func normalizeOtelName(name string) string {
	normalizedName := strings.Replace(name, "/", "", 1)
	normalizedName = strings.Replace(normalizedName, ":", "_", -1)
	normalizedName = strings.TrimSpace(strings.ReplaceAll(normalizedName, "/", "_"))
	return normalizedName
}
