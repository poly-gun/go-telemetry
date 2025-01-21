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
)

// ctx, cancel represent the server's runtime context and cancellation handler.
var ctx, cancel = context.WithCancel(context.Background())

func main() {
    return
}

func init() {
    // Setup the telemetry pipeline and cancellation handler.
    shutdown := telemetry.Setup(ctx, func(o *telemetry.Settings) {
        if os.Getenv("CI") == "" { // Example of running the program in a local, development environment.
            o.Zipkin.Enabled = false

            o.Tracer.Local = true
            o.Tracer.Options = nil
            o.Tracer.Writer = io.Discard

            o.Metrics.Local = true
            o.Metrics.Options = nil
            o.Metrics.Writer = io.Discard

            o.Logs.Local = true
            o.Logs.Options = nil
            o.Logs.Writer = io.Discard
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
