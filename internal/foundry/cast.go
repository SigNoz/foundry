package foundry

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/signoz/foundry/api/v1alpha1"
)

func (foundry *Foundry) Cast(ctx context.Context, config v1alpha1.Casting, outputPath string) error {
	foundry.Logger.InfoContext(ctx, "starting casting pipeline", slog.String("casting.metadata.name", config.Metadata.Name))

	// Get the casting for the deployment mode
	casting, err := foundry.CastingByDeploymentMode(config.Spec.Deployment.Mode)
	if err != nil {
		return err
	}

	// Check if the pours directory was created by forge before
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("pours directory does not exist at path: %s. Please run forge before cast", outputPath)
	}

	err = casting.Cast(ctx, config, outputPath)
	if err != nil {
		foundry.Logger.ErrorContext(ctx, err.Error())
		return err
	}

	return nil
}
