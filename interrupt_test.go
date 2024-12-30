package telemetry_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/poly-gun/go-telemetry"
)

func TestInterrupt(t *testing.T) {
	const service = "test-service"
	const version = "0.0.0"

	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", fmt.Sprintf("service.name=%s,service.version=%s", service, version))

	t.Run("Telemetry-Graceful-Shutdown", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource:   true,
			Level:       slog.LevelDebug,
			ReplaceAttr: nil,
		}))

		slog.SetDefault(logger)

		// Telemetry Setup + Cancellation Handler
		shutdown := telemetry.Setup(ctx, func(options *telemetry.Settings) {
			options.Zipkin.Enabled = false // disabled during testing

			options.Tracer.Local = true
			options.Metrics.Local = true
			options.Logs.Local = true

			options.Metrics.Writer = io.Discard // prevent output from filling the test logs
		})

		listener := telemetry.Interrupt(ctx, cancel, shutdown)

		time.Sleep(5 * time.Second)

		listener <- syscall.SIGTERM

		<-ctx.Done()
	})
}

func ExampleInterrupt() {
	const service = "example-service"
	const version = "0.0.0"

	_ = os.Setenv("OTEL_RESOURCE_ATTRIBUTES", fmt.Sprintf("service.name=%s,service.version=%s", service, version))

	ctx, cancel := context.WithCancel(context.Background())

	// Telemetry Setup + Cancellation Handler
	shutdown := telemetry.Setup(ctx, func(options *telemetry.Settings) {
		options.Zipkin.Enabled = false // disabled during testing

		options.Tracer.Local = true
		options.Metrics.Local = true
		options.Logs.Local = true

		options.Metrics.Writer = io.Discard // prevent output from filling the test logs
	})

	listener := telemetry.Interrupt(ctx, cancel, shutdown)

	time.Sleep(5 * time.Second)

	listener <- syscall.SIGTERM

	<-ctx.Done()

	fmt.Println("Telemetry Shutdown Complete")

	// Output:
	// Telemetry Shutdown Complete
}
