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
	return nil
}
