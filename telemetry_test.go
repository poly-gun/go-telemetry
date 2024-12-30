package telemetry_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/poly-gun/go-telemetry"
)

func Test(t *testing.T) {
	const service = "test-service"
	const version = "0.0.0"

	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", fmt.Sprintf("service.name=%s,service.version=%s", service, version))

	t.Run("Telemetry-Initialization-Metrics", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource:   true,
			Level:       slog.LevelWarn,
			ReplaceAttr: nil,
		}))

		slog.SetDefault(logger)

		var traces, metrics, logs bytes.Buffer

		// Telemetry Setup + Cancellation Handler
		shutdown := telemetry.Setup(ctx, func(options *telemetry.Settings) {
			options.Zipkin.Enabled = false // disabled during testing

			options.Tracer = &telemetry.Tracer{
				Local:  true,
				Writer: &traces,
			}

			options.Metrics = &telemetry.Metrics{
				Local:  true,
				Writer: &metrics,
			}

			options.Logs = &telemetry.Logs{
				Local:  true,
				Writer: &logs,
			}
		})

		listener := telemetry.Interrupt(ctx, cancel, shutdown)

		t.Cleanup(func() {
			listener <- syscall.SIGTERM

			<-ctx.Done()
		})

		time.Sleep(10 * time.Second)

		if metrics.Len() == 0 {
			t.Error("No Metrics Received")
		} else {
			t.Logf("Metrics:\n%s", metrics.String())
		}
	})

	t.Run("Telemetry-HTTP-Handler", func(t *testing.T) {
		t.Setenv("OTEL_RESOURCE_ATTRIBUTES", fmt.Sprintf("service.name=%s,service.version=%s", service, version))

		ctx, cancel := context.WithCancel(context.Background())

		var traces, metrics, logs bytes.Buffer

		// Telemetry Setup + Cancellation Handler
		shutdown := telemetry.Setup(ctx, func(options *telemetry.Settings) {
			options.Zipkin.Enabled = false // disabled during testing

			options.Tracer = &telemetry.Tracer{
				Local:  true,
				Writer: &traces,
			}

			options.Metrics = &telemetry.Metrics{
				Local:  true,
				Writer: &metrics,
			}

			options.Logs = &telemetry.Logs{
				Local:  true,
				Writer: &logs,
			}
		})

		listener := telemetry.Interrupt(ctx, cancel, shutdown)

		t.Cleanup(func() {
			listener <- syscall.SIGTERM

			<-ctx.Done()
		})

		mux := http.NewServeMux()

		mux.Handle("GET /", otelhttp.WithRouteTag("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const name = "test-endpoint"

			ctx := r.Context()

			kind := trace.WithSpanKind(trace.SpanKindServer)
			links := trace.WithLinks(trace.LinkFromContext(ctx))
			attributes := []attribute.KeyValue{attribute.String("url", r.URL.String()), attribute.String("method", r.Method)}

			ctx, span := otel.Tracer(service).Start(ctx, name, kind, trace.WithAttributes(attributes...), links)
			labeler, _ := otelhttp.LabelerFromContext(ctx)

			defer span.End()

			logger := otelslog.NewLogger(name)

			labeler.Add(attribute.String("label", name))

			logger.InfoContext(ctx, "Test Endpoint Logger Message")

			datum := map[string]interface{}{
				"key-1": "value-1",
				"key-2": "value-2",
				"key-3": "value-3",
			}

			defer json.NewEncoder(w).Encode(datum)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			return
		})))

		// Add HTTP instrumentation for the whole server.
		handler := otelhttp.NewHandler(mux, "/")

		server := httptest.NewServer(handler)
		defer server.Close()

		client := server.Client()
		request, e := http.NewRequest(http.MethodGet, server.URL, nil)
		if e != nil {
			t.Fatalf("Unexpected Error While Generating Request: %v", e)
		}

		response, e := client.Do(request)
		if e != nil {
			t.Fatalf("Unexpected Error While Sending Request: %v", e)
		}

		defer response.Body.Close()

		time.Sleep(10 * time.Second)

		t.Run("Traces", func(t *testing.T) {
			if traces.Len() == 0 {
				t.Error("Traces Not Reported")
			} else {
				t.Logf("Traces:\n%s", traces.String())
			}
		})

		t.Run("Metrics", func(t *testing.T) {
			if metrics.Len() == 0 {
				t.Error("Metrics Not Reported")
			} else {
				t.Logf("Metrics:\n%s", metrics.String())
			}
		})

		t.Run("Logs", func(t *testing.T) {
			if logs.Len() == 0 {
				t.Error("Logs Not Reported")
			} else {
				t.Logf("Logs:\n%s", logs.String())
			}
		})
	})
}
