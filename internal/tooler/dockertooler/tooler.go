package dockertooler

import (
	"context"

	root "github.com/signoz/foundry/internal/tooler"
)

var _ root.Tooler = (*dockerTooler)(nil)

type dockerTooler struct{}

func New() *dockerTooler {
	return &dockerTooler{}
}

func (tooler *dockerTooler) Name() string {
	return "docker"
}

func (tooler *dockerTooler) Gauge(ctx context.Context) error {
	return root.ExecChecker(ctx, "docker")
}

func (tooler *dockerTooler) Install(ctx context.Context) error {
	return nil
}
