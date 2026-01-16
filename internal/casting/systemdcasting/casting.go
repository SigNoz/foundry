package systemdcasting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/types"
)

const (
	svcSuffix = ".service"
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

func (l *linuxCasting) Enricher(ctx context.Context, cfg *v1alpha1.Casting) (molding.MoldingEnricher, error) {
	return newLinuxMoldingEnricher(cfg), nil
}

func (l *linuxCasting) Forge(ctx context.Context, cfg v1alpha1.Casting) ([]types.Material, error) {
	var materials []types.Material
	for _, tmpl := range l.castings {
		m, err := l.forgeCasting(tmpl, &cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to forge: %w", err)
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

func (l *linuxCasting) forgeCasting(tmpl *types.Template, cfg *v1alpha1.Casting) ([]types.Material, error) {
	switch tmpl {
	case signozServiceTemplate:
		return l.forgeSignoz(tmpl, cfg)
	case metaStoreServiceTemplate:
		return l.forgeMetaStore(tmpl, cfg)
	case ingesterServiceTemplate:
		return l.forgeIngester(tmpl, cfg)
	case telemetryStoreServiceTemplate:
		return l.forgeTelemetryStore(tmpl, cfg)
	case telemetryKeeperServiceTemplate:
		return l.forgeTelemetryKeeper(tmpl, cfg)
	default:
		return nil, nil
	}
}

// --- Component Handlers ---

func (l *linuxCasting) forgeIngester(tmpl *types.Template, cfg *v1alpha1.Casting) ([]types.Material, error) {
	spec := &cfg.Spec.Ingester
	if !spec.Spec.Enabled {
		return nil, nil
	}

	name := cfg.Metadata.Name
	if spec.Status.Extras == nil {
		spec.Status.Extras = make(map[string]string)
	}
	if spec.Status.Config.Data == nil{
		return []types.Material{}, fmt.Errorf("No data has been molded for the Molding %s", v1alpha1.MoldingKindIngester)
	}
	mats, err := l.configMaterials(spec.Status.Config.Data)
	if err != nil {
		return nil, err
	}
	spec.Status.Extras["cfgPath"] = mats[0].Path()
	spec.Status.Extras["cfgOpampPath"] = mats[1].Path()

	m, err := l.materialzedTemplate(tmpl, cfg, name+"-ingester"+svcSuffix)
	mats = append(mats, m)
	return mats, err
}

func (l *linuxCasting) forgeSignoz(tmpl *types.Template, cfg *v1alpha1.Casting) ([]types.Material, error) {
	spec := &cfg.Spec.Signoz
	if !spec.Spec.Enabled {
		return nil, nil
	}
	return l.forgeEnvService(tmpl, cfg, &spec.Status, cfg.Metadata.Name+"-signoz")
}

func (l *linuxCasting) forgeMetaStore(tmpl *types.Template, cfg *v1alpha1.Casting) ([]types.Material, error) {
	spec := &cfg.Spec.MetaStore
	if !spec.Spec.Enabled {
		return nil, nil
	}
	prefix := fmt.Sprintf("%s-metastore-%s", cfg.Metadata.Name, spec.Kind.String())
	return l.forgeEnvService(tmpl, cfg, &spec.Status, prefix)
}

func (l *linuxCasting) forgeTelemetryStore(tmpl *types.Template, cfg *v1alpha1.Casting) ([]types.Material, error) {
	spec := &cfg.Spec.TelemetryStore
	if !spec.Spec.Enabled {
		return nil, nil
	}

	kind, name := spec.Kind.String(), cfg.Metadata.Name
	reps, shards := max(1, *spec.Spec.Cluster.Replicas+1), max(1, *spec.Spec.Cluster.Shards)

	if spec.Status.Config.Data == nil{
		return []types.Material{}, fmt.Errorf("No data has been molded for the Molding %s", v1alpha1.MoldingKindTelemetryStore)
	}
	mats, err := l.configMaterials(spec.Status.Config.Data)
	if err != nil {
		return nil, err
	}

	if spec.Status.Extras == nil {
		spec.Status.Extras = make(map[string]string)
	}
	for s := 0; s < shards; s++ {
		for r := 0; r < reps; r++ {
			spec.Status.Extras["cfgPath"] = mats[0].Path()
			svcName := fmt.Sprintf("%s-telemetrystore-%s-%d-%d%s", name, kind, s, r, svcSuffix)
			m, err := l.materialzedTemplate(tmpl, cfg, svcName)
			if err != nil {
				return nil, err
			}
			mats = append(mats, m)
		}
	}
	return mats, nil
}

func (l *linuxCasting) forgeTelemetryKeeper(tmpl *types.Template, cfg *v1alpha1.Casting) ([]types.Material, error) {
	spec := &cfg.Spec.TelemetryKeeper
	if !spec.Spec.Enabled {
		return nil, nil
	}

	kind, name := spec.Kind.String(), cfg.Metadata.Name
	reps := max(1, *spec.Spec.Cluster.Replicas)
	if spec.Status.Config.Data == nil{
		return []types.Material{}, fmt.Errorf("No data has been molded for the Molding %s", v1alpha1.MoldingKindTelemetryKeeper)
	}
	mats, err := l.configMaterials(spec.Status.Config.Data)
	if err != nil {
		return nil, err
	}

	if spec.Status.Extras == nil {
		spec.Status.Extras = make(map[string]string)
	}
	for r := 0; r < reps; r++ {
		spec.Status.Extras["cfgPath"] = mats[0].Path()
		svcName := fmt.Sprintf("%s-telemetrykeeper-%s-%d%s", name, kind, r, svcSuffix)
		m, err := l.materialzedTemplate(tmpl, cfg, svcName)
		if err != nil {
			return nil, err
		}
		mats = append(mats, m)
	}
	return mats, nil
}

func (l *linuxCasting) forgeEnvService(tmpl *types.Template, cfg *v1alpha1.Casting, status *v1alpha1.MoldingStatus, prefix string) ([]types.Material, error) {
	if status.Env == nil {
		return nil, fmt.Errorf("envs not enriched for %s", prefix)
	}

	envFile := prefix + ".env"
	mEnv, err := l.materializedEnv(status.Env, envFile)
	if err != nil {
		return nil, err
	}

	if status.Extras == nil {
		status.Extras = make(map[string]string)
	}
	status.Extras["envPath"] = envFile

	mSvc, err := l.materialzedTemplate(tmpl, cfg, prefix+svcSuffix)
	if err != nil {
		return nil, err
	}

	return []types.Material{mEnv, mSvc}, nil
}

func (l *linuxCasting) materialzedTemplate(tmpl *types.Template, cfg *v1alpha1.Casting, path string) (types.Material, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return types.Material{}, fmt.Errorf("execute template %s: %w", path, err)
	}
	return types.NewINIMaterial(buf.Bytes(), path)
}

func (l *linuxCasting) materializedEnv(envs map[string]string, path string) (types.Material, error) {
	jb, _ := json.Marshal(envs)
	ib, err := types.JSONToINI(jb)
	if err != nil {
		return types.Material{}, fmt.Errorf("failed to convert env to INI: %w", err)
	}
	return types.NewINIMaterial(ib, path)
}

func (l *linuxCasting) configMaterials(data map[string]string) ([]types.Material, error) {
	mats := make([]types.Material, 0, len(data))
	for file, content := range data {
		m, err := types.NewYAMLMaterial([]byte(content), file)
		if err != nil {
			return nil, fmt.Errorf("failed to create config material %s: %w", file, err)
		}
		mats = append(mats, m)
	}
	return mats, nil
}
