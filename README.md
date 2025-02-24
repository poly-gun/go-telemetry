# `go-telemetry` - OTEL HTTP Telemetry

## Documentation

Official `godoc` documentation (with examples) can be found at the [Package Registry](https://pkg.go.dev/github.com/poly-gun/go-telemetry).

## Usage

###### Add Package Dependency

```bash
go get -u github.com/poly-gun/go-telemetry
```

###### Import and Implement

`main.go`

```go
package main

import (
    "context"
    "os"
    "io"

    "github.com/poly-gun/go-telemetry"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

// ctx, cancel represent the server's runtime context and cancellation handler.
var ctx, cancel = context.WithCancel(context.Background())

func main() {
    defer cancel() // eventually stop the open-telemetry client.

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
```

- Please refer to the [code examples](./example_test.go) for additional usage and implementation details.
- See https://pkg.go.dev/github.com/poly-gun/go-telemetry for additional documentation.

## Contributions

See the [**Contributing Guide**](./CONTRIBUTING.md) for additional details on getting started.

## Task-Board

- [ ] Create a Resource Detector for Kubernetes Telemetry.
