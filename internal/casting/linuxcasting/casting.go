package linuxcasting

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/molding/ingestermolding"
	"github.com/signoz/foundry/internal/molding/telemetrystoremolding"
	"github.com/signoz/foundry/internal/types"
)

var _ casting.Casting = (*linuxCasting)(nil)

type linuxCasting struct {
	logger   *slog.Logger
	castings []*types.Template
}

// serviceConfig holds the configuration for generating service materials.
type serviceConfig struct {
	enabled     bool
	serviceName string
	configFiles map[string]string
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
	var materials []types.Material

	for _, tmpl := range casting.castings {
		m, err := casting.forgeCasting(tmpl, &config)
		if err != nil {
			return nil, err
		}
		materials = append(materials, m...)
	}

	return materials, nil
}

func (casting *linuxCasting) Cast(ctx context.Context, config v1alpha1.Casting) error {
	casting.logger.InfoContext(ctx, "Executing commands for platform")

	// Create a context with 5-minute timeout
	runctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

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

func executeServiceTemplate(template types.Template, config *v1alpha1.Casting, path string) (types.Material, error) {
	buf := bytes.NewBuffer(nil)
	err := template.Execute(buf, config)
	if err != nil {
		return types.Material{}, fmt.Errorf("failed to execute template: %w", err)
	}

	return types.NewSystemdMaterial(buf.Bytes(), path)
}

// getReplicaCount returns the replica count with a default of 1.
func getReplicaCount(replicas int) int {
	return max(replicas, 1)
}

// createConfigMaterials creates YAML materials from config data map.
func createConfigMaterials(configData map[string]string) ([]types.Material, error) {
	var materials []types.Material
	for filename, content := range configData {
		m, err := types.NewYAMLMaterial([]byte(content), filename)
		if err != nil {
			return nil, fmt.Errorf("failed to create config material for %s: %w", filename, err)
		}
		materials = append(materials, m)
	}
	return materials, nil
}

// createServiceMaterials is a helper that generates service and config materials.
func createServiceMaterials(tmpl types.Template, config *v1alpha1.Casting, svcCfg serviceConfig) ([]types.Material, error) {
	if !svcCfg.enabled {
		return nil, nil
	}

	// Create service material
	material, err := executeServiceTemplate(tmpl, config, svcCfg.serviceName+".service")
	if err != nil {
		return nil, fmt.Errorf("failed to get service material for %s: %w", svcCfg.serviceName, err)
	}

	// Create config materials
	configMaterials, err := createConfigMaterials(svcCfg.configFiles)
	if err != nil {
		return nil, err
	}

	return append([]types.Material{material}, configMaterials...), nil
}

func (casting *linuxCasting) forgeCasting(tmpl *types.Template, config *v1alpha1.Casting) ([]types.Material, error) {
	switch tmpl {
	case signozServiceTemplate:
		serviceName := fmt.Sprintf("%s-signoz", config.Metadata.Name)
		return createServiceMaterials(*tmpl, config, serviceConfig{
			enabled:     config.Spec.Signoz.Spec.Enabled,
			serviceName: serviceName,
			configFiles: config.Spec.Signoz.Spec.Config.Data,
		})

	case ingesterServiceTemplate:
		if config.Spec.Ingester.Status.Extras == nil {
			config.Spec.Ingester.Status.Extras = make(map[string]string)
		}
		metaDataName := config.Metadata.Name
		config.Spec.Ingester.Status.Extras["cfgPath"] = fmt.Sprintf(ingestermolding.IngesterConfigFileFormat, v1alpha1.MoldingKindIngester.String(), metaDataName, ingestermolding.ConfigV0129xTemplate.String())
		config.Spec.Ingester.Status.Extras["cfgOpampPath"] = fmt.Sprintf(ingestermolding.IngesterOpampFileFormat, v1alpha1.MoldingKindIngester.String(), metaDataName, ingestermolding.OpampV0129xTemplate)
		serviceName := fmt.Sprintf("%s-ingester", config.Metadata.Name)
		return createServiceMaterials(*tmpl, config, serviceConfig{
			enabled:     config.Spec.Ingester.Spec.Enabled,
			serviceName: serviceName,
			configFiles: config.Spec.Ingester.Spec.Config.Data,
		})

	case telemetryStoreServiceTemplate:
		return casting.forgeTelemetryStore(tmpl, config)

	case telemetryKeeperServiceTemplate:
		return casting.forgeTelemetryKeeper(tmpl, config)

	case metaStoreServiceTemplate:
		serviceName := fmt.Sprintf("%s-metastore-%s-0-0", config.Metadata.Name, config.Spec.MetaStore.Kind.String())
		return createServiceMaterials(*tmpl, config, serviceConfig{
			enabled:     config.Spec.MetaStore.Spec.Enabled,
			serviceName: serviceName,
			configFiles: config.Spec.MetaStore.Spec.Config.Data,
		})
	}

	return nil, nil
}

func (casting *linuxCasting) forgeTelemetryStore(tmpl *types.Template, config *v1alpha1.Casting) ([]types.Material, error) {
	if !config.Spec.TelemetryStore.Spec.Enabled {
		return nil, nil
	}

	spec := &config.Spec.TelemetryStore.Spec
	storeKind := config.Spec.TelemetryStore.Kind.String()
	replicas := getReplicaCount(*spec.Cluster.Replicas + 1)
	shards := getReplicaCount(*spec.Cluster.Shards)
	metaDataName := config.Metadata.Name
	// Create config materials
	materials, err := createConfigMaterials(spec.Config.Data)
	if err != nil {
		return nil, err
	}
	if config.Spec.TelemetryStore.Status.Extras == nil {
		config.Spec.TelemetryStore.Status.Extras = make(map[string]string)
	}
	// Create service materials for each shard/replica
	for shard := 0; shard < shards; shard++ {
		for replica := 0; replica < replicas; replica++ {
			config.Spec.TelemetryStore.Status.Extras["cfgPath"] = fmt.Sprintf(telemetrystoremolding.TelemetryStorePerInstanceFileFormat, metaDataName, v1alpha1.MoldingKindTelemetryStore.String(), storeKind, shard, replica, telemetrystoremolding.ConfigClickhousev2556YAML.String())
			serviceName := fmt.Sprintf("%s-telemetrystore-%s-%d-%d", metaDataName, storeKind, shard, replica)
			material, err := executeServiceTemplate(*tmpl, config, serviceName+".service")
			if err != nil {
				return nil, fmt.Errorf("failed to get service material for %s: %w", serviceName, err)
			}
			materials = append(materials, material)
		}
	}

	return materials, nil
}

func (casting *linuxCasting) forgeTelemetryKeeper(tmpl *types.Template, config *v1alpha1.Casting) ([]types.Material, error) {
	if !config.Spec.TelemetryKeeper.Spec.Enabled {
		return nil, nil
	}

	spec := &config.Spec.TelemetryKeeper.Spec
	keeperKind := config.Spec.TelemetryKeeper.Kind.String()
	replicas := getReplicaCount(*spec.Cluster.Replicas)

	// Create config materials
	materials, err := createConfigMaterials(spec.Config.Data)
	if err != nil {
		return nil, err
	}

	if config.Spec.TelemetryKeeper.Status.Extras == nil {
		config.Spec.TelemetryKeeper.Status.Extras = make(map[string]string)
	}
	metaDataName := config.Metadata.Name
	// Create service materials for each replica
	for replica := 0; replica < replicas; replica++ {
		config.Spec.TelemetryKeeper.Status.Extras["cfgPath"] = fmt.Sprintf("%s-telemetrykeeper-%s-%d.yaml", metaDataName, keeperKind, replica)
		serviceName := fmt.Sprintf("%s-telemetrykeeper-%s-%d", config.Metadata.Name, keeperKind, replica)
		material, err := executeServiceTemplate(*tmpl, config, serviceName+".service")
		if err != nil {
			return nil, fmt.Errorf("failed to get service material for %s: %w", serviceName, err)
		}
		materials = append(materials, material)
	}

	return materials, nil
}
