package kubectltooler

import (
	"context"

	root "github.com/signoz/foundry/internal/tooler"
)

var _ root.Tooler = (*kubectlTooler)(nil)

type kubectlTooler struct{}

func New() *kubectlTooler {
	return &kubectlTooler{}
}

func (t *kubectlTooler) Name() string {
	return "kubectl"
}

func (t *kubectlTooler) Gauge(ctx context.Context) error {
	return root.ExecChecker(ctx, "kubectl")
}

func (t *kubectlTooler) Install(ctx context.Context) error {
	return nil
}
