package dockercomposetooler

import (
	"context"

	root "github.com/signoz/foundry/internal/tooler"
)

var _ root.Tooler = (*dockerComposeTooler)(nil)

type dockerComposeTooler struct{}

func New() *dockerComposeTooler {
	return &dockerComposeTooler{}
}

func (tooler *dockerComposeTooler) Name() string {
	return "docker-compose"
}

func (tooler *dockerComposeTooler) Gauge(ctx context.Context) error {
	return root.AnyOneExecChecker(ctx, "docker-compose", "docker compose")
}

func (tooler *dockerComposeTooler) Install(ctx context.Context) error {
	return nil
}
