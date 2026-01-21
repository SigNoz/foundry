package systemdcasting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

const svcSuffix = ".service"

var _ casting.Casting = (*systemdCasting)(nil)

type systemdCasting struct {
	logger   *slog.Logger
	castings []*types.Template
}

func New(logger *slog.Logger) *systemdCasting {
	return &systemdCasting{
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

func (c *systemdCasting) Enricher(ctx context.Context, config *v1alpha1.Casting) (molding.MoldingEnricher, error) {
	return newLinuxMoldingEnricher(config), nil
}

func (c *systemdCasting) Forge(ctx context.Context, cfg v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	var materials []types.Material
	for _, tmpl := range c.castings {
		m, err := c.forgeCasting(tmpl, &cfg, poursPath)
		if err != nil {
			return nil, fmt.Errorf("failed to forge: %w", err)
		}
		materials = append(materials, m...)
	}
	return materials, nil
}

func (c *systemdCasting) Cast(ctx context.Context, config v1alpha1.Casting, poursPath string) error {
	c.logger.InfoContext(ctx, "Starting systemd service installation", slog.String("pours_path", poursPath))

	runctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Join commands with && to run in sequence
	//command := strings.Join(cast.Execute, " && ")
	command := ""

	c.logger.DebugContext(runctx, "Running command", slog.String("command", command))

	cmd := exec.CommandContext(runctx, "sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		c.logger.ErrorContext(runctx, "Command execution failed", slog.String("error", err.Error()))
		return err
	}

	c.logger.InfoContext(runctx, "Command executed successfully")
	return nil
}

func (c *systemdCasting) forgeCasting(tmpl *types.Template, cfg *v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	switch tmpl {
	case signozServiceTemplate:
		return c.forgeSignoz(tmpl, cfg, poursPath)
	case metaStoreServiceTemplate:
		return c.forgeMetaStore(tmpl, cfg, poursPath)
	case ingesterServiceTemplate:
		return c.forgeIngester(tmpl, cfg, poursPath)
	case telemetryStoreServiceTemplate:
		return c.forgeTelemetryStore(tmpl, cfg, poursPath)
	case telemetryKeeperServiceTemplate:
		return c.forgeTelemetryKeeper(tmpl, cfg, poursPath)
	default:
		return nil, nil
	}
}

// --- Forge Handlers ---

func (c *systemdCasting) forgeIngester(tmpl *types.Template, cfg *v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	spec := &cfg.Spec.Ingester
	if !spec.Spec.Enabled {
		return nil, nil
	}
	if spec.Status.Config.Data == nil {
		return nil, fmt.Errorf("no config molded for %s", v1alpha1.MoldingKindIngester)
	}

	// Initialize status extras
	if spec.Status.Extras == nil {
		spec.Status.Extras = make(map[string]string)
	}

	// Create config materials
	mats, err := c.configMaterials(spec.Status.Config.Data, "ingestor")
	if err != nil {
		return nil, err
	}

	// Set extras for template
	spec.Status.Extras["cfgPath"] = filepath.Join(poursPath, mats[0].Path())
	spec.Status.Extras["cfgOpampPath"] = filepath.Join(poursPath, mats[1].Path())
	spec.Status.Extras["workingDir"] = "/opt/ingester"

	// Create service material
	svcMat, err := c.renderTemplate(tmpl, cfg, cfg.Metadata.Name+"-ingester"+svcSuffix)
	if err != nil {
		return nil, err
	}
	return append(mats, svcMat), nil
}

func (c *systemdCasting) forgeSignoz(tmpl *types.Template, cfg *v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	spec := &cfg.Spec.Signoz
	if !spec.Spec.Enabled {
		return nil, nil
	}

	// Initialize status maps
	if spec.Status.Extras == nil {
		spec.Status.Extras = make(map[string]string)
	}
	if spec.Status.Env == nil {
		spec.Status.Env = make(map[string]string)
	}

	// Create env material
	prefix := cfg.Metadata.Name + "-signoz"
	envMat, err := c.envMaterial(spec.Status.Env, prefix)
	if err != nil {
		return nil, err
	}

	// Set extras for template
	spec.Status.Extras["envPath"] = filepath.Join(poursPath, envMat.Path())
	spec.Status.Extras["workingDir"] = "/opt/signoz"

	// Create service material
	svcMat, err := c.renderTemplate(tmpl, cfg, prefix+svcSuffix)
	if err != nil {
		return nil, err
	}
	return []types.Material{envMat, svcMat}, nil
}

func (c *systemdCasting) forgeMetaStore(tmpl *types.Template, cfg *v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	spec := &cfg.Spec.MetaStore
	if !spec.Spec.Enabled {
		return nil, nil
	}

	// Initialize status extras
	if spec.Status.Extras == nil {
		spec.Status.Extras = make(map[string]string)
	}

	// Create env material
	prefix := fmt.Sprintf("%s-metastore-%s", cfg.Metadata.Name, spec.Kind.String())
	envMat, err := c.envMaterial(spec.Status.Env, prefix)
	if err != nil {
		return nil, err
	}

	// Set extras for template
	spec.Status.Extras["envPath"] = filepath.Join(poursPath, envMat.Path())

	// Create service material
	svcMat, err := c.renderTemplate(tmpl, cfg, prefix+svcSuffix)
	if err != nil {
		return nil, err
	}
	return []types.Material{envMat, svcMat}, nil
}

func (c *systemdCasting) forgeTelemetryStore(tmpl *types.Template, cfg *v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	spec := &cfg.Spec.TelemetryStore
	if !spec.Spec.Enabled {
		return nil, nil
	}
	if spec.Status.Config.Data == nil {
		return nil, fmt.Errorf("no config molded for %s", v1alpha1.MoldingKindTelemetryStore)
	}

	// Initialize status extras
	if spec.Status.Extras == nil {
		spec.Status.Extras = make(map[string]string)
	}

	kind := spec.Kind.String()
	reps := max(1, *spec.Spec.Cluster.Replicas+1)
	shards := max(1, *spec.Spec.Cluster.Shards)

	// Create config materials
	mats, err := c.configMaterials(spec.Status.Config.Data, kind)
	if err != nil {
		return nil, err
	}

	// Set config path for template
	spec.Status.Extras["cfgPath"] = filepath.Join("/etc/clickhouse-server/", filepath.Base(mats[0].Path()))

	// Create service materials for each shard/replica
	for s := 0; s < shards; s++ {
		for r := 0; r < reps; r++ {
			svcName := fmt.Sprintf("%s-telemetrystore-%s-%d-%d%s", cfg.Metadata.Name, kind, s, r, svcSuffix)
			svcMat, err := c.renderTemplate(tmpl, cfg, svcName)
			if err != nil {
				return nil, err
			}
			mats = append(mats, svcMat)
		}
	}
	return mats, nil
}

func (c *systemdCasting) forgeTelemetryKeeper(tmpl *types.Template, cfg *v1alpha1.Casting, poursPath string) ([]types.Material, error) {
	spec := &cfg.Spec.TelemetryKeeper
	if !spec.Spec.Enabled {
		return nil, nil
	}
	if spec.Status.Config.Data == nil {
		return nil, fmt.Errorf("no config molded for %s", v1alpha1.MoldingKindTelemetryKeeper)
	}

	// Initialize status extras
	if spec.Status.Extras == nil {
		spec.Status.Extras = make(map[string]string)
	}

	kind := spec.Kind.String()
	reps := max(1, *spec.Spec.Cluster.Replicas)

	// Create config materials
	mats, err := c.configMaterials(spec.Status.Config.Data, kind)
	if err != nil {
		return nil, err
	}

	// Set config path for template
	spec.Status.Extras["cfgPath"] = filepath.Join("/etc/clickhouse-keeper/", filepath.Base(mats[0].Path()))

	// Create service materials for each replica
	for r := 0; r < reps; r++ {
		svcName := fmt.Sprintf("%s-telemetrykeeper-%s-%d%s", cfg.Metadata.Name, kind, r, svcSuffix)
		svcMat, err := c.renderTemplate(tmpl, cfg, svcName)
		if err != nil {
			return nil, err
		}
		mats = append(mats, svcMat)
	}
	return mats, nil
}

// --- Material Helpers ---

func (c *systemdCasting) renderTemplate(tmpl *types.Template, cfg *v1alpha1.Casting, path string) (types.Material, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return types.Material{}, fmt.Errorf("execute template %s: %w", path, err)
	}
	return types.NewINIMaterial(buf.Bytes(), path)
}

func (c *systemdCasting) envMaterial(envs map[string]string, prefix string) (types.Material, error) {
	if envs == nil {
		return types.Material{}, fmt.Errorf("envs not enriched for %s", prefix)
	}
	jb, _ := json.Marshal(envs)
	ib, err := types.JSONToINI(jb)
	if err != nil {
		return types.Material{}, fmt.Errorf("failed to convert env to INI: %w", err)
	}
	return types.NewINIMaterial(ib, fmt.Sprintf("%s/%s.env", prefix, prefix))
}

func (c *systemdCasting) configMaterials(data map[string]string, path string) ([]types.Material, error) {
	mats := make([]types.Material, 0, len(data))
	for file, content := range data {
		m, err := types.NewYAMLMaterial([]byte(content), filepath.Join(path, file))
		if err != nil {
			return nil, fmt.Errorf("failed to create config material %s: %w", file, err)
		}
		mats = append(mats, m)
	}
	return mats, nil
}
