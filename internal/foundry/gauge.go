package foundry

import (
	"context"
	"fmt"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/ux"
)

func (foundry *Foundry) Gauge(ctx context.Context, config v1alpha1.Casting) error {
	foundry.UX.Header(fmt.Sprintf("Gauging tools for %s (%s/%s)", config.Metadata.Name, config.Spec.Deployment.Mode, config.Spec.Deployment.Flavor))

	toolers, err := foundry.Registry.Toolers(config.Spec.Deployment)
	if err != nil {
		return err
	}

	var unavailable []ux.MissingTool

	for _, tooler := range toolers {
		foundry.UX.StartStep(fmt.Sprintf("Checking %s", tooler.Name()))
		err := tooler.Gauge(ctx)
		if err != nil {
			foundry.UX.FinishStep(fmt.Sprintf("%s not available", tooler.Name()), err)
			unavailable = append(unavailable, ux.MissingTool{
				Name:        tooler.Name(),
				InstallHint: tooler.InstallHint(),
			})
			continue
		}

		foundry.UX.FinishStep(fmt.Sprintf("%s available", tooler.Name()), nil)
	}

	if len(unavailable) > 0 {
		foundry.UX.PrintMissingTools(unavailable)

		names := make([]string, len(unavailable))
		for i, t := range unavailable {
			names[i] = t.Name
		}
		return fmt.Errorf("tools are not available, please install them and try again: %s", strings.Join(names, ", "))
	}

	return nil
}
