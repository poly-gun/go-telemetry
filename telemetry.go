package telemetry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func resources(ctx context.Context) *resource.Resource {
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "local"
	}

	options := []resource.Option{
		resource.WithFromEnv(), // Discover and provide attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables.
		// resource.WithTelemetrySDK(), // Discover and provide information about the OpenTelemetry SDK used.
		// resource.WithOS(),           // Discover and provide OS information.
		// resource.WithHost(),         // Discover and provide host information.
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithContainer(),
		resource.WithContainerID(),
		resource.WithHost(),
		resource.WithHostID(),
		resource.WithAttributes(
			semconv.ServiceNamespaceKey.String(namespace),
		),
	}

	instance, e := resource.New(ctx, options...)
	if errors.Is(e, resource.ErrPartialResource) || errors.Is(e, resource.ErrSchemaURLConflict) {
		slog.WarnContext(ctx, "Non-Fatal Open-Telemetry Error", slog.String("error", e.Error()))
	} else if e != nil {
		e = fmt.Errorf("unable to generate exportable resource: %w", e)
		slog.ErrorContext(ctx, "Fatal Open-Telemetry Error", slog.String("error", e.Error()))
		panic(e)
	}

	// Merge a default tracer with the initial one, overwriting anything in default.
	instance, e = resource.Merge(resource.Default(), instance)
	if e != nil {
		e = fmt.Errorf("unable to merge resource: %w", e)
		slog.ErrorContext(ctx, "Fatal Open-Telemetry Error", slog.String("error", e.Error()))
		panic(e)
	}

	return instance
}

func propagator(settings *Settings) {
	provider := propagation.NewCompositeTextMapPropagator(settings.Propagators...)

	// Register the global propagation provider.
	otel.SetTextMapPropagator(provider)

	return
}

func traces(ctx context.Context, settings *Settings) *trace.TracerProvider {
	options := []trace.TracerProviderOption{
		trace.WithResource(resources(ctx)),
		trace.WithSampler(trace.AlwaysSample()),
	}

	if settings.Tracer.Local && settings.Tracer.Debugger == nil {
		var e error

		var writer io.Writer = os.Stdout
		if settings.Tracer.Writer != nil {
			writer = settings.Tracer.Writer
		}

		settings.Tracer.Debugger, e = stdouttrace.New(stdouttrace.WithoutTimestamps(), stdouttrace.WithPrettyPrint(), stdouttrace.WithWriter(writer))
		if e != nil {
			e = fmt.Errorf("unable to instantiate local tracer: %w", e)
			panic(e)
		}

		exporter := settings.Tracer.Debugger

		options = append(options, trace.WithBatcher(exporter, trace.WithBatchTimeout(time.Second*5)))
	} else if settings.Tracer.Debugger != nil {
		exporter := settings.Tracer.Debugger

		options = append(options, trace.WithBatcher(exporter, trace.WithBatchTimeout(time.Second*5)))
	} else {
		exporter, e := otlptracehttp.New(ctx, settings.Tracer.Options...)
		if e != nil {
			panic(e)
		}

		options = append(options, trace.WithBatcher(exporter, trace.WithBatchTimeout(time.Second*30)))

		if settings.Zipkin.Enabled {
			z, e := zipkin.New(settings.Zipkin.URL)
			if e != nil {
				panic(e)
			}

			options = append(options, trace.WithBatcher(z, trace.WithBatchTimeout(time.Second*30)))
		}
	}

	provider := trace.NewTracerProvider(options...)

	// Register the global tracer provider.
	otel.SetTracerProvider(provider)

	return provider
}

func metrics(ctx context.Context, settings *Settings) *metric.MeterProvider {
	// metricExporter, err := otlpmetrichttp.New(ctx, settings.Metrics.Options...)
	// if err != nil {
	// 	return nil, err
	// }
	//
	// meterProvider := metric.NewMeterProvider(
	// 	metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(30*time.Second))),
	// )
	// return meterProvider, nil

	options := make([]metric.Option, 0)

	if settings.Metrics.Local && settings.Metrics.Debugger == nil {
		var e error

		var writer io.Writer = os.Stdout
		if settings.Metrics.Writer != nil {
			writer = settings.Metrics.Writer
		}

		settings.Metrics.Debugger, e = stdoutmetric.New(stdoutmetric.WithPrettyPrint(), stdoutmetric.WithWriter(writer))
		if e != nil {
			e = fmt.Errorf("unable to instantiate local metrics exporter: %w", e)
			panic(e)
		}

		exporter := settings.Metrics.Debugger

		options = append(options, metric.WithReader(metric.NewPeriodicReader(exporter, metric.WithInterval(5*time.Second))))
	} else if settings.Metrics.Debugger != nil {
		exporter := settings.Metrics.Debugger

		options = append(options, metric.WithReader(metric.NewPeriodicReader(exporter, metric.WithInterval(5*time.Second))))
	} else {
		exporter, e := otlpmetrichttp.New(ctx, settings.Metrics.Options...)
		if e != nil {
			e = fmt.Errorf("unable to instantiate primary metrics exporter: %w", e)
			panic(e)
		}

		options = append(options, metric.WithReader(metric.NewPeriodicReader(exporter, metric.WithInterval(30*time.Second))))
	}

	provider := metric.NewMeterProvider(options...)

	// Set the global meter provider.
	otel.SetMeterProvider(provider)

	return provider
}

func logexporter(ctx context.Context, settings *Settings) *log.LoggerProvider {
	options := make([]log.LoggerProviderOption, 0)

	if settings.Logs.Local && settings.Logs.Debugger == nil {
		var e error

		var writer io.Writer = os.Stdout
		if settings.Logs.Writer != nil {
			writer = settings.Logs.Writer
		}

		settings.Logs.Debugger, e = stdoutlog.New(stdoutlog.WithPrettyPrint(), stdoutlog.WithWriter(writer))
		if e != nil {
			e = fmt.Errorf("unable to instantiate local log exporter: %w", e)
			panic(e)
		}

		exporter := settings.Logs.Debugger

		options = append(options, log.WithProcessor(log.NewSimpleProcessor(exporter)))
	} else if settings.Logs.Debugger != nil {
		exporter := settings.Logs.Debugger

		options = append(options, log.WithProcessor(log.NewSimpleProcessor(exporter)))
	} else {
		exporter, e := otlploghttp.New(ctx, settings.Logs.Options...)
		if e != nil {
			e = fmt.Errorf("unable to instantiate primary log exporter: %w", e)
			panic(e)
		}

		options = append(options, log.WithProcessor(log.NewBatchProcessor(exporter)))
	}

	provider := log.NewLoggerProvider(options...)

	// Register the global logger provider.
	global.SetLoggerProvider(provider)

	return provider
}

// Setup bootstraps the OpenTelemetry pipeline.
func Setup(ctx context.Context, options ...Variadic) (shutdown func(context.Context) error) {
	slog.DebugContext(ctx, "Starting the Telemetry Pipeline ...")

	o := Options()
	for _, option := range options {
		option(o)
	}

	var shutdowns []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var e error
		for _, fn := range shutdowns {
			e = errors.Join(e, fn(ctx))
		}

		shutdowns = nil
		return e
	}

	// Set up trace provider and add shutdown handler.
	shutdowns = append(shutdowns, traces(ctx, o).Shutdown)

	// Set up meter provider and add shutdown handler.
	shutdowns = append(shutdowns, metrics(ctx, o).Shutdown)

	// Set the global logger provider and add shutdown handler.
	shutdowns = append(shutdowns, logexporter(ctx, o).Shutdown)

	// Set up the global propagator.
	propagator(o)

	return
}
