package systemdtooler

import (
	"context"

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
	return root.ExecChecker(ctx, "systemctl")
}

func (tooler *systemdTooler) Install(ctx context.Context) error {
	return nil
}

func (tooler *systemdTooler) InstallHint() string {
	return "systemd is a system package — ensure you are on a systemd-based Linux distribution"
}
