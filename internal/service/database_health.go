package service

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

const (
	databaseHealthInterval = 24 * time.Hour
	databaseHealthTimeout  = 10 * time.Second
)

// DatabaseHealthChecker reads the application's database at startup and once
// every 24 hours while the server is running. It uses the existing connection;
// no Supabase API key or external scheduler is required.
type DatabaseHealthChecker struct {
	probe   func(context.Context) error
	logger  *slog.Logger
	timeout time.Duration
}

func NewDatabaseHealthChecker(probe func(context.Context) error, logger *slog.Logger) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{probe: probe, logger: logger, timeout: databaseHealthTimeout}
}

func (h *DatabaseHealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(databaseHealthInterval)
	defer ticker.Stop()
	h.run(ctx, ticker.C)
}

func (h *DatabaseHealthChecker) run(ctx context.Context, ticks <-chan time.Time) {
	for {
		if ctx.Err() != nil {
			return
		}
		h.check(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticks:
		}
	}
}

func (h *DatabaseHealthChecker) check(ctx context.Context) {
	started := time.Now()
	checkCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()
	err := h.probe(checkCtx)
	if ctx.Err() != nil {
		return // Normal shutdown is not a database failure.
	}
	if err == nil {
		err = checkCtx.Err()
	}
	fields := []any{
		"event", "database_healthcheck",
		"checked_at", started.UTC().Format(time.RFC3339),
		"duration_ms", time.Since(started).Milliseconds(),
		"interval_hours", int(databaseHealthInterval / time.Hour),
	}
	if err != nil {
		reason := "query_failed"
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(checkCtx.Err(), context.DeadlineExceeded) {
			reason = "timeout"
		}
		// Driver errors can contain connection credentials. Log a safe category,
		// never the raw error or database URL.
		h.logger.Error("database health check failed", append(fields, "status", "error", "reason", reason)...)
		return
	}
	h.logger.Info("database health check passed", append(fields, "status", "ok")...)
}
