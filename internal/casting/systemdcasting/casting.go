package systemdcasting

import (
	"context"
	"fmt"
	"log/slog"

	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/molding"

	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
)

const svcSuffix = ".service"

var _ rootcasting.Casting = (*systemdCasting)(nil)

type systemdCasting struct {
	logger   *slog.Logger
	castings []*domain.Template
}

func New(logger *slog.Logger) *systemdCasting {
	return &systemdCasting{
		logger: logger,
		castings: []*domain.Template{
			telemetryKeeperServiceTemplate,
			telemetryStoreServiceTemplate,
			metaStoreServiceTemplate,
			signozServiceTemplate,
			ingesterServiceTemplate,
			telemetryStoreMigratorServiceTemplate,
		},
	}
}

func (c *systemdCasting) Enricher(ctx context.Context, config *installation.Casting) (molding.MoldingEnricher, error) {
	return newLinuxMoldingEnricher(config), nil
}

func (c *systemdCasting) Forge(ctx context.Context, cfg installation.Casting, poursPath string) ([]domain.Material, error) {
	var materials []domain.Material
	for _, tmpl := range c.castings {
		m, err := c.forgeCasting(tmpl, &cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to forge: %w", err)
		}
		materials = append(materials, m...)
	}
	return materials, nil
}

func (c *systemdCasting) Cast(ctx context.Context, config installation.Casting, poursPath string) error {
	c.logger.InfoContext(ctx, "Starting systemd service installation", slog.String("pours_path", poursPath))

	runctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Discover and prepare services
	services, err := c.discoverAndPrepareServices(runctx, poursPath)
	if err != nil {
		return err
	}
	if len(services) == 0 {
		c.logger.WarnContext(runctx, "No service files found in pours directory")
		return nil
	}

	// Setup system environment
	if err := c.setupSystemEnvironment(runctx, &config, poursPath); err != nil {
		return err
	}

	if config.Spec.MetaStore.Spec.IsEnabled() {
		switch config.Spec.MetaStore.Kind {
		case installation.MetaStoreKindPostgres:
			if err := c.initializePostgres(ctx, &config); err != nil {
				return err
			}
		case installation.MetaStoreKindSQLite:
			if err := os.MkdirAll("/var/lib/signoz", 0755); err != nil {
				return fmt.Errorf("failed to create sqlite data directory: %w", err)
			}
			_ = c.execCommand(ctx, "chown", "-R", "signoz:signoz", "/var/lib/signoz")
		}
	}

	// Start all services - systemd dependencies handle ordering
	if err := c.startAllServices(runctx, services); err != nil {
		return err
	}

	c.logger.InfoContext(runctx, "Successfully installed all systemd services")
	return nil
}

func (c *systemdCasting) forgeCasting(tmpl *domain.Template, cfg *installation.Casting) ([]domain.Material, error) {
	switch tmpl {
	case signozServiceTemplate:
		if !cfg.Spec.Signoz.Spec.IsEnabled() {
			return nil, nil
		}
		return c.forgeSignoz(tmpl, cfg)
	case metaStoreServiceTemplate:
		if !cfg.Spec.MetaStore.Spec.IsEnabled() {
			return nil, nil
		}
		return c.forgeMetaStore(tmpl, cfg)
	case ingesterServiceTemplate:
		if !cfg.Spec.Ingester.Spec.IsEnabled() {
			return nil, nil
		}
		return c.forgeIngester(tmpl, cfg)
	case telemetryStoreServiceTemplate:
		if !cfg.Spec.TelemetryStore.Spec.IsEnabled() {
			return nil, nil
		}
		return c.forgeTelemetryStore(tmpl, cfg)
	case telemetryKeeperServiceTemplate:
		if !cfg.Spec.TelemetryKeeper.Spec.IsEnabled() {
			return nil, nil
		}
		return c.forgeTelemetryKeeper(tmpl, cfg)
	case telemetryStoreMigratorServiceTemplate:
		if !cfg.Spec.TelemetryStore.Spec.IsEnabled() {
			return nil, nil
		}
		return c.forgeMigrator(tmpl, cfg)
	default:
		return nil, nil
	}
}

