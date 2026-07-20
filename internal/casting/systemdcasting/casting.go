package systemdcasting

import (
	"context"
	"fmt"
	"log/slog"

	rootcasting "github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/binarytooler"

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
	logger *slog.Logger
}

func New(logger *slog.Logger) *systemdCasting {
	return &systemdCasting{logger: logger}
}

func (c *systemdCasting) Enricher(ctx context.Context, config *installation.Casting) (molding.MoldingEnricher, error) {
	return newSystemdMoldingEnricher(config), nil
}

func (c *systemdCasting) Forge(ctx context.Context, cfg installation.Casting, poursPath string) ([]domain.Material, error) {
	// Build order matches molding order: later components read status enriched by
	// earlier ones. Each builder no-ops when its component is disabled.
	builders := []func(*installation.Casting) ([]domain.Material, error){
		c.forgeTelemetryKeeper,
		c.forgeTelemetryStore,
		c.forgeMetaStore,
		c.forgeSignoz,
		c.forgeIngester,
		c.forgeMCP,
		c.forgeMigrator,
	}

	var materials []domain.Material
	for _, build := range builders {
		m, err := build(&cfg)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to forge")
		}
		materials = append(materials, m...)
	}
	return materials, nil
}

func (c *systemdCasting) Cast(ctx context.Context, config installation.Casting, poursPath string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	units, err := c.discoverUnits(poursPath)
	if err != nil {
		return err
	}
	if len(units) == 0 {
		c.logger.WarnContext(ctx, "no service units found in pours directory", slog.String("pours_path", poursPath))
		return nil
	}

	if err := c.provision(ctx, &config, poursPath); err != nil {
		return err
	}
	if err := c.initializeMetaStore(ctx, &config); err != nil {
		return err
	}
	if err := c.initializeTelemetryKeeper(ctx, &config); err != nil {
		return err
	}
	if err := c.startUnits(ctx, units); err != nil {
		return err
	}

	c.logger.InfoContext(ctx, "installed systemd services", slog.Int("count", len(units)))
	return nil
}

func (c *systemdCasting) forgeTelemetryKeeper(cfg *installation.Casting) ([]domain.Material, error) {
	if !cfg.Spec.TelemetryKeeper.Spec.IsEnabled() {
		return nil, nil
	}

	if cfg.Spec.TelemetryKeeper.Kind == installation.TelemetryKeeperKindZookeeper {
		return c.forgeZookeeper(cfg)
	}

	spec := &cfg.Spec.TelemetryKeeper
	if spec.Status.Extras == nil {
		spec.Status.Extras = make(map[string]string)
	}

	kind := spec.Kind.String()
	reps := max(1, *spec.Spec.Cluster.Replicas)

	// Config materials first: the cfgPath extra is derived from them.
	cfgMats, err := c.configMaterials(spec.Spec.Config.Data, "telemetrykeeper", kind)
	if err != nil {
		return nil, err
	}
	if len(cfgMats) > 0 {
		spec.Status.Extras["cfgPath"] = filepath.Join("/etc/clickhouse-keeper/", filepath.Base(cfgMats[0].Path()))
	}

	var materials []domain.Material
	for r := range reps {
		svcMat, err := c.renderTemplate(telemetryKeeperServiceTemplate, cfg, fmt.Sprintf("%s-telemetrykeeper-%s-%d%s", cfg.Metadata.Name, kind, r, svcSuffix))
		if err != nil {
			return nil, err
		}
		materials = append(materials, svcMat)
	}

	return append(materials, cfgMats...), nil
}

// forgeZookeeper renders the zookeeper unit and its zoo.cfg. The config file is
// part of the unit's materialization (the unit's ExecStart points at it), not
// molding config, so it is tuned via patches rather than spec.config.
func (c *systemdCasting) forgeZookeeper(cfg *installation.Casting) ([]domain.Material, error) {
	kind := cfg.Spec.TelemetryKeeper.Kind.String()

	svcMat, err := c.renderTemplate(zookeeperServiceTemplate, cfg, fmt.Sprintf("%s-telemetrykeeper-%s-0%s", cfg.Metadata.Name, kind, svcSuffix))
	if err != nil {
		return nil, err
	}

	cfgMat, err := c.renderTemplate(zookeeperConfigTemplate, cfg, filepath.Join("telemetrykeeper", kind, "zoo.cfg"))
	if err != nil {
		return nil, err
	}

	return []domain.Material{svcMat, cfgMat}, nil
}

