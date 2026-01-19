package systemdtooler

import (
	"context"
	"fmt"
	"os"

	root "github.com/signoz/foundry/internal/tooler"
)

var _ root.Tooler = (*systemdTooler)(nil)

type systemdTooler struct{}

func New() *systemdTooler {
	return &systemdTooler{}
}

func (tooler *systemdTooler) Name() string {
	return "systemd"
}

func (tooler *systemdTooler) Gauge(ctx context.Context) error {
	// Check if systemctl command is available
	if err := root.ExecChecker(ctx, "systemctl"); err != nil {
		return fmt.Errorf("systemctl command not found: %w", err)
	}

	// Check if systemd system directory exists
	systemDir := "/etc/systemd/system"
	if _, err := os.Stat(systemDir); os.IsNotExist(err) {
		return fmt.Errorf("systemd system directory does not exist: %s", systemDir)
	}

	return nil
}

func (tooler *systemdTooler) Install(ctx context.Context) error {
	return nil
}
