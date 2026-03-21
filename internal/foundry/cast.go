package foundry

import (
	"context"
	"fmt"

	"github.com/signoz/foundry/api/v1alpha1"
)

func (foundry *Foundry) Cast(ctx context.Context, config v1alpha1.Casting, poursPath string) error {
	foundry.UX.Header(fmt.Sprintf("Casting %s (%s/%s)", config.Metadata.Name, config.Spec.Deployment.Mode, config.Spec.Deployment.Flavor))

	// Get the casting for the deployment mode
	casting, err := foundry.Registry.Casting(config.Spec.Deployment)
	if err != nil {
		return err
	}

	foundry.UX.StartStep("Deploying")
	err = casting.Cast(ctx, config, poursPath)
	if err != nil {
		foundry.UX.FinishStep("Deploying", err)
		foundry.Logger.ErrorContext(ctx, err.Error())
		return err
	}
	foundry.UX.FinishStep("Deployed successfully", nil)

	return nil
}
