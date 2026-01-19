package postgrestooler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	root "github.com/signoz/foundry/internal/tooler"
)

var _ root.Tooler = (*postgresTooler)(nil)

type postgresTooler struct{}

func New() *postgresTooler {
	return &postgresTooler{}
}

func (tooler *postgresTooler) Name() string {
	return "postgres"
}

func (tooler *postgresTooler) Gauge(ctx context.Context) error {
	// Check if postgres command is available
	if err := root.ExecChecker(ctx, "postgres"); err == nil {
		return nil
	}

	// Check if psql command is available (common PostgreSQL client, often installed with server)
	if err := root.ExecChecker(ctx, "psql"); err == nil {
		return nil
	}

	// Fallback: check common binary locations
	commonPaths := []string{
		"/usr/local/pgsql/bin/postgres",
		"/usr/bin/postgres",
		"/usr/lib/postgresql/*/bin/postgres",
	}

	for _, path := range commonPaths {
		// Handle glob patterns
		if matches, err := filepath.Glob(path); err == nil && len(matches) > 0 {
			for _, match := range matches {
				if _, err := os.Stat(match); err == nil {
					return nil
				}
			}
		} else if _, err := os.Stat(path); err == nil {
			return nil
		}
	}

	return fmt.Errorf("postgres not found: neither command nor binary in common locations")
}

func (tooler *postgresTooler) Install(ctx context.Context) error {
	// PostgreSQL is typically installed via package manager
	// Installation instructions would depend on the OS distribution
	return nil
}
