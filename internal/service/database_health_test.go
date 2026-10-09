package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestDatabaseHealthCheckResults(t *testing.T) {
	for _, tc := range []struct {
		name   string
		probe  func(context.Context) error
		status string
		reason string
	}{
		{
			name: "healthy", status: "ok",
			probe: func(ctx context.Context) error {
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > databaseHealthTimeout {
					t.Error("health probe must receive a bounded context")
				}
				return nil
			},
		},
		{
			name: "failed query does not expose credentials", status: "error", reason: "query_failed",
			probe: func(context.Context) error {
				return errors.New("postgres://owner:PRIVATE_PASSWORD@private-db.example/postgres")
			},
		},
		{
			name: "timeout", status: "error", reason: "timeout",
			probe: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			checker := NewDatabaseHealthChecker(tc.probe, slog.New(slog.NewJSONHandler(&logs, nil)))
			if tc.reason == "timeout" {
				checker.timeout = 10 * time.Millisecond
			}
			checker.check(context.Background())
			var entry struct {
				Event         string `json:"event"`
				Status        string `json:"status"`
				Reason        string `json:"reason"`
				CheckedAt     string `json:"checked_at"`
				IntervalHours int    `json:"interval_hours"`
			}
			if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
				t.Fatalf("health result log: %v", err)
			}
			if entry.Event != "database_healthcheck" || entry.Status != tc.status || entry.Reason != tc.reason {
				t.Fatalf("unexpected health result: %+v", entry)
			}
			if entry.IntervalHours != 24 || entry.CheckedAt == "" {
				t.Fatalf("health result lacks daily schedule or timestamp: %+v", entry)
			}
			for _, secret := range []string{"PRIVATE_PASSWORD", "private-db.example", "postgres://"} {
				if strings.Contains(logs.String(), secret) {
					t.Fatalf("health log leaked connection details: %s", logs.String())
				}
			}
		})
	}
}

func TestDatabaseHealthLoopStartupAndScheduledRecovery(t *testing.T) {
	// Advance a virtual clock through a whole day to exercise the production
	// ticker without waiting 24 hours or changing the deployed schedule.
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var count atomic.Int32
		var logs bytes.Buffer
		checker := NewDatabaseHealthChecker(func(context.Context) error {
			if count.Add(1) == 1 {
				return errors.New("temporarily unavailable")
			}
			return nil
		}, slog.New(slog.NewJSONHandler(&logs, nil)))
		done := make(chan struct{})
		go func() {
			defer close(done)
			checker.Start(ctx)
		}()
		synctest.Wait()
		if count.Load() != 1 {
			t.Fatalf("startup probes: got %d, want 1", count.Load())
		}
		time.Sleep(databaseHealthInterval - time.Nanosecond)
		synctest.Wait()
		if count.Load() != 1 {
			t.Fatalf("probe ran before the next day: %d", count.Load())
		}
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		if count.Load() != 2 {
			t.Fatalf("probes after one day: got %d, want 2", count.Load())
		}
		cancel()
		synctest.Wait()
		select {
		case <-done:
		default:
			t.Fatal("health loop did not stop on shutdown")
		}
		if !strings.Contains(logs.String(), `"status":"error"`) || !strings.Contains(logs.String(), `"status":"ok"`) {
			t.Fatalf("health logs did not record failure and later recovery: %s", logs.String())
		}
	})
}

func TestDatabaseHealthLoopCancelledBeforeStartup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	checker := NewDatabaseHealthChecker(func(context.Context) error {
		t.Fatal("cancelled server must not query the database")
		return nil
	}, slog.Default())
	checker.run(ctx, make(chan time.Time))
}

func TestDatabaseHealthLoopShutdownCancelsInFlightProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	var logs bytes.Buffer
	checker := NewDatabaseHealthChecker(func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}, slog.New(slog.NewJSONHandler(&logs, nil)))
	done := make(chan struct{})
	go func() {
		defer close(done)
		checker.run(ctx, make(chan time.Time))
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("startup probe did not start")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not cancel the database probe")
	}
	if logs.Len() != 0 {
		t.Fatalf("normal shutdown was reported as a database failure: %s", logs.String())
	}
}
