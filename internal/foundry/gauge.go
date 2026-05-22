package foundry

import (
	"context"
	"log/slog"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
)

func (f *Foundry) Gauge(ctx context.Context, machinery v1alpha1.Machinery) error {
	p, err := f.newPlanner(ctx, machinery)
	if err != nil {
		return err
	}

	var unavailable []string
	for _, t := range p.Toolers() {
		if err := t.Gauge(ctx); err != nil {
			f.Logger.ErrorContext(ctx, "tool is not available or cannot be detected properly", slog.String("tool.name", t.Name()), foundryerrors.LogAttr(err))
			unavailable = append(unavailable, t.Name())
			continue
		}
		f.Logger.InfoContext(ctx, "tool is available", slog.String("tool.name", t.Name()))
	}
	if len(unavailable) > 0 {
		return foundryerrors.Newf(foundryerrors.TypeNotFound, "tools are not available, please install them and try again: %s", strings.Join(unavailable, ", "))
	}
	return nil
}
