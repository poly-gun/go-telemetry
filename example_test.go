package telemetry_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/poly-gun/go-telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ctx, cancel represent the server's runtime context and cancellation handler.
var ctx, cancel = context.WithCancel(context.Background())

// mock represents the type for capturing metrics, tracing, and log-related telemetry output. See [example] for implementation details.
type mock struct {
	tracing *bytes.Buffer
	metrics *bytes.Buffer
	logs    *bytes.Buffer
}

// For example output capturing purposes only - example presents a series of buffers that allow capture of metrics, tracing, and log-related telemetry output.
var example = mock{
	tracing: &bytes.Buffer{},
	metrics: &bytes.Buffer{},
	logs:    &bytes.Buffer{},
}

// capture represents a very small part of the larger telemetry trace message.
type capture struct {
	Name   string `json:"Name"`
	Events []struct {
		Name       string `json:"Name"`
		Attributes []struct {
			Key   string `json:"Key"`
			Value struct {
				Type  string `json:"Type"`
				Value string `json:"Value"`
			} `json:"Value"`
		} `json:"Attributes"`
	} `json:"Events"`
}

// handler represents a non-pure function that returns a string "hello world" after simulating processing via a 5-second timer.
func handler(ctx context.Context) string {
	time.Sleep(1 * time.Second)

	return "hello world"
}

func Example() {
	ctx, span := otel.Tracer("example").Start(ctx, "main", trace.WithSpanKind(trace.SpanKindUnspecified))

	// Typical use case of the span would be to defer span.End() after initialization; however, in the example, we need to
	// control when it ends in order to capture the output and write it out as the example.

	// defer span.End()

	// Initialize a result that simulates an operation.
	result := handler(ctx)

	// Add an event (in many observability tools, this gets represented as a log message), using the result as the message's content.
	span.AddEvent("example-event-log-1", trace.WithAttributes(attribute.String("message", result)))

	span.End()

	time.Sleep(5 * time.Second)

	var instance capture
	if e := json.Unmarshal(example.tracing.Bytes(), &instance); e != nil {
		panic(e)
	}

	fmt.Printf("Name: %s\n", instance.Name)

	fmt.Printf("Message: %s\n", instance.Events[0].Attributes[0].Value.Value)

	// Output:
	// Name: main
	// Message: hello world
}

func init() {
	// Setup the telemetry pipeline and cancellation handler.
	shutdown := telemetry.Setup(ctx, func(o *telemetry.Settings) {
		if os.Getenv("CI") == "" { // Example of running the program in a local, development environment.
			o.Zipkin.Enabled = false

			o.Tracer.Local = true
			o.Tracer.Options = nil
			o.Tracer.Writer = example.tracing // os.Stdout

			o.Metrics.Local = true
			o.Metrics.Options = nil
			o.Metrics.Writer = example.metrics // os.Stdout

			o.Logs.Local = true
			o.Logs.Options = nil
			o.Logs.Writer = example.logs // os.Stdout
		} else {
			o.Zipkin.URL = "http://zipkin.istio-system.svc.cluster.local:9411"
		}
	})

	// Initialize the telemetry interrupt handler.
	telemetry.Interrupt(ctx, cancel, shutdown)
}
