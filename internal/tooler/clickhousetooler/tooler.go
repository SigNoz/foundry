package clickhousetooler

import (
	"context"
	"fmt"
	"os"

	root "github.com/signoz/foundry/internal/tooler"
)

var _ root.Tooler = (*clickhouseTooler)(nil)

type clickhouseTooler struct{}

func New() *clickhouseTooler {
	return &clickhouseTooler{}
}

func (tooler *clickhouseTooler) Name() string {
	return "clickhouse"
}

func (tooler *clickhouseTooler) Gauge(ctx context.Context) error {
	// Check if clickhouse-server command is available
	if err := root.ExecChecker(ctx, "clickhouse-server"); err == nil {
		return nil
	}

	// Fallback: check if the binary exists at the standard location
	binaryPath := "/usr/bin/clickhouse-server"
	if _, err := os.Stat(binaryPath); err == nil {
		return nil
	}

	return fmt.Errorf("clickhouse-server not found: neither command nor binary at %s", binaryPath)
}

func (tooler *clickhouseTooler) Install(ctx context.Context) error {
	// ClickHouse is typically installed via package manager
	// Installation instructions would depend on the OS distribution
	return nil
}
