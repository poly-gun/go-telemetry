package client

import (
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Options struct {
	Headers map[string]string
	Timeout time.Duration
	Name    string
	Level   slog.Level

	Attributes []attribute.KeyValue
}

func (o *Options) defaults() *Options {
	if o == nil {
		*o = Options{
			Headers:    make(map[string]string),
			Timeout:    15 * time.Second,
			Name:       "github.com/poly-gun/go-kubernetes-telemetry",
			Attributes: make([]attribute.KeyValue, 0),
			Level:      slog.LevelInfo,
		}
	}

	if o.Headers == nil {
		o.Headers = make(map[string]string)
	}

	if o.Timeout <= 0 {
		o.Timeout = 15 * time.Second
	}

	if o.Name == "" {
		o.Name = "github.com/poly-gun/go-kubernetes-telemetry"
	}

	if o.Attributes == nil {
		o.Attributes = make([]attribute.KeyValue, 0)
	}

	return o
}

type Client struct {
	client *http.Client

	options *Options
}

func New(settings ...func(o *Options)) *Client {
	options := new(Options).defaults()
	for _, setting := range settings {
		if setting != nil {
			setting(options)
		}
	}

	return &Client{
		client: &http.Client{
			Timeout: options.Timeout,
		},
		options: options,
	}
}

func (c *Client) Do(r *http.Request) (*http.Response, error) {
	if c == nil {
		*c = *New()
	}

	ctx := r.Context()
	attributes := c.options.Attributes
	ctx, span := trace.SpanFromContext(ctx).TracerProvider().Tracer(c.options.Name).Start(ctx, r.URL.String(), trace.WithSpanKind(trace.SpanKindClient), trace.WithTimestamp(time.Now()), trace.WithAttributes(attributes...))

	defer span.End()

	slog.Log(ctx, c.options.Level, "Log Message From HTTP Client Transport", slog.String("name", c.options.Name), slog.String("url", r.URL.String()))
	for key, value := range c.options.Headers {
		r.Header.Set(key, value)
	}

	return c.client.Do(r)
}