func (c *systemdCasting) forgeTelemetryStore(cfg *installation.Casting) ([]domain.Material, error) {
	if !cfg.Spec.TelemetryStore.Spec.IsEnabled() {
		return nil, nil
	}

	spec := &cfg.Spec.TelemetryStore
	kind := spec.Kind.String()
	reps := max(1, *spec.Spec.Cluster.Replicas+1)
	shards := max(1, *spec.Spec.Cluster.Shards)

	var materials []domain.Material
	for s := range shards {
		for r := range reps {
			svcMat, err := c.renderTemplate(telemetryStoreServiceTemplate, cfg, fmt.Sprintf("%s-telemetrystore-%s-%d-%d%s", cfg.Metadata.Name, kind, s, r, svcSuffix))
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

	return append(materials, cfgMats...), nil
}

func (c *systemdCasting) forgeMetaStore(cfg *installation.Casting) ([]domain.Material, error) {
	if !cfg.Spec.MetaStore.Spec.IsEnabled() || cfg.Spec.MetaStore.Kind != installation.MetaStoreKindPostgres {
		return nil, nil
	}

	svcMat, err := c.renderTemplate(metaStoreServiceTemplate, cfg, fmt.Sprintf("%s-metastore-%s%s", cfg.Metadata.Name, cfg.Spec.MetaStore.Kind.String(), svcSuffix))
	if err != nil {
		return nil, err
	}
	return []domain.Material{svcMat}, nil
}

func (c *systemdCasting) forgeSignoz(cfg *installation.Casting) ([]domain.Material, error) {
	if !cfg.Spec.Signoz.Spec.IsEnabled() {
		return nil, nil
	}

	svcMat, err := c.renderTemplate(signozServiceTemplate, cfg, cfg.Metadata.Name+"-signoz"+svcSuffix)
	if err != nil {
		return nil, err
	}
	return []domain.Material{svcMat}, nil
}

func (c *systemdCasting) forgeIngester(cfg *installation.Casting) ([]domain.Material, error) {
	if !cfg.Spec.Ingester.Spec.IsEnabled() {
		return nil, nil
	}

	spec := &cfg.Spec.Ingester
	if spec.Status.Extras == nil {
		spec.Status.Extras = make(map[string]string)
	}
	spec.Status.Extras["cfgPath"] = filepath.Join("ingester", "ingester.yaml")
	spec.Status.Extras["cfgOpampPath"] = filepath.Join("ingester", "opamp.yaml")

	svcMat, err := c.renderTemplate(ingesterServiceTemplate, cfg, cfg.Metadata.Name+"-ingester"+svcSuffix)
	if err != nil {
		return nil, err
	}

	cfgMats, err := c.configMaterials(spec.Spec.Config.Data, "ingester", "")
	if err != nil {
		return nil, err
	}

	return append([]domain.Material{svcMat}, cfgMats...), nil
}

func (c *systemdCasting) forgeMCP(cfg *installation.Casting) ([]domain.Material, error) {
	if !cfg.Spec.MCP.Spec.IsEnabled() {
		return nil, nil
	}

	svcMat, err := c.renderTemplate(mcpServiceTemplate, cfg, cfg.Metadata.Name+"-mcp"+svcSuffix)
	if err != nil {
		return nil, err
	}
	return []domain.Material{svcMat}, nil
}

func (c *systemdCasting) forgeMigrator(cfg *installation.Casting) ([]domain.Material, error) {
	if !cfg.Spec.TelemetryStore.Spec.IsEnabled() {
		return nil, nil
	}

	svcMat, err := c.renderTemplate(telemetryStoreMigratorServiceTemplate, cfg, cfg.Metadata.Name+"-telemetrystore-migrator"+svcSuffix)
	if err != nil {
		return nil, err
	}
	return []domain.Material{svcMat}, nil
}

func (c *systemdCasting) configMaterials(data map[string]string, component string, kind string) ([]domain.Material, error) {
	mats := make([]domain.Material, 0, len(data))
	for filename, content := range data {
		m, err := domain.NewYAMLMaterial([]byte(content), filepath.Join(rootcasting.DeploymentDir, component, kind, filename))
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to create %s config material %s", component, filename)
		}
		mats = append(mats, m)
	}
	return mats, nil
}

func (c *systemdCasting) renderTemplate(tmpl *domain.Template, cfg *installation.Casting, path string) (domain.Material, error) {
	return tmpl.Render(cfg, filepath.Join(rootcasting.DeploymentDir, path))
}

// discoverUnits returns the absolute paths of the *.service units forged into
// the deployment directory.
func (c *systemdCasting) discoverUnits(poursPath string) ([]string, error) {
	deploymentPath := filepath.Join(poursPath, rootcasting.DeploymentDir)
	entries, err := os.ReadDir(deploymentPath)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to read directory %s", deploymentPath)
	}

	var units []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), svcSuffix) {
			continue
		}
		units = append(units, filepath.Join(deploymentPath, entry.Name()))
	}
	return units, nil
}

