package azureclitooler

import (
	"context"

	root "github.com/signoz/foundry/internal/tooler"
)

var _ root.Tooler = (*azureCliTooler)(nil)

type azureCliTooler struct{}

func New() *azureCliTooler {
	return &azureCliTooler{}
}

func (t *azureCliTooler) Name() string {
	return "az"
}

func (t *azureCliTooler) Gauge(ctx context.Context) error {
	return root.ExecChecker(ctx, "az")
}

func (t *azureCliTooler) Install(ctx context.Context) error {
	return nil
}
