package systemdcasting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/molding"

	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/types"
)

const svcSuffix = ".service"

const (
	serviceStartTimeout  = 2 * time.Minute
	serviceReadyWait     = 10 * time.Second
	clickhouseReadyWait  = 60 * time.Second
	clickhouseRetryDelay = 2 * time.Second
)

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

	// Discover and prepare services
	serviceMap, err := c.discoverAndPrepareServices(runctx, poursPath)
	if err != nil {
		return err
	}
	if serviceMap == nil {
		c.logger.WarnContext(runctx, "No service files found in pours directory")
		return nil
	}

	// Setup system environment
	if err := c.setupSystemEnvironment(runctx, &config, serviceMap, poursPath); err != nil {
		return err
	}

	// Initialize and start foundation services
	if err := c.initAndStartFoundation(runctx, &config, serviceMap); err != nil {
		return err
	}

	// Run migrations and start application services
	if err := c.runMigrationsAndStartApps(runctx, &config, serviceMap, poursPath); err != nil {
		return err
	}

	c.logger.InfoContext(runctx, "Successfully installed all systemd services")
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
	for s := range shards {
		for r := range reps {
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
	for r := range reps {
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

// execCommand executes a command and returns an error if it fails.
func (c *systemdCasting) execCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// discoverAndPrepareServices discovers service files, categorizes them, and prepares systemd.
// Returns nil serviceMap if no services found.
func (c *systemdCasting) discoverAndPrepareServices(ctx context.Context, poursPath string) (map[string][]string, error) {
	entries, err := os.ReadDir(poursPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", poursPath, err)
	}

	serviceMap := map[string][]string{"keeper": {}, "store": {}, "postgres": {}, "signoz": {}, "ingester": {}}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".service") {
			continue
		}
		servicePath := filepath.Join(poursPath, entry.Name())
		baseName := strings.TrimSuffix(entry.Name(), ".service")

		switch {
		case strings.Contains(baseName, "-telemetrykeeper-"):
			serviceMap["keeper"] = append(serviceMap["keeper"], servicePath)
		case strings.Contains(baseName, "-telemetrystore-"):
			serviceMap["store"] = append(serviceMap["store"], servicePath)
		case strings.Contains(baseName, "-metastore-"):
			serviceMap["postgres"] = append(serviceMap["postgres"], servicePath)
		case strings.HasSuffix(baseName, "-signoz"):
			serviceMap["signoz"] = append(serviceMap["signoz"], servicePath)
		case strings.HasSuffix(baseName, "-ingester"):
			serviceMap["ingester"] = append(serviceMap["ingester"], servicePath)
		default:
			c.logger.WarnContext(ctx, "Unknown service type, skipping", slog.String("service", servicePath))
		}
	}

	// Check if any services were found
	total := 0
	for cat, svcs := range serviceMap {
		if len(svcs) > 0 {
			c.logger.DebugContext(ctx, "Found services", slog.String("category", cat), slog.Int("count", len(svcs)))
			total += len(svcs)
		}
	}
	if total == 0 {
		return map[string][]string{}, nil
	}

	// Reload systemd to pick up new service files
	c.logger.DebugContext(ctx, "Reloading systemd daemon")
	if err := c.execCommand(ctx, "systemctl", "daemon-reload"); err != nil {
		return nil, fmt.Errorf("systemd daemon-reload failed: %w", err)
	}

	return serviceMap, nil
}

// setupSystemEnvironment creates signoz user, directories, copies configs, and validates binaries.
func (c *systemdCasting) setupSystemEnvironment(ctx context.Context, config *v1alpha1.Casting, serviceMap map[string][]string, poursPath string) error {
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
	_ = c.execCommand(ctx, "chown", "-R", "signoz:signoz", poursPath) // best effort

	// Copy clickhouse configs to standard locations
	if config.Spec.TelemetryStore.Spec.Enabled {
		if err := c.copyDir(filepath.Join(poursPath, config.Spec.TelemetryStore.Kind.String()), "/etc/clickhouse-server/"); err != nil {
			return fmt.Errorf("failed to copy clickhouse-server configs: %w", err)
		}
	}
	if config.Spec.TelemetryKeeper.Spec.Enabled {
		if err := c.copyDir(filepath.Join(poursPath, config.Spec.TelemetryKeeper.Kind.String()), "/etc/clickhouse-keeper/"); err != nil {
			return fmt.Errorf("failed to copy clickhouse-keeper configs: %w", err)
		}
	}

	// Validate required binaries
	return c.validateBinaries(serviceMap)
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

// validateBinaries checks if required binaries exist.
func (c *systemdCasting) validateBinaries(serviceMap map[string][]string) error {
	checks := []struct {
		category, path, name string
	}{
		{"signoz", "/opt/signoz/bin/signoz", "signoz"},
		{"ingester", "/opt/ingester/bin/signoz-otel-collector", "signoz-otel-collector"},
	}

	var missing []string
	for _, chk := range checks {
		if len(serviceMap[chk.category]) > 0 {
			if _, err := os.Stat(chk.path); os.IsNotExist(err) {
				missing = append(missing, chk.name)
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing binaries: %s - please install before running cast", strings.Join(missing, ", "))
	}
	return nil
}

// initAndStartFoundation initializes postgres if needed and starts foundation services.
func (c *systemdCasting) initAndStartFoundation(ctx context.Context, config *v1alpha1.Casting, serviceMap map[string][]string) error {
	// Initialize PostgreSQL if postgres services exist
	if len(serviceMap["postgres"]) > 0 {
		if err := c.initializePostgres(ctx, config); err != nil {
			return err
		}
	}

	// Start foundation services
	return c.startServicesByCategory(ctx, serviceMap, []string{"keeper", "store", "postgres"})
}

// runMigrationsAndStartApps waits for foundation, runs migrations, and starts app services.
func (c *systemdCasting) runMigrationsAndStartApps(ctx context.Context, config *v1alpha1.Casting, serviceMap map[string][]string, poursPath string) error {
	// Wait for store and keeper services to be active
	waitFor := append(serviceMap["store"], serviceMap["keeper"]...)
	if err := c.waitForServices(ctx, waitFor); err != nil {
		return fmt.Errorf("services not ready for migration: %w", err)
	}

	// Run migrations if telemetry store is enabled
	if config.Spec.TelemetryStore.Spec.Enabled {
		// Wait for ClickHouse to accept connections
		if err := c.waitForClickHouse(ctx, config); err != nil {
			return fmt.Errorf("clickhouse not ready: %w", err)
		}

		c.logger.InfoContext(ctx, "Running database migrations")
		if err := c.runMigrator(ctx, config, poursPath); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// Start application services
	return c.startServicesByCategory(ctx, serviceMap, []string{"signoz", "ingester"})
}

// startServicesByCategory enables and starts services for given categories.
func (c *systemdCasting) startServicesByCategory(ctx context.Context, serviceMap map[string][]string, categories []string) error {
	for _, cat := range categories {
		for _, svc := range serviceMap[cat] {
			unitName := filepath.Base(svc)
			c.logger.DebugContext(ctx, "Enabling service", slog.String("service", svc))
			if err := c.execCommand(ctx, "systemctl", "enable", svc); err != nil {
				return fmt.Errorf("failed to enable service %s: %w", svc, err)
			}
			c.logger.InfoContext(ctx, "Starting service", slog.String("service", unitName))
			startCtx, cancel := context.WithTimeout(ctx, serviceStartTimeout)
			err := c.execCommand(startCtx, "systemctl", "start", unitName)
			cancel()
			if err != nil {
				return fmt.Errorf("failed to start service %s: %w", unitName, err)
			}
		}
	}
	return nil
}

// waitForServices waits for all services to be active.
func (c *systemdCasting) waitForServices(ctx context.Context, services []string) error {
	if len(services) == 0 {
		return nil
	}

	deadline := time.Now().Add(serviceReadyWait)
	for {
		allActive := true
		for _, svc := range services {
			cmd := exec.CommandContext(ctx, "systemctl", "is-active", filepath.Base(svc))
			if err := cmd.Run(); err != nil {
				allActive = false
				break
			}
		}
		if allActive {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for services to be active")
		}
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// waitForClickHouse waits for ClickHouse to accept TCP connections.
func (c *systemdCasting) waitForClickHouse(ctx context.Context, config *v1alpha1.Casting) error {
	addrs := config.Spec.TelemetryStore.Status.Addresses.TCP
	if len(addrs) == 0 {
		return fmt.Errorf("no clickhouse addresses configured")
	}

	// Extract host:port from the address (format: tcp://host:port)
	addr := addrs[0]
	addr = strings.TrimPrefix(addr, "tcp://")

	c.logger.DebugContext(ctx, "Waiting for ClickHouse to be ready", slog.String("address", addr))

	deadline := time.Now().Add(clickhouseReadyWait)
	for {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			_ = conn.Close()
			c.logger.DebugContext(ctx, "ClickHouse is ready")
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for clickhouse at %s: %w", addr, err)
		}

		c.logger.DebugContext(ctx, "ClickHouse not ready, retrying...", slog.String("error", err.Error()))
		select {
		case <-time.After(clickhouseRetryDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// runMigrator finds and runs the schema migrator.
func (c *systemdCasting) runMigrator(ctx context.Context, config *v1alpha1.Casting, poursPath string) error {
	// Find migrator binary
	migratorBinary := ""
	paths := []string{
		"/usr/bin/signoz-schema-migrator",
		"/usr/local/bin/signoz-schema-migrator",
		filepath.Join(poursPath, "signoz-schema-migrator/bin/signoz-schema-migrator"),
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			migratorBinary = p
			break
		}
	}
	if migratorBinary == "" {
		if _, err := exec.LookPath("signoz-schema-migrator"); err != nil {
			return fmt.Errorf("signoz-schema-migrator binary not found")
		}
		migratorBinary = "signoz-schema-migrator"
	}

	// Get DSN
	var dsn string
	if addrs := config.Spec.TelemetryStore.Status.Addresses.TCP; len(addrs) > 0 {
		dsn = addrs[0]
	}

	// Run migrations
	c.logger.DebugContext(ctx, "Running migrator sync")
	if err := c.execCommandSilent(ctx, migratorBinary, "sync", "--dsn="+dsn, "--replication=true", "--up="); err != nil {
		return fmt.Errorf("migrator sync failed: %w", err)
	}
	c.logger.DebugContext(ctx, "Running migrator async")
	if err := c.execCommandSilent(ctx, migratorBinary, "async", "--dsn="+dsn, "--replication=true", "--up="); err != nil {
		return fmt.Errorf("migrator async failed: %w", err)
	}
	return nil
}

// execCommandSilent executes a command without outputting to stdout/stderr.
func (c *systemdCasting) execCommandSilent(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Run()
}

// initializePostgres sets up the PostgreSQL data directory.
func (c *systemdCasting) initializePostgres(ctx context.Context, config *v1alpha1.Casting) error {
	pgDataDir := "/usr/local/pgsql/data"

	// Skip if already initialized
	if _, err := os.Stat(pgDataDir); err == nil {
		c.logger.DebugContext(ctx, "PostgreSQL already initialized", slog.String("path", pgDataDir))
		return nil
	}

	c.logger.InfoContext(ctx, "Initializing PostgreSQL")

	// Create directories
	if err := os.MkdirAll(pgDataDir, 0700); err != nil {
		return fmt.Errorf("failed to create PostgreSQL data directory: %w", err)
	}
	if err := c.execCommand(ctx, "chown", "-R", "postgres:postgres", pgDataDir); err != nil {
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
	pwfile := "/tmp/postgres_pwfile_init"
	if err := os.WriteFile(pwfile, []byte(pgPass+"\n"), 0600); err != nil {
		return fmt.Errorf("failed to create password file: %w", err)
	}
	_ = c.execCommand(ctx, "chown", "postgres:postgres", pwfile)

	// Initialize database
	c.logger.DebugContext(ctx, "Running initdb", slog.String("user", pgUser))
	if err := c.execCommand(ctx, "su", "-", "postgres", "-c",
		fmt.Sprintf("initdb -D %s --username=%s --pwfile=%s", pgDataDir, pgUser, pwfile)); err != nil {
		return fmt.Errorf("failed to initialize PostgreSQL: %w", err)
	}

	// Start temp server and create database
	c.logger.DebugContext(ctx, "Starting temporary PostgreSQL for DB creation")
	if err := c.execCommand(ctx, "su", "-", "postgres", "-c",
		fmt.Sprintf("pg_ctl -D %s -o \"-c listen_addresses=localhost\" -w start", pgDataDir)); err != nil {
		return fmt.Errorf("failed to start temporary postgres: %w", err)
	}

	// Create database
	c.logger.DebugContext(ctx, "Creating database", slog.String("database", dbName))
	cmd := exec.CommandContext(ctx, "psql", "-U", pgUser, "-h", "localhost", "-d", "postgres", "-c", fmt.Sprintf("CREATE DATABASE %s;", dbName))
	cmd.Env = append(os.Environ(), "PGPASSWORD="+pgPass)
	_ = cmd.Run() // ignore error - database may already exist

	// Stop temporary PostgreSQL
	if err := c.execCommand(ctx, "su", "-", "postgres", "-c", fmt.Sprintf("pg_ctl -D %s -m fast -w stop", pgDataDir)); err != nil {
		return fmt.Errorf("failed to stop temporary postgres: %w", err)
	}

	// Clean up password file
	if err := os.Remove(pwfile); err != nil {
		return fmt.Errorf("failed to remove password file: %w", err)
	}

	return nil
}
