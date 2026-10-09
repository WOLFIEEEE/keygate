package store

import (
	"context"
	"errors"
)

// CheckDatabaseHealth verifies that a real query can read the application's
// schema. Reading the migration table also works with a read-only database and
// does not create health-check rows or access customer records.
func (s *Store) CheckDatabaseHealth(ctx context.Context) error {
	var migrated bool
	if err := s.DB.NewRaw("SELECT EXISTS (SELECT 1 FROM schema_migrations LIMIT 1)").Scan(ctx, &migrated); err != nil {
		return err
	}
	if !migrated {
		return errors.New("database has no applied application migrations")
	}
	return nil
}
