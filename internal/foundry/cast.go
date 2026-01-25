package foundry

import (
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
)

func (foundry *Foundry) Cast(ctx context.Context, config v1alpha1.Casting, poursPath string) error {
	foundry.Logger.InfoContext(ctx, "starting cast pipeline", slog.String("casting.metadata.name", config.Metadata.Name))

	// Get the casting for the deployment mode
	casting, err := foundry.Registry.Casting(config.Spec.Deployment)
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