// provision creates the signoz user and runtime directories, copies component
// configs into their runtime locations, and validates the required binaries.
func (c *systemdCasting) provision(ctx context.Context, config *installation.Casting, poursPath string) error {
	if _, err := user.Lookup("signoz"); err != nil {
		c.logger.InfoContext(ctx, "creating signoz user")
		if err := c.execCommand(ctx, "useradd", "-d", poursPath, "signoz"); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to create signoz user")
		}
	}

	// Create runtime directories before chowning them, so they end up owned by
	// the signoz user instead of whoever ran cast (root). Only directories for
	// enabled components are created.
	dirs := []string{poursPath}
	if config.Spec.Signoz.Spec.IsEnabled() {
		dirs = append(dirs, "/opt/signoz")
	}
	if config.Spec.Ingester.Spec.IsEnabled() {
		dirs = append(dirs, "/opt/ingester", "/var/lib/ingester")
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to create directory %s", dir)
		}
		_ = c.execCommand(ctx, "chown", "-R", "signoz:signoz", dir) // best effort
	}

	// Copy clickhouse configs to standard locations
	if config.Spec.TelemetryStore.Spec.IsEnabled() {
		src := filepath.Join(poursPath, rootcasting.DeploymentDir, "telemetrystore", config.Spec.TelemetryStore.Kind.String())
		if err := c.copyDir(src, "/etc/clickhouse-server/"); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to copy clickhouse-server configs")
		}
	}
	if config.Spec.TelemetryKeeper.Spec.IsEnabled() {
		src := filepath.Join(poursPath, rootcasting.DeploymentDir, "telemetrykeeper", config.Spec.TelemetryKeeper.Kind.String())
		dst := "/etc/clickhouse-keeper/"
		if config.Spec.TelemetryKeeper.Kind == installation.TelemetryKeeperKindZookeeper {
			dst = "/etc/zookeeper/"
		}
		if err := c.copyDir(src, dst); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to copy telemetrykeeper configs")
		}
	}

	// Copy ingester configs out of the pours directory (which lives under root's
	// home, unreadable by the signoz user) into the ingester working directory.
	if config.Spec.Ingester.Spec.IsEnabled() {
		src := filepath.Join(poursPath, rootcasting.DeploymentDir, "ingester")
		if _, err := os.Stat(src); err == nil {
			if err := c.copyDir(src, "/opt/ingester/"); err != nil {
				return errors.Wrapf(err, errors.TypeInternal, "failed to copy ingester configs")
			}
			_ = c.execCommand(ctx, "chown", "-R", "signoz:signoz", "/opt/ingester/") // best effort
		}
	}

	return c.validateBinaries(ctx, config)
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

