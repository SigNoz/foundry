package helmtooler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/tooler"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	helmrelease "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
)

var _ tooler.Tooler = (*Tooler)(nil)

// deployTimeout is helm's own --timeout, not a clock foundry puts on the work.
const deployTimeout = 10 * time.Minute

type Repo struct {
	Name string
	URL  string
}

type Release struct {
	domain.Release

	Namespace string

	// Chart is a local chart path or a repo-qualified name ("signoz/signoz").
	Chart string

	Repo Repo

	Values map[string]any
}

func (r Release) Validate() error {
	if err := r.Release.Validate(); err != nil {
		return err
	}

	if r.Namespace == "" {
		return errors.Newf(errors.TypeInvalidInput, "failed to validate release: no namespace is stated")
	}

	return nil
}

type Tooler struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Tooler {
	return &Tooler{logger: logger}
}

func Lookup(toolers []tooler.Tooler) (*Tooler, error) {
	for _, t := range toolers {
		if helm, ok := t.(*Tooler); ok {
			return helm, nil
		}
	}

	return nil, errors.Newf(errors.TypeNotFound, "failed to look up the helm tooler: it is not registered for this casting")
}

func (t *Tooler) Name() string {
	return "helm"
}

// Gauge has no reach check: helm is the SDK, so there is no binary to find.
func (t *Tooler) Gauge(ctx context.Context) error {
	return nil
}

// Upgrade is helm's own upgrade --install: action.Upgrade.Install is
// informative only, so the caller dispatches install-vs-upgrade.
func (t *Tooler) Upgrade(ctx context.Context, release Release) error {
	if err := release.Validate(); err != nil {
		return err
	}

	if release.Chart == "" {
		return errors.Newf(errors.TypeInvalidInput, "failed to run helm upgrade: no chart is stated")
	}

	settings, config, err := t.configure(ctx, release.Namespace)
	if err != nil {
		return err
	}

	if release.Repo.Name != "" && release.Repo.URL != "" {
		if err := addRepo(settings, release.Repo); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to run helm upgrade: could not add repo %q", release.Repo.Name)
		}
	}

	found, err := t.claim(ctx, config, release.Name, release.Owner)
	if err != nil {
		return err
	}

	if !found {
		return t.install(ctx, settings, config, release)
	}

	return t.upgrade(ctx, settings, config, release)
}

// Uninstall has no context-aware form in the SDK, so ctx is honored at entry
// alone; the removal cannot abort midway.
func (t *Tooler) Uninstall(ctx context.Context, release Release) error {
	if err := release.Validate(); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to run helm uninstall")
	}

	_, config, err := t.configure(ctx, release.Namespace)
	if err != nil {
		return err
	}

	if _, err := t.claim(ctx, config, release.Name, release.Owner); err != nil {
		return err
	}

	uninstall := action.NewUninstall(config)
	uninstall.Wait = true
	uninstall.Timeout = deployTimeout

	if _, err := uninstall.Run(release.Name); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to run helm uninstall")
	}

	return nil
}

func (t *Tooler) install(ctx context.Context, settings *cli.EnvSettings, config *action.Configuration, release Release) error {
	install := action.NewInstall(config)
	install.ReleaseName = release.Name
	install.Namespace = release.Namespace
	install.CreateNamespace = true
	install.Wait = true
	install.Timeout = deployTimeout
	install.Labels = release.Owner

	chart, err := loadChart(settings, install.LocateChart, release.Chart)
	if err != nil {
		return err
	}

	if _, err := install.RunWithContext(ctx, chart, release.Values); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to run helm install")
	}

	return nil
}

func (t *Tooler) upgrade(ctx context.Context, settings *cli.EnvSettings, config *action.Configuration, release Release) error {
	upgrade := action.NewUpgrade(config)
	upgrade.Install = true
	upgrade.Namespace = release.Namespace
	upgrade.Wait = true
	upgrade.Timeout = deployTimeout
	upgrade.Labels = release.Owner

	chart, err := loadChart(settings, upgrade.LocateChart, release.Chart)
	if err != nil {
		return err
	}

	if _, err := upgrade.RunWithContext(ctx, release.Name, chart, release.Values); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to run helm upgrade")
	}

	return nil
}

// verify compares only the keys foundry stamps: a release also carries helm's
// own system labels.
func (t *Tooler) verify(ctx context.Context, rel *helmrelease.Release, owner domain.Owner) error {
	if len(owner) == 0 {
		return nil
	}

	stamped := domain.Owner{}
	for key := range owner {
		stamped[key] = rel.Labels[key]
	}

	ownership := domain.NewOwnership(stamped)

	if foreign, conflict := ownership.Foreign(owner); conflict {
		return errors.Newf(errors.TypeInvalidInput, "failed to run helm: release %q already belongs to [%s], not [%s]: remove it, or deploy under a different name", rel.Name, foreign, owner)
	}

	if ownership.HasUnowned() {
		t.logger.WarnContext(ctx, "helm release has no ownership labels", slog.String("release", rel.Name))
	}

	return nil
}

func (t *Tooler) claim(ctx context.Context, config *action.Configuration, name string, owner domain.Owner) (bool, error) {
	history := action.NewHistory(config)
	history.Max = 1

	releases, err := history.Run(name)
	if err != nil || len(releases) == 0 {
		return false, nil
	}

	return true, t.verify(ctx, releases[len(releases)-1], owner)
}

func (t *Tooler) configure(ctx context.Context, namespace string) (*cli.EnvSettings, *action.Configuration, error) {
	settings := cli.New()
	settings.SetNamespace(namespace)

	config := new(action.Configuration)
	if err := config.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...any) {
		t.logger.DebugContext(ctx, fmt.Sprintf(format, v...))
	}); err != nil {
		return nil, nil, errors.Wrapf(err, errors.TypeInternal, "failed to initialize helm: the cluster is not reachable")
	}

	return settings, config, nil
}

func loadChart(settings *cli.EnvSettings, locate func(string, *cli.EnvSettings) (string, error), ref string) (*chart.Chart, error) {
	path, err := locate(ref, settings)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeNotFound, "failed to locate chart %q", ref)
	}

	loaded, err := loader.Load(path)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to load chart %q", ref)
	}

	return loaded, nil
}

func addRepo(settings *cli.EnvSettings, r Repo) error {
	entry := &repo.Entry{Name: r.Name, URL: r.URL}

	chartRepo, err := repo.NewChartRepository(entry, getter.All(settings))
	if err != nil {
		return err
	}

	chartRepo.CachePath = settings.RepositoryCache
	if _, err := chartRepo.DownloadIndexFile(); err != nil {
		return err
	}

	file, err := repo.LoadFile(settings.RepositoryConfig)
	if err != nil {
		file = repo.NewFile()
	}

	file.Update(entry)

	return file.WriteFile(settings.RepositoryConfig, 0o644)
}
