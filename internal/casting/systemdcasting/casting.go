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
	"strings"
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

func (casting *linuxCasting) Cast(ctx context.Context, config v1alpha1.Casting, outputPath string) error {
	casting.logger.InfoContext(ctx, "Installing systemd services", slog.String("outputPath", outputPath))

	// Create a context with 5-minute timeout
	runctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Read all files from output directory
	files, err := os.ReadDir(outputPath)
	if err != nil {
		return fmt.Errorf("failed to read output directory %s: %w", outputPath, err)
	}

	var serviceFiles []string
	var configFiles []string
	var envFiles []string

	// Categorize files by type
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if filepath.Ext(name) == ".service" {
			serviceFiles = append(serviceFiles, name)
		} else if filepath.Ext(name) == ".yaml" || filepath.Ext(name) == ".yml" {
			configFiles = append(configFiles, name)
		} else if filepath.Ext(name) == ".env" {
			envFiles = append(envFiles, name)
		}
	}

	// Copy service files to /etc/systemd/system/
	systemdDir := "/etc/systemd/system"
	if err := os.MkdirAll(systemdDir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd directory: %w", err)
	}

	for _, serviceFile := range serviceFiles {
		srcPath := filepath.Join(outputPath, serviceFile)
		dstPath := filepath.Join(systemdDir, serviceFile)
		if err := casting.copyFile(runctx, srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to copy service file %s: %w", serviceFile, err)
		}
		casting.logger.InfoContext(runctx, "Copied service file", slog.String("file", serviceFile), slog.String("destination", dstPath))
	}

	// Copy config files to their target directories
	configFileMap := casting.buildConfigFileMap(&config)

	for _, configFile := range configFiles {
		srcPath := filepath.Join(outputPath, configFile)
		var dstPath string

		// Check if we have a mapping for this file from the config
		if targetPath, ok := configFileMap[configFile]; ok {
			dstPath = targetPath
		} else {
			// Fallback: Determine target directory based on config file name patterns
			baseName := filepath.Base(configFile)
			if strings.Contains(baseName, "clickhouse") && !strings.Contains(baseName, "keeper") {
				// ClickHouse config files
				targetDir := "/etc/clickhouse-server"
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					return fmt.Errorf("failed to create clickhouse-server directory: %w", err)
				}
				dstPath = filepath.Join(targetDir, baseName)
			} else if strings.Contains(baseName, "keeper") {
				// ClickHouse Keeper config files
				targetDir := "/etc/clickhouse-keeper"
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					return fmt.Errorf("failed to create clickhouse-keeper directory: %w", err)
				}
				dstPath = filepath.Join(targetDir, baseName)
			} else if strings.Contains(baseName, "ingester") || strings.Contains(baseName, "opamp") || strings.Contains(baseName, "v0129x") {
				// Ingester config files
				targetDir := "/opt/ingester"
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					return fmt.Errorf("failed to create ingester directory: %w", err)
				}
				dstPath = filepath.Join(targetDir, baseName)
			} else {
				// Unknown config file, skip it
				casting.logger.WarnContext(runctx, "Unknown config file type, skipping", slog.String("file", configFile))
				continue
			}
		}

		// Ensure target directory exists
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return fmt.Errorf("failed to create target directory for %s: %w", configFile, err)
		}

		if err := casting.copyFile(runctx, srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to copy config file %s: %w", configFile, err)
		}
		casting.logger.InfoContext(runctx, "Copied config file", slog.String("file", configFile), slog.String("destination", dstPath))
	}

	// Copy env files to /opt/signoz/conf/
	envDir := "/opt/signoz/conf"
	if len(envFiles) > 0 {
		if err := os.MkdirAll(envDir, 0755); err != nil {
			return fmt.Errorf("failed to create signoz conf directory: %w", err)
		}
		for _, envFile := range envFiles {
			srcPath := filepath.Join(outputPath, envFile)
			dstPath := filepath.Join(envDir, envFile)
			if err := casting.copyFile(runctx, srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to copy env file %s: %w", envFile, err)
			}
			casting.logger.InfoContext(runctx, "Copied env file", slog.String("file", envFile), slog.String("destination", dstPath))
		}
	}

	// Reload systemd daemon
	casting.logger.InfoContext(runctx, "Reloading systemd daemon")
	if err := casting.execCommand(runctx, "systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Enable and start services
	for _, serviceFile := range serviceFiles {
		serviceName := filepath.Base(serviceFile)
		casting.logger.InfoContext(runctx, "Enabling service", slog.String("service", serviceName))
		if err := casting.execCommand(runctx, "systemctl", "enable", serviceName); err != nil {
			casting.logger.WarnContext(runctx, "Failed to enable service", slog.String("service", serviceName), slog.String("error", err.Error()))
			// Continue even if enable fails
		}

		casting.logger.InfoContext(runctx, "Starting service", slog.String("service", serviceName))
		if err := casting.execCommand(runctx, "systemctl", "start", serviceName); err != nil {
			casting.logger.WarnContext(runctx, "Failed to start service", slog.String("service", serviceName), slog.String("error", err.Error()))
			// Continue even if start fails
		}
	}

	casting.logger.InfoContext(runctx, "Successfully installed systemd services")
	return nil
}

