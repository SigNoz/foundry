package terraformtooler

import (
	"context"

	root "github.com/signoz/foundry/internal/tooler"
)

var _ root.Tooler = (*terraformTooler)(nil)

type terraformTooler struct{}

func New() *terraformTooler {
	return &terraformTooler{}
}

func (t *terraformTooler) Name() string {
	return "terraform"
}

func (t *terraformTooler) Gauge(ctx context.Context) error {
	return root.ExecChecker(ctx, "terraform")
}

func (t *terraformTooler) Install(ctx context.Context) error {
	// Terraform installation is platform-specific and typically requires manual installation
	// or use of a package manager. We return nil here as users are expected to have
	// terraform installed.
	return nil
}