// validateBinaries checks that every component binary the casting will exec
// exists at its resolved path: the annotation override if set, otherwise the
// default. Each binary is verified through a binary tooler and all misses are
// aggregated into a single error.
func (c *systemdCasting) validateBinaries(ctx context.Context, config *installation.Casting) error {
	var missing []string
	var binaries []tooler.Tooler

	annotations := config.Metadata.Annotations

	if config.Spec.TelemetryKeeper.Spec.IsEnabled() && config.Spec.TelemetryKeeper.Kind == installation.TelemetryKeeperKindClickhouseKeeper {
		binaries = append(binaries, binarytooler.New("clickhouse-keeper", installation.TelemetryKeeperClickHouseKeeperBinaryPath.Resolve(annotations)))
	}
	if config.Spec.TelemetryKeeper.Spec.IsEnabled() && config.Spec.TelemetryKeeper.Kind == installation.TelemetryKeeperKindZookeeper {
		binaries = append(binaries, binarytooler.New("zookeeper", installation.TelemetryKeeperZookeeperBinaryPath.Resolve(annotations)))
	}
	if config.Spec.TelemetryStore.Spec.IsEnabled() {
		binaries = append(binaries, binarytooler.New("clickhouse", installation.TelemetryStoreClickHouseBinaryPath.Resolve(annotations)))
	}
	if config.Spec.MetaStore.Spec.IsEnabled() && config.Spec.MetaStore.Kind == installation.MetaStoreKindPostgres {
		binaries = append(binaries, binarytooler.New("postgres", installation.MetaStorePostgresBinaryPath.Resolve(annotations)))
	}
	if config.Spec.Signoz.Spec.IsEnabled() {
		binaries = append(binaries, binarytooler.New("signoz", installation.SignozBinaryPath.Resolve(annotations)))
	}
	if config.Spec.Ingester.Spec.IsEnabled() {
		binaries = append(binaries, binarytooler.New("ingester", installation.IngesterBinaryPath.Resolve(annotations)))
	}
	if config.Spec.MCP.Spec.IsEnabled() {
		binaries = append(binaries, binarytooler.New("mcp", installation.MCPBinaryPath.Resolve(annotations)))
	}

	for _, t := range binaries {
		if err := t.Gauge(ctx); err != nil {
			missing = append(missing, err.Error())
		}
	}

	if len(missing) > 0 {
		return errors.Newf(errors.TypeNotFound, "missing binaries: %s - please install before running cast", strings.Join(missing, "; "))
	}
	return nil
}

// initializeMetaStore prepares the metastore's on-disk state before its unit
// starts: bootstrapping postgres, or creating the sqlite data directory.
func (c *systemdCasting) initializeMetaStore(ctx context.Context, config *installation.Casting) error {
	if !config.Spec.MetaStore.Spec.IsEnabled() {
		return nil
	}

	switch config.Spec.MetaStore.Kind {
	case installation.MetaStoreKindPostgres:
		return c.initializePostgres(ctx, config)
	case installation.MetaStoreKindSQLite:
		if err := os.MkdirAll("/var/lib/signoz", 0755); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to create sqlite data directory")
		}
		_ = c.execCommand(ctx, "chown", "-R", "signoz:signoz", "/var/lib/signoz") // best effort
	}
	return nil
}

