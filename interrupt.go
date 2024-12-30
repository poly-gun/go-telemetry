package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/term"
)

// Interrupt is a graceful interrupt + signal handler for the telemetry pipeline.
func Interrupt(ctx context.Context, cancel context.CancelFunc, shutdown func(context.Context) error) chan os.Signal {
	// Listen for syscall signals for process to interrupt/quit.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-interrupt

		if term.IsTerminal(int(os.Stdout.Fd())) {
			fmt.Print("\r")
		}

		slog.DebugContext(ctx, "Initializing Telemetry Pipeline Shutdown ...")

		// Shutdown signal with grace period of 30 seconds.
		handler, timeout := context.WithTimeout(ctx, 30*time.Second)
		defer timeout()
		go func() {
			<-handler.Done()
			if errors.Is(handler.Err(), context.DeadlineExceeded) {
				slog.Log(ctx, slog.LevelError, "Graceful Telemetry Pipeline Shutdown Timeout - Forcing an Exit ...")

				os.Exit(124) // For portability, 134 cannot be used.
			}
		}()

		// Trigger graceful shutdown.
		if e := shutdown(handler); e != nil {
			slog.ErrorContext(ctx, "Exception During Telemetry Pipeline Shutdown", slog.String("error", e.Error()))
		}

		slog.InfoContext(ctx, "Telemetry Pipeline Shutdown Complete")

		cancel()
	}()

	return interrupt
}