// copyFile copies a file from src to dst.
func (casting *linuxCasting) copyFile(ctx context.Context, src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// execCommand executes a command and returns an error if it fails.
func (casting *linuxCasting) execCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// buildConfigFileMap builds a map of config file names to their target paths
// based on the config's Extras metadata.
func (casting *linuxCasting) buildConfigFileMap(config *v1alpha1.Casting) map[string]string {
	fileMap := make(map[string]string)

	// Check TelemetryStore configs
	if config.Spec.TelemetryStore.Spec.Enabled && config.Spec.TelemetryStore.Status.Extras != nil {
		if cfgPath, ok := config.Spec.TelemetryStore.Status.Extras["cfgPath"]; ok {
			// Extract filename from path (e.g., "config.clickhouse.v2556.yaml")
			fileName := filepath.Base(cfgPath)
			// Target: /etc/clickhouse-server/{filename}
			fileMap[fileName] = filepath.Join("/etc/clickhouse-server", fileName)
		}
	}

	// Check TelemetryKeeper configs
	if config.Spec.TelemetryKeeper.Spec.Enabled && config.Spec.TelemetryKeeper.Status.Extras != nil {
		if cfgPath, ok := config.Spec.TelemetryKeeper.Status.Extras["cfgPath"]; ok {
			fileName := filepath.Base(cfgPath)
			// Target: /etc/clickhouse-keeper/{filename}
			fileMap[fileName] = filepath.Join("/etc/clickhouse-keeper", fileName)
		}
	}

	// Check Ingester configs
	if config.Spec.Ingester.Spec.Enabled && config.Spec.Ingester.Status.Extras != nil {
		if cfgPath, ok := config.Spec.Ingester.Status.Extras["cfgPath"]; ok {
			fileName := filepath.Base(cfgPath)
			fileMap[fileName] = filepath.Join("/opt/ingester", fileName)
		}
		if cfgOpampPath, ok := config.Spec.Ingester.Status.Extras["cfgOpampPath"]; ok {
			fileName := filepath.Base(cfgOpampPath)
			fileMap[fileName] = filepath.Join("/opt/ingester", fileName)
		}
	}

	return fileMap
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
	if spec.Status.Config.Data == nil {
		return []types.Material{}, fmt.Errorf("no config has been molded for the molding %s", v1alpha1.MoldingKindIngester)
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

	if spec.Status.Config.Data == nil {
		return []types.Material{}, fmt.Errorf("no config has been molded for the molding %s", v1alpha1.MoldingKindTelemetryStore)
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
	if spec.Status.Config.Data == nil {
		return []types.Material{}, fmt.Errorf("no config has been molded for the molding %s", v1alpha1.MoldingKindTelemetryKeeper)
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
