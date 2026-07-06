package binarytooler

import (
	"context"
	"os"

	"github.com/signoz/foundry/internal/errors"
	root "github.com/signoz/foundry/internal/tooler"
)

var _ root.Tooler = (*binaryTooler)(nil)

// binaryTooler checks that a named binary exists at a resolved path. The path is
// supplied at construction (resolved from casting annotations with a fallback by
// the caller), so the tooler itself stays config-blind: it only verifies that
// the binary the deployment will exec is actually present.
type binaryTooler struct {
	name string
	path string
}

func New(name, path string) *binaryTooler {
	return &binaryTooler{name: name, path: path}
}

func (tooler *binaryTooler) Name() string {
	return tooler.name
}

func (tooler *binaryTooler) Gauge(ctx context.Context) error {
	if _, err := os.Stat(tooler.path); err != nil {
		return errors.Newf(errors.TypeNotFound, "%s binary not found at %q", tooler.name, tooler.path)
	}
	return nil
}

func (tooler *binaryTooler) Install(ctx context.Context) error {
	return nil
}
