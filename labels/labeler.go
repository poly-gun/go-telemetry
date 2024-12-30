package labels

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Options defines configuration options including logging level settings.
type Options struct {
	// Attributes is an optional map[string]string to use when generating the [Labeler] function's [slog.Log]-related [slog.String] message(s).
	//
	// 	- The default is an empty map.
	Attributes map[string]string

	// Level specifies the logging level for controlling the verbosity of log output in the configuration options.
	//
	// 	- The default value is [slog.LevelWarn].
	Level slog.Level
}

// defaults initializes and returns a default Options instance with predefined configuration settings.
func defaults() *Options {
	return &Options{
		Attributes: map[string]string{},
		Level:      slog.LevelWarn,
	}
}

// Labeler retrieves an [otelhttp.Labeler] from the given context or logs a message if none exists, using optional configuration [Options].
func Labeler(ctx context.Context, settings ...func(options *Options)) *otelhttp.Labeler {
	// Construct the options configuration.
	options := defaults()
	for _, setting := range settings {
		if setting != nil {
			setting(options)
		}
	}

	labeler, found := otelhttp.LabelerFromContext(ctx)
	if !(found) {
		attributes := make([]slog.Attr, 0)
		for k, v := range options.Attributes {
			attributes = append(attributes, slog.String(k, v))
		}

		slog.LogAttrs(ctx, options.Level, "No Labeler Found in Context - Any Labeler Attributes Will be Superfluous", attributes...)
	}

	return labeler
}
