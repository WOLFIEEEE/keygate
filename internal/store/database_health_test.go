package store_test

import (
	"context"
	"errors"
	"testing"
)

func TestCheckDatabaseHealth(t *testing.T) {
	s := setupTestDB(t)
	defer s.Close()
	ctx := context.Background()
	// Keep all queries on this private pool's single connection so the session
	// read-only setting covers the health check too.
	s.DB.SetMaxOpenConns(1)
	s.DB.SetMaxIdleConns(1)
	if _, err := s.DB.ExecContext(ctx, "SET default_transaction_read_only = on"); err != nil {
		t.Fatalf("read-only connection: %v", err)
	}
	if err := s.CheckDatabaseHealth(ctx); err != nil {
		t.Fatalf("health check must work without writes: %v", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.CheckDatabaseHealth(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled check: got %v, want context.Canceled", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close connection: %v", err)
	}
	if err := s.CheckDatabaseHealth(ctx); err == nil {
		t.Fatal("closed database was reported healthy")
	}
}
