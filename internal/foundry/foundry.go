package foundry

import (
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/config"
	"github.com/signoz/foundry/internal/config/yamlconfig"
	"github.com/signoz/foundry/internal/patch"
	"github.com/signoz/foundry/internal/patch/jsonpatch"
)

// Foundry holds only shared facilities. Per-Kind components (registries,
// moldings, casting strategies, enrichers) live behind the planner abstraction
// in planner.go and are constructed on demand from a Machinery.
type Foundry struct {
	Config   config.Config
	Patchers map[string]patch.Patch
	Logger   *slog.Logger
}

func New(logger *slog.Logger) (*Foundry, error) {
	return &Foundry{
		Config: yamlconfig.New(),
		Patchers: map[string]patch.Patch{
			v1alpha1.PatchTypeJSONPatch: jsonpatch.New(),
		},
		Logger: logger,
	}, nil
}
