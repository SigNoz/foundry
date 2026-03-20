package helmtooler

import (
	"context"

	root "github.com/signoz/foundry/internal/tooler"
)

var _ root.Tooler = (*helmTooler)(nil)

type helmTooler struct{}

func New() *helmTooler {
	return &helmTooler{}
}

func (tooler *helmTooler) Name() string {
	return "helm"
}

func (tooler *helmTooler) Gauge(ctx context.Context) error {
	return root.ExecChecker(ctx, "helm")
}

func (tooler *helmTooler) Install(ctx context.Context) error {
	return nil
}
