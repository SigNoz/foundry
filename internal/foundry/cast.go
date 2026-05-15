package foundry

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
)

func (foundry *Foundry) Cast(ctx context.Context, machinery v1alpha1.Machinery, poursPath string) error {
	switch c := machinery.(type) {
	case *installation.Casting:
		return foundry.castInstallation(ctx, *c, poursPath)
	}
	return fmt.Errorf("unsupported casting kind %q", machinery.Kind())
}

func (foundry *Foundry) castInstallation(ctx context.Context, config installation.Casting, poursPath string) error {
	foundry.Logger.InfoContext(ctx, "starting cast pipeline", slog.String("casting.metadata.name", config.Metadata.Name))

	spec := &config.Spec

	casting, err := foundry.Registry.Casting(spec.Deployment)
	if err != nil {
		return err
	}

	err = casting.Cast(ctx, config, poursPath)
	if err != nil {
		foundry.Logger.ErrorContext(ctx, err.Error())
		return err
	}

	return nil
}
