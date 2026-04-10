package observability

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// SetupOTelSDK bootstraps the OpenTelemetry pipeline for shipping to otel-collector.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func SetupOTelSDK(ctx context.Context, serviceName, serviceVersion, otelCollectorEndpoint, otelHeaders string) (func(context.Context) error, error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	shutdown := func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) error {
		return errors.Join(inErr, shutdown(ctx))
	}

	// Create resource with service information
	res, err := newResource(serviceName, serviceVersion)
	if err != nil {
		return shutdown, handleErr(err)
	}

	// Set up propagator
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up trace provider
	tracerProvider, err := newTracerProvider(ctx, res, otelCollectorEndpoint, otelHeaders)
	if err != nil {
		return shutdown, handleErr(err)
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set up meter provider
	meterProvider, err := newMeterProvider(ctx, res, otelCollectorEndpoint, otelHeaders)
	if err != nil {
		return shutdown, handleErr(err)
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	return shutdown, nil
}

func newResource(serviceName, serviceVersion string) (*resource.Resource, error) {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(serviceVersion),
		semconv.ServiceInstanceID("pos-service"),
	), nil
}

// GetLogger returns a structured logger that integrates with OpenTelemetry
func GetLogger(name string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With("service", name)
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTracerProvider(ctx context.Context, res *resource.Resource, endpoint, headers string) (*trace.TracerProvider, error) {
	slog.Info("Configuring OTLP tracer",
		slog.String("endpoint", endpoint),
		slog.String("headers_raw", headers))

	// Parse headers from the environment variable format
	headerMap := make(map[string]string)
	if headers != "" {
		pairs := strings.Split(headers, ",")
		for _, pair := range pairs {
			if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
				headerMap[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
		slog.Info("OTLP headers configured", slog.Int("header_count", len(headerMap)))
	}

	slog.Info("Using OTLP endpoint", slog.String("endpoint", endpoint))

	options := []otlptracehttp.Option{
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithEndpoint(endpoint),
	}

	if len(headerMap) > 0 {
		options = append(options, otlptracehttp.WithHeaders(headerMap))
	}

	traceExporter, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		return nil, err
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter,
			trace.WithBatchTimeout(time.Second*5),
			trace.WithMaxExportBatchSize(512),
		),
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()),
	)
	return tracerProvider, nil
}

func newMeterProvider(ctx context.Context, res *resource.Resource, endpoint, headers string) (*sdkmetric.MeterProvider, error) {
	slog.Info("Configuring OTLP metrics exporter", slog.String("endpoint", endpoint))

	// Parse headers from the environment variable format (if any)
	headerMap := make(map[string]string)
	if headers != "" {
		pairs := strings.Split(headers, ",")
		for _, pair := range pairs {
			if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
				headerMap[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(
			&noOpMetricExporter{},
			sdkmetric.WithInterval(30*time.Second),
		)),
	)

	slog.Info("Metrics provider configured")
	return meterProvider, nil
}

// Temporary no-op exporter until we add OTLP metrics support
type noOpMetricExporter struct{}

func (e *noOpMetricExporter) Temporality(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (e *noOpMetricExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(kind)
}

func (e *noOpMetricExporter) Export(context.Context, *metricdata.ResourceMetrics) error {
	slog.Debug("Metrics exported (no-op)")
	return nil
}

func (e *noOpMetricExporter) ForceFlush(context.Context) error { return nil }
func (e *noOpMetricExporter) Shutdown(context.Context) error   { return nil }

// SetupFileLogger configures slog to write JSON logs to a file
func SetupFileLogger(logFilePath string) error {
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	jsonHandler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	})

	logger := slog.New(jsonHandler)
	slog.SetDefault(logger)

	slog.Info("File logging configured",
		slog.String("log_file", logFilePath),
		slog.String("log_dir", logDir))

	return nil
}

// CreateMetrics creates and returns common application metrics
func CreateMetrics() (*AppMetrics, error) {
	meter := otel.Meter("pos-service")

	requestCounter, err := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		return nil, err
	}

	requestDuration, err := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
	)
	if err != nil {
		return nil, err
	}

	dbConnections, err := meter.Int64UpDownCounter(
		"db_connections_active",
		metric.WithDescription("Number of active database connections"),
	)
	if err != nil {
		return nil, err
	}

	return &AppMetrics{
		RequestCounter:  requestCounter,
		RequestDuration: requestDuration,
		DBConnections:   dbConnections,
	}, nil
}

// AppMetrics holds the application metrics
type AppMetrics struct {
	RequestCounter  metric.Int64Counter
	RequestDuration metric.Float64Histogram
	DBConnections   metric.Int64UpDownCounter
}

// LogConfig holds configuration for file logging
type LogConfig struct {
	FilePath   string
	MaxSizeMB  int64
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

// DefaultLogConfig returns a default logging configuration
func DefaultLogConfig() LogConfig {
	return LogConfig{
		FilePath:   "/logs/otel-logs.json",
		MaxSizeMB:  100,
		MaxBackups: 5,
		MaxAgeDays: 30,
		Compress:   true,
	}
}

// SetupAdvancedFileLogger configures slog with advanced file logging features
func SetupAdvancedFileLogger(config LogConfig) error {
	logDir := filepath.Dir(config.FilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logFile, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	jsonHandler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   "timestamp",
					Value: slog.StringValue(time.Now().UTC().Format(time.RFC3339)),
				}
			}
			return a
		},
	})

	logger := slog.New(jsonHandler)
	slog.SetDefault(logger)

	slog.Info("Advanced file logging configured",
		slog.String("log_file", config.FilePath),
		slog.String("log_dir", logDir),
		slog.Int64("max_size_mb", config.MaxSizeMB),
		slog.Int("max_backups", config.MaxBackups),
		slog.Int("max_age_days", config.MaxAgeDays))

	return nil
}
