package telemetry_test

import (
	"context"
	"os"
	"time"

	"github.com/poly-gun/go-telemetry"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ctx, cancel represent the server's runtime context and cancellation handler.
var ctx, cancel = context.WithCancel(context.Background())

func Example() {
	defer cancel() // Eventually stop the open-telemetry client.

	ctx, span := otel.Tracer("example").Start(ctx, "main", trace.WithSpanKind(trace.SpanKindUnspecified))

	_ = ctx // Real implementation is likely to make use of the ctx.

	// Typical use case of the span would be to defer span.End() after initialization; however, in the example, we need to
	// control when it ends in order to capture the output and write it out as the example.

	// defer span.End()

	// Add an event (in many observability tools, this gets represented as a log message).
	span.AddEvent("example-event-log-1", trace.WithAttributes(attribute.String("message", "Hello World")))

	span.End()

	time.Sleep(5 * time.Second)

	// The output will include metrics and trace message(s) in JSON format, printed to standard-output.
	// Output:
}

func init() {
	// Setup the telemetry pipeline and cancellation handler.
	shutdown := telemetry.Setup(ctx, func(o *telemetry.Settings) {
		if os.Getenv("CI") == "" { // Example of running the program in a local, development environment.
			o.Zipkin.Enabled = false

			o.Tracer.Local = true
			o.Tracer.Options = nil
			o.Tracer.Writer = os.Stdout

			o.Metrics.Local = true
			o.Metrics.Options = nil
			o.Metrics.Writer = os.Stdout

			o.Logs.Local = true
			o.Logs.Options = nil
			o.Logs.Writer = os.Stdout
		} else {
			o.Zipkin.URL = "http://zipkin.istio-system.svc.cluster.local:9411"
		}
	})

	// Initialize the telemetry interrupt handler.
	telemetry.Interrupt(ctx, cancel, shutdown)
}
