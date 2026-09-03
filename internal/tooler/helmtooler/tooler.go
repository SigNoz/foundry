// Package helmtooler speaks helm.
package helmtooler

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/tooler"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	helmrelease "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/transport"
)

var _ tooler.Tooler = (*Tooler)(nil)

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

	// The zero value is the ambient kubeconfig, which is also how the
	// in-cluster case resolves.
	Connection tooler.Connection
}

func (r Release) Validate() error {
	if err := r.Release.Validate(); err != nil {
		return err
	}

	if r.Namespace == "" {
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "failed to validate release: no namespace is stated")
	}

	return nil
}

type Tooler struct {
	tooler.Tool
}

func New(logger *slog.Logger) *Tooler {
	return &Tooler{Tool: tooler.NewTool("helm", logger)}
}

func Lookup(toolers []tooler.Tooler) (*Tooler, error) {
	for _, t := range toolers {
		if helm, ok := t.(*Tooler); ok {
			return helm, nil
		}
	}

	return nil, foundryerrors.Newf(foundryerrors.TypeNotFound, "failed to look up the helm tooler: it is not registered for this casting")
}

// Gauge proves a kubeconfig exists and parses, not that a cluster answers:
// reach is a per-verb question.
func (t *Tooler) Gauge(ctx context.Context) error {
	if _, err := cli.New().RESTClientGetter().ToRESTConfig(); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeNotFound, "failed to reach a cluster: no kubeconfig resolved")
	}

	return nil
}

// Upgrade is helm's own upgrade --install: action.Upgrade.Install is
// informative only, so the caller dispatches install-vs-upgrade.
func (t *Tooler) Upgrade(ctx context.Context, release Release) error {
	if err := release.Validate(); err != nil {
		return err
	}

	if release.Chart == "" {
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "failed to run helm upgrade: no chart is stated")
	}

	env, config, err := t.configure(release)
	if err != nil {
		return err
	}

	if release.Repo.Name != "" && release.Repo.URL != "" {
		if err := addRepo(env, release.Repo); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to run helm upgrade: could not add repo %q", release.Repo.Name)
		}
	}

	found, err := t.claim(ctx, config, release.Name, release.Owner)
	if err != nil {
		return err
	}

	if !found {
		return t.install(ctx, env, config, release)
	}

	return t.upgrade(ctx, env, config, release)
}

// Uninstall has no context-aware form in the SDK, so ctx is honored at entry
// alone; the removal cannot abort midway.
func (t *Tooler) Uninstall(ctx context.Context, release Release) error {
	if err := release.Validate(); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to run helm uninstall")
	}

	_, config, err := t.configure(release)
	if err != nil {
		return err
	}

	if _, err := t.claim(ctx, config, release.Name, release.Owner); err != nil {
		return err
	}

	uninstall := action.NewUninstall(config)
	uninstall.Wait = true

	if _, err := uninstall.Run(release.Name); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to run helm uninstall")
	}

	return nil
}

func (t *Tooler) install(ctx context.Context, env *cli.EnvSettings, config *action.Configuration, release Release) error {
	install := action.NewInstall(config)
	install.ReleaseName = release.Name
	install.Namespace = release.Namespace
	install.CreateNamespace = true
	install.Wait = true
	install.Labels = release.Owner

	chart, err := loadChart(env, install.LocateChart, release.Chart)
	if err != nil {
		return err
	}

	if _, err := install.RunWithContext(ctx, chart, release.Values); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to run helm install")
	}

	return nil
}

func (t *Tooler) upgrade(ctx context.Context, env *cli.EnvSettings, config *action.Configuration, release Release) error {
	upgrade := action.NewUpgrade(config)
	upgrade.Install = true
	upgrade.Namespace = release.Namespace
	upgrade.Wait = true
	upgrade.Labels = release.Owner

	chart, err := loadChart(env, upgrade.LocateChart, release.Chart)
	if err != nil {
		return err
	}

	if _, err := upgrade.RunWithContext(ctx, release.Name, chart, release.Values); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to run helm upgrade")
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

// verify compares only the keys foundry stamps: a release also carries helm's
// own system labels.
func (t *Tooler) verify(ctx context.Context, rel *helmrelease.Release, owner domain.Owner) error {
	if len(owner) == 0 {
		return nil
	}

	return tooler.Verify(ctx, t.Tool, domain.Release{Name: rel.Name, Owner: owner}, func(context.Context) (domain.Ownership, error) {
		return domain.NewOwnership(owner.Read(rel.Labels)), nil
	})
}

func loadChart(env *cli.EnvSettings, locate func(string, *cli.EnvSettings) (string, error), ref string) (*chart.Chart, error) {
	path, err := locate(ref, env)
	if err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeNotFound, "failed to locate chart %q", ref)
	}

	loaded, err := loader.Load(path)
	if err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to load chart %q", ref)
	}

	return loaded, nil
}

func addRepo(env *cli.EnvSettings, r Repo) error {
	entry := &repo.Entry{Name: r.Name, URL: r.URL}

	chartRepo, err := repo.NewChartRepository(entry, getter.All(env))
	if err != nil {
		return err
	}

	chartRepo.CachePath = env.RepositoryCache
	if _, err := chartRepo.DownloadIndexFile(); err != nil {
		return err
	}

	file, err := repo.LoadFile(env.RepositoryConfig)
	if err != nil {
		file = repo.NewFile()
	}

	file.Update(entry)

	return file.WriteFile(env.RepositoryConfig, 0o644)
}

func (t *Tooler) configure(release Release) (*cli.EnvSettings, *action.Configuration, error) {
	env := cli.New()
	env.SetNamespace(release.Namespace)

	config := new(action.Configuration)
	if err := config.Init(clientGetter(env, release), release.Namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...any) {
		_, _ = fmt.Fprintf(t.Settings.Sink(), format+"\n", v...)
	}); err != nil {
		return nil, nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to initialize helm: the cluster is not reachable")
	}

	return env, config, nil
}

// A stated connection means helm never consults a kubeconfig foundry did not
// give it.
func clientGetter(env *cli.EnvSettings, release Release) genericclioptions.RESTClientGetter {
	if release.Connection.IsZero() {
		return env.RESTClientGetter()
	}

	config := &rest.Config{Host: release.Connection.Address().String()}
	config.CAData = release.Connection.CA()

	// The token is minted per request, never once: an EKS token outlives
	// neither a slow install nor a wait.
	config.Wrap(transport.TokenSourceWrapTransport(transport.NewCachedTokenSource(release.Connection.TokenSource())))

	return restGetter{config: config, namespace: release.Namespace}
}

type restGetter struct {
	config *rest.Config

	namespace string
}

func (c restGetter) ToRESTConfig() (*rest.Config, error) {
	return c.config, nil
}

func (c restGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	client, err := discovery.NewDiscoveryClientForConfig(c.config)
	if err != nil {
		return nil, foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to build the kubernetes client")
	}

	return memory.NewMemCacheClient(client), nil
}

func (c restGetter) ToRESTMapper() (meta.RESTMapper, error) {
	client, err := c.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	return restmapper.NewDeferredDiscoveryRESTMapper(client), nil
}

// An exact connection has no kubeconfig behind it, so the loader carries the
// namespace alone.
func (c restGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	overrides := &clientcmd.ConfigOverrides{Context: clientcmdapi.Context{Namespace: c.namespace}}

	return clientcmd.NewDefaultClientConfig(*clientcmdapi.NewConfig(), overrides)
}