func (c *systemdCasting) forgeIngester(tmpl *domain.Template, cfg *installation.Casting) ([]domain.Material, error) {
	spec := &cfg.Spec.Ingester

	if spec.Status.Extras == nil {
		spec.Status.Extras = make(map[string]string)
	}
	spec.Status.Extras["cfgPath"] = filepath.Join("ingester", "ingester.yaml")
	spec.Status.Extras["cfgOpampPath"] = filepath.Join("ingester", "opamp.yaml")
	spec.Status.Extras["workingDir"] = "/opt/ingester"

	var materials []domain.Material

	svcMat, err := c.renderTemplate(tmpl, cfg, cfg.Metadata.Name+"-ingester"+svcSuffix)
	if err != nil {
		return nil, err
	}
	materials = append(materials, svcMat)

	cfgMats, err := c.configMaterials(spec.Spec.Config.Data, "ingester", "")
	if err != nil {
		return nil, err
	}
	materials = append(materials, cfgMats...)

	return materials, nil
}

func (c *systemdCasting) forgeSignoz(tmpl *domain.Template, cfg *installation.Casting) ([]domain.Material, error) {
	spec := &cfg.Spec.Signoz

	if spec.Status.Extras == nil {
		spec.Status.Extras = make(map[string]string)
	}
	spec.Status.Extras["workingDir"] = "/opt/signoz"

	var materials []domain.Material

	svcMat, err := c.renderTemplate(tmpl, cfg, cfg.Metadata.Name+"-signoz"+svcSuffix)
	if err != nil {
		return nil, err
	}
	materials = append(materials, svcMat)

	return materials, nil
}

func (c *systemdCasting) forgeMetaStore(tmpl *domain.Template, cfg *installation.Casting) ([]domain.Material, error) {
	spec := &cfg.Spec.MetaStore

	if spec.Status.Extras == nil {
		spec.Status.Extras = make(map[string]string)
	}

	var materials []domain.Material

	switch spec.Kind {
	case installation.MetaStoreKindPostgres:
		svcMat, err := c.renderTemplate(tmpl, cfg, fmt.Sprintf("%s-metastore-%s%s", cfg.Metadata.Name, spec.Kind.String(), svcSuffix))
		if err != nil {
			return nil, err
		}
		materials = append(materials, svcMat)
	}

	return materials, nil
}

func (c *systemdCasting) forgeTelemetryStore(tmpl *domain.Template, cfg *installation.Casting) ([]domain.Material, error) {
	spec := &cfg.Spec.TelemetryStore

	if spec.Status.Extras == nil {
		spec.Status.Extras = make(map[string]string)
	}

	kind := spec.Kind.String()
	reps := max(1, *spec.Spec.Cluster.Replicas+1)
	shards := max(1, *spec.Spec.Cluster.Shards)

	var materials []domain.Material

	for s := range shards {
		for r := range reps {
			svcMat, err := c.renderTemplate(tmpl, cfg, fmt.Sprintf("%s-telemetrystore-%s-%d-%d%s", cfg.Metadata.Name, kind, s, r, svcSuffix))
			if err != nil {
				return nil, err
			}
			materials = append(materials, svcMat)
		}
	}

	cfgMats, err := c.configMaterials(spec.Spec.Config.Data, "telemetrystore", kind)
	if err != nil {
		return nil, err
	}
	materials = append(materials, cfgMats...)

	return materials, nil
}

func (c *systemdCasting) forgeTelemetryKeeper(tmpl *domain.Template, cfg *installation.Casting) ([]domain.Material, error) {
	spec := &cfg.Spec.TelemetryKeeper

	if spec.Status.Extras == nil {
		spec.Status.Extras = make(map[string]string)
	}

	kind := spec.Kind.String()
	reps := max(1, *spec.Spec.Cluster.Replicas)

	// Config materials are created first because cfgPath extra is derived from them
	cfgMats, err := c.configMaterials(spec.Spec.Config.Data, "telemetrykeeper", kind)
	if err != nil {
		return nil, err
	}
	if len(cfgMats) > 0 {
		spec.Status.Extras["cfgPath"] = filepath.Join("/etc/clickhouse-keeper/", filepath.Base(cfgMats[0].Path()))
	}

	var materials []domain.Material

	for r := range reps {
		svcMat, err := c.renderTemplate(tmpl, cfg, fmt.Sprintf("%s-telemetrykeeper-%s-%d%s", cfg.Metadata.Name, kind, r, svcSuffix))
		if err != nil {
			return nil, err
		}
		materials = append(materials, svcMat)
	}

	materials = append(materials, cfgMats...)

	return materials, nil
}

