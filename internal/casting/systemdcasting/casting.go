package systemdcasting

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/casting"
)

var _ casting.Casting = (*linuxCasting)(nil)

type linuxCasting struct {
	logger   *slog.Logger
	castings []*types.Template
}

func New(logger *slog.Logger) *linuxCasting {
	return &linuxCasting{
		logger: logger,
		castings: []*types.Template{
			telemetryKeeperServiceTemplate,
			telemetryStoreServiceTemplate,
			metaStoreServiceTemplate,
			signozServiceTemplate,
			ingesterServiceTemplate,
		},
	}
}

func (casting *linuxCasting) Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error) {
	return newLinuxMoldingEnricher(config)
}

func (casting *linuxCasting) Forge(ctx context.Context, config v1alpha1.Casting) ([]types.Material, error) {
	// execute service templates

	return []types.Material{}, nil
}

func (casting *linuxCasting) Cast(ctx context.Context, config v1alpha1.Casting, outputPath string) error {
	casting.logger.InfoContext(ctx, "Executing commands for platform")

	// Create a context with 5-minute timeout
	runctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Join commands with && to run in sequence
	//command := strings.Join(cast.Execute, " && ")
	command := ""

	casting.logger.DebugContext(runctx, "Running command", slog.String("command", command))

	cmd := exec.CommandContext(runctx, "sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		casting.logger.ErrorContext(runctx, "Command execution failed", slog.String("error", err.Error()))
		return err
	}

	casting.logger.InfoContext(runctx, "Command executed successfully")
	return nil

}

func getServiceMaterial(config *v1alpha1.Casting, path string) (types.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := signozServiceTemplate.Execute(buf, config)
	if err != nil {
		return types.Material{}, fmt.Errorf("failed to execute signoz service template: %w", err)
	}

	return types.NewYAMLMaterial(buf.Bytes(), path)
}
