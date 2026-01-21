package clickhousekeepertooler

import (
	"context"
	"fmt"
	"os"

	root "github.com/signoz/foundry/internal/tooler"
)

var _ root.Tooler = (*clickhouseKeeperTooler)(nil)

type clickhouseKeeperTooler struct{}

func New() *clickhouseKeeperTooler {
	return &clickhouseKeeperTooler{}
}

func (tooler *clickhouseKeeperTooler) Name() string {
	return "clickhouse-keeper"
}

func (tooler *clickhouseKeeperTooler) Gauge(ctx context.Context) error {
	// Check if clickhouse-keeper command is available
	if err := root.ExecChecker(ctx, "clickhouse-keeper"); err == nil {
		return nil
	}

	// Fallback: check if the binary exists at the standard location
	binaryPath := "/usr/bin/clickhouse-keeper"
	if _, err := os.Stat(binaryPath); err == nil {
		return nil
	}

	return fmt.Errorf("clickhouse-keeper not found: neither command nor binary at %s", binaryPath)
}

func (tooler *clickhouseKeeperTooler) Install(ctx context.Context) error {
	// ClickHouse Keeper is typically installed via package manager
	// Installation instructions would depend on the OS distribution
	return nil
}