func (c *systemdCasting) forgeMigrator(tmpl *domain.Template, cfg *installation.Casting) ([]domain.Material, error) {
	var materials []domain.Material

	svcMat, err := c.renderTemplate(tmpl, cfg, cfg.Metadata.Name+"-telemetrystore-migrator"+svcSuffix)
	if err != nil {
		return nil, err
	}
	materials = append(materials, svcMat)

	return materials, nil
}

func (c *systemdCasting) configMaterials(data map[string]string, component string, kind string) ([]domain.Material, error) {
	mats := make([]domain.Material, 0, len(data))
	for filename, content := range data {
		m, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, component, kind, filename))
		if err != nil {
			return nil, fmt.Errorf("failed to create %s config material %s: %w", component, filename, err)
		}
		mats = append(mats, m)
	}
	return mats, nil
}

func (c *systemdCasting) renderTemplate(tmpl *domain.Template, cfg *installation.Casting, path string) (domain.Material, error) {
	return tmpl.Render(cfg, filepath.Join(rootcasting.DeploymentDir, path))
}

// execCommand executes a command and returns an error if it fails.
func (c *systemdCasting) execCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// discoverAndPrepareServices discovers service files in the pours directory and reloads systemd.
func (c *systemdCasting) discoverAndPrepareServices(ctx context.Context, poursPath string) ([]string, error) {
	deploymentPath := filepath.Join(poursPath, rootcasting.DeploymentDir)
	entries, err := os.ReadDir(deploymentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", deploymentPath, err)
	}

	var services []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), svcSuffix) {
			continue
		}
		services = append(services, filepath.Join(deploymentPath, entry.Name()))
	}

	if len(services) == 0 {
		return nil, nil
	}

	c.logger.DebugContext(ctx, "Found services", slog.Int("count", len(services)))

	if err := c.execCommand(ctx, "systemctl", "daemon-reload"); err != nil {
		return nil, fmt.Errorf("systemd daemon-reload failed: %w", err)
	}

	return services, nil
}

// setupSystemEnvironment creates signoz user, directories, copies configs, and validates binaries.
func (c *systemdCasting) setupSystemEnvironment(ctx context.Context, config *installation.Casting, poursPath string) error {
	// Create signoz user if needed
	if _, err := user.Lookup("signoz"); err != nil {
		c.logger.InfoContext(ctx, "Creating user: signoz")
		if err := c.execCommand(ctx, "useradd", "-d", poursPath, "signoz"); err != nil {
			return fmt.Errorf("failed to create signoz user: %w", err)
		}
	}

	// Setup working directory
	if err := os.MkdirAll(poursPath, 0755); err != nil {
		return fmt.Errorf("failed to create working directory %s: %w", poursPath, err)
	}
	_ = c.execCommand(ctx, "chown", "-R", "signoz:signoz", poursPath)      // best effort
	_ = c.execCommand(ctx, "chown", "-R", "signoz:signoz", "/opt/signoz/") // best effort

	// Copy clickhouse configs to standard locations
	if config.Spec.TelemetryStore.Spec.IsEnabled() {
		src := filepath.Join(poursPath, rootcasting.DeploymentDir, "telemetrystore", config.Spec.TelemetryStore.Kind.String())
		if err := c.copyDir(src, "/etc/clickhouse-server/"); err != nil {
			return fmt.Errorf("failed to copy clickhouse-server configs: %w", err)
		}
	}
	if config.Spec.TelemetryKeeper.Spec.IsEnabled() {
		src := filepath.Join(poursPath, rootcasting.DeploymentDir, "telemetrykeeper", config.Spec.TelemetryKeeper.Kind.String())
		if err := c.copyDir(src, "/etc/clickhouse-keeper/"); err != nil {
			return fmt.Errorf("failed to copy clickhouse-keeper configs: %w", err)
		}
	}

	// Validate required binaries
	return c.validateBinaries(config)
}