// initializeTelemetryKeeper creates the zookeeper service user; the data and
// log directories are created by systemd via StateDirectory/LogsDirectory.
func (c *systemdCasting) initializeTelemetryKeeper(ctx context.Context, config *installation.Casting) error {
	if !config.Spec.TelemetryKeeper.Spec.IsEnabled() || config.Spec.TelemetryKeeper.Kind != installation.TelemetryKeeperKindZookeeper {
		return nil
	}

	if _, err := user.Lookup("zookeeper"); err != nil {
		c.logger.InfoContext(ctx, "creating zookeeper user")
		if err := c.execCommand(ctx, "useradd", "-r", "-s", "/sbin/nologin", "zookeeper"); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to create zookeeper user")
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
		c.logger.DebugContext(ctx, "postgres already initialized", slog.String("path", pgDataDir))
		return nil
	}

	c.logger.InfoContext(ctx, "initializing postgres")

	// Clean up any leftover state from previous failed initialization
	c.cleanupPostgresInit(ctx, pgDataDir, pwfile)

	// Create directories
	if err := os.MkdirAll(pgDataDir, 0700); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to create PostgreSQL data directory")
	}

	// A binary/tarball postgres install does not create the `postgres` system
	// user, so the chown and `su - postgres` steps below would fail. Create it.
	if _, err := user.Lookup("postgres"); err != nil {
		c.logger.InfoContext(ctx, "creating postgres user")
		if err := c.execCommand(ctx, "useradd", "-r", "-s", "/sbin/nologin", "postgres"); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to create postgres user")
		}
	}

	if err := c.execCommand(ctx, "chown", "-R", "postgres:postgres", filepath.Dir(pgDataDir)); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to set ownership on PostgreSQL data directory")
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
		return errors.Wrapf(err, errors.TypeInternal, "failed to create password file")
	}
	_ = c.execCommand(ctx, "chown", "postgres:postgres", pwfile)

	// Resolve the postgres binary path from the annotation catalog to determine
	// its bin directory (which also holds initdb and pg_ctl).
	postgresBin := installation.MetaStorePostgresBinaryPath.Resolve(config.Metadata.Annotations)
	postgresBinDir := filepath.Dir(postgresBin)
	initdbPath := filepath.Join(postgresBinDir, "initdb")
	pgCtlPath := filepath.Join(postgresBinDir, "pg_ctl")

	// Initialize database
	c.logger.DebugContext(ctx, "running initdb", slog.String("user", pgUser), slog.String("initdb", initdbPath))
	if err := c.execCommand(ctx, "su", "-", "postgres", "-c",
		fmt.Sprintf("%s -D %s --username=%s --pwfile=%s", initdbPath, pgDataDir, pgUser, pwfile)); err != nil {
		c.cleanupPostgresInit(ctx, pgDataDir, pwfile)
		return errors.Wrapf(err, errors.TypeInternal, "failed to initialize PostgreSQL")
	}

	// Start temp server and create database
	c.logger.DebugContext(ctx, "starting temporary postgres for database creation")
	if err := c.execCommand(ctx, "su", "-", "postgres", "-c",
		fmt.Sprintf("%s -D %s -o \"-c listen_addresses=localhost\" -w start", pgCtlPath, pgDataDir)); err != nil {
		c.cleanupPostgresInit(ctx, pgDataDir, pwfile)
		return errors.Wrapf(err, errors.TypeInternal, "failed to start temporary postgres")
	}

	// Create database
	c.logger.DebugContext(ctx, "creating database", slog.String("database", dbName))
	cmd := exec.CommandContext(ctx, "psql", "-U", pgUser, "-h", "localhost", "-d", "postgres", "-c", fmt.Sprintf("CREATE DATABASE %s;", dbName))
	cmd.Env = append(os.Environ(), "PGPASSWORD="+pgPass)
	_ = cmd.Run() // ignore error - database may already exist

	// Stop temporary PostgreSQL
	if err := c.execCommand(ctx, "su", "-", "postgres", "-c", fmt.Sprintf("%s -D %s -m fast -w stop", pgCtlPath, pgDataDir)); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to stop temporary postgres")
	}

	// Clean up password file
	if err := os.Remove(pwfile); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to remove password file")
	}

	return nil
}

// cleanupPostgresInit removes leftover state from a failed PostgreSQL initialization.
func (c *systemdCasting) cleanupPostgresInit(ctx context.Context, pgDataDir, pwfile string) {
	// Remove password file if it exists
	if _, err := os.Stat(pwfile); err == nil {
		c.logger.DebugContext(ctx, "removing leftover password file", slog.String("path", pwfile))
		_ = os.Remove(pwfile)
	}

	// Remove data directory if it exists but is not properly initialized
	if _, err := os.Stat(pgDataDir); err == nil {
		if _, err := os.Stat(filepath.Join(pgDataDir, "PG_VERSION")); os.IsNotExist(err) {
			c.logger.DebugContext(ctx, "removing incomplete postgres data directory", slog.String("path", pgDataDir))
			_ = os.RemoveAll(pgDataDir)
		}
	}
}

// startUnits enables every unit (so dependency references resolve), reloads
// systemd to pick up the new unit files, then starts them. Ordering between
// units is handled by systemd via After=/Requires=.
func (c *systemdCasting) startUnits(ctx context.Context, units []string) error {
	for _, unit := range units {
		name := filepath.Base(unit)
		c.logger.DebugContext(ctx, "enabling unit", slog.String("unit", name))
		if err := c.systemctl(ctx, "enable", unit); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to enable unit %s", name)
		}
	}

	if err := c.systemctl(ctx, "daemon-reload"); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "systemd daemon-reload failed")
	}

	for _, unit := range units {
		name := filepath.Base(unit)
		c.logger.InfoContext(ctx, "starting unit", slog.String("unit", name))
		if err := c.systemctl(ctx, "start", "--no-block", name); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to start unit %s", name)
		}
	}

	return nil
}

// execCommand runs a command, streaming its output, and returns an error if it fails.
func (c *systemdCasting) execCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// systemctl runs a systemctl subcommand.
func (c *systemdCasting) systemctl(ctx context.Context, args ...string) error {
	return c.execCommand(ctx, "systemctl", args...)
}