// copyDir copies all files from srcDir to dstDir.
func (c *systemdCasting) copyDir(srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(srcDir, entry.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dstDir, entry.Name()), data, 0644); err != nil {
			return err
		}
	}
	return nil
}

// validateBinaries checks if binaries exist at annotation paths.
// Only validates if annotations are set; defaults are handled in templates.
func (c *systemdCasting) validateBinaries(config *installation.Casting) error {
	annotations := config.Metadata.Annotations
	if annotations == nil {
		return fmt.Errorf("no binary paths found in annotations")
	}

	var missing []string

	// Check signoz binary if annotation is set
	if signozPath := annotations["foundry.signoz.io/signoz-binary-path"]; signozPath != "" {
		if _, err := os.Stat(signozPath); os.IsNotExist(err) {
			missing = append(missing, fmt.Sprintf("signoz binary (at %s)", signozPath))
		}
	}

	// Check ingester binary if annotation is set
	if ingesterPath := annotations["foundry.signoz.io/ingester-binary-path"]; ingesterPath != "" {
		if _, err := os.Stat(ingesterPath); os.IsNotExist(err) {
			missing = append(missing, fmt.Sprintf("ingester binary (at %s)", ingesterPath))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing binaries: %s - please install before running cast", strings.Join(missing, ", "))
	}
	return nil
}

// startAllServices enables and starts all discovered services.
// All services are enabled first so that dependency references resolve,
// then started — systemd handles ordering via After=/Requires=.
func (c *systemdCasting) startAllServices(ctx context.Context, services []string) error {
	// Enable all services first so dependencies can be resolved
	for _, svc := range services {
		unitName := filepath.Base(svc)
		c.logger.DebugContext(ctx, "Enabling service", slog.String("service", unitName))
		if err := c.execCommand(ctx, "systemctl", "enable", svc); err != nil {
			return fmt.Errorf("failed to enable service %s: %w", unitName, err)
		}
	}

	// Reload systemd to pick up all enabled services
	if err := c.execCommand(ctx, "systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("systemd daemon-reload failed: %w", err)
	}

	// Start all services without blocking — systemd dependencies handle ordering
	for _, svc := range services {
		unitName := filepath.Base(svc)
		c.logger.InfoContext(ctx, "Starting service", slog.String("service", unitName))
		if err := c.execCommand(ctx, "systemctl", "start", "--no-block", unitName); err != nil {
			return fmt.Errorf("failed to start service %s: %w", unitName, err)
		}
	}

	return nil
}

// initializePostgres sets up the PostgreSQL data directory.
func (c *systemdCasting) initializePostgres(ctx context.Context, config *installation.Casting) error {
	pgDataDir := "/usr/local/pgsql/data"
	pwfile := "/tmp/postgres_pwfile_init"

	// Check if PostgreSQL is already initialized by looking for PG_VERSION file
	if _, err := os.Stat(filepath.Join(pgDataDir, "PG_VERSION")); err == nil {
		c.logger.DebugContext(ctx, "PostgreSQL already initialized", slog.String("path", pgDataDir))
		return nil
	}

	c.logger.InfoContext(ctx, "Initializing PostgreSQL")

	// Clean up any leftover state from previous failed initialization
	c.cleanupPostgresInit(ctx, pgDataDir, pwfile)

	// Create directories
	if err := os.MkdirAll(pgDataDir, 0700); err != nil {
		return fmt.Errorf("failed to create PostgreSQL data directory: %w", err)
	}

	if err := c.execCommand(ctx, "chown", "-R", "postgres:postgres", filepath.Dir(pgDataDir)); err != nil {
		return fmt.Errorf("failed to set ownership on PostgreSQL data directory: %w", err)
	}

	// Get credentials
	env := config.Spec.MetaStore.Status.Env
	pgUser := env["POSTGRES_USER"]
	if pgUser == "" {
		pgUser = "postgres"
	}
	pgPass := env["POSTGRES_PASSWORD"]
	if pgPass == "" {
		pgPass = "postgres"
	}
	dbName := env["POSTGRES_DB"]
	if dbName == "" {
		dbName = pgUser
	}

	// Create password file
	if err := os.WriteFile(pwfile, []byte(pgPass+"\n"), 0600); err != nil {
		return fmt.Errorf("failed to create password file: %w", err)
	}
	_ = c.execCommand(ctx, "chown", "postgres:postgres", pwfile)

	// Get postgres binary path from config to determine bin directory
	postgresBin := config.Metadata.Annotations["foundry.signoz.io/metastore-postgres-binary-path"]

	if postgresBin == "" {
		return fmt.Errorf("metastore postgres binary is missing in annotations")
	}

	postgresBinDir := filepath.Dir(postgresBin)
	initdbPath := filepath.Join(postgresBinDir, "initdb")
	pgCtlPath := filepath.Join(postgresBinDir, "pg_ctl")

	// Initialize database
	c.logger.DebugContext(ctx, "Running initdb", slog.String("user", pgUser), slog.String("initdb", initdbPath))
	if err := c.execCommand(ctx, "su", "-", "postgres", "-c",
		fmt.Sprintf("%s -D %s --username=%s --pwfile=%s", initdbPath, pgDataDir, pgUser, pwfile)); err != nil {
		c.cleanupPostgresInit(ctx, pgDataDir, pwfile)
		return fmt.Errorf("failed to initialize PostgreSQL: %w", err)
	}

	// Start temp server and create database
	c.logger.DebugContext(ctx, "Starting temporary PostgreSQL for DB creation")
	if err := c.execCommand(ctx, "su", "-", "postgres", "-c",
		fmt.Sprintf("%s -D %s -o \"-c listen_addresses=localhost\" -w start", pgCtlPath, pgDataDir)); err != nil {
		c.cleanupPostgresInit(ctx, pgDataDir, pwfile)
		return fmt.Errorf("failed to start temporary postgres: %w", err)
	}

	// Create database
	c.logger.DebugContext(ctx, "Creating database", slog.String("database", dbName))
	cmd := exec.CommandContext(ctx, "psql", "-U", pgUser, "-h", "localhost", "-d", "postgres", "-c", fmt.Sprintf("CREATE DATABASE %s;", dbName))
	cmd.Env = append(os.Environ(), "PGPASSWORD="+pgPass)
	_ = cmd.Run() // ignore error - database may already exist

	// Stop temporary PostgreSQL
	if err := c.execCommand(ctx, "su", "-", "postgres", "-c", fmt.Sprintf("%s -D %s -m fast -w stop", pgCtlPath, pgDataDir)); err != nil {
		return fmt.Errorf("failed to stop temporary postgres: %w", err)
	}

	// Clean up password file
	if err := os.Remove(pwfile); err != nil {
		return fmt.Errorf("failed to remove password file: %w", err)
	}

	return nil
}

// cleanupPostgresInit removes leftover state from a failed PostgreSQL initialization.
func (c *systemdCasting) cleanupPostgresInit(ctx context.Context, pgDataDir, pwfile string) {
	// Remove password file if it exists
	if _, err := os.Stat(pwfile); err == nil {
		c.logger.DebugContext(ctx, "Removing leftover password file", slog.String("path", pwfile))
		_ = os.Remove(pwfile)
	}

	// Remove data directory if it exists but is not properly initialized
	if _, err := os.Stat(pgDataDir); err == nil {
		if _, err := os.Stat(filepath.Join(pgDataDir, "PG_VERSION")); os.IsNotExist(err) {
			c.logger.DebugContext(ctx, "Removing incomplete PostgreSQL data directory", slog.String("path", pgDataDir))
			_ = os.RemoveAll(pgDataDir)
		}
	}
}
