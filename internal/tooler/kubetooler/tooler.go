// Package kubetooler speaks the Kubernetes API.
package kubetooler

import (
	"context"
	"io"
	"log/slog"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/tooler"
)

var _ tooler.Tooler = (*Tooler)(nil)

type Release struct {
	domain.Release

	Namespace string

	// Dir is the kustomize root rendered and applied.
	Dir string

	// FieldManager is who the API server records as owning the fields this
	// apply sets. The casting composes it, so two castings never contend over
	// a cluster-scoped object.
	FieldManager string

	// The zero value is the ambient kubeconfig, which is also how the
	// in-cluster case resolves.
	Connection tooler.Connection
}

func (r Release) Validate() error {
	if err := r.Release.Validate(); err != nil {
		return err
	}

	if r.Namespace == "" {
		return errors.Newf(errors.TypeInvalidInput, "failed to validate release: no namespace is stated")
	}

	if r.Dir == "" {
		return errors.Newf(errors.TypeInvalidInput, "failed to validate release: no directory is stated")
	}

	if r.FieldManager == "" {
		return errors.Newf(errors.TypeInvalidInput, "failed to validate release: no field manager is stated")
	}

	return nil
}

type Tooler struct {
	tooler.Tool

	// client is the dialed cluster, memoized by dial: every verb of a run
	// speaks the one connection the casting states.
	client *client
}

func New(logger *slog.Logger) *Tooler {
	return &Tooler{Tool: tooler.NewTool("kubernetes", logger)}
}

func Lookup(toolers []tooler.Tooler) (*Tooler, error) {
	for _, t := range toolers {
		if kube, ok := t.(*Tooler); ok {
			return kube, nil
		}
	}

	return nil, errors.Newf(errors.TypeNotFound, "failed to look up the kubernetes tooler: it is not registered for this casting")
}

// Gauge proves a kubeconfig exists and parses, not that a cluster answers:
// reach is a per-verb question.
func (t *Tooler) Gauge(ctx context.Context) error {
	_, err := t.restConfig(tooler.Connection{})

	return err
}

// Apply writes the objects in the order the render produced: which objects
// depend on which is the casting's knowledge, stated by its pour structure.
func (t *Tooler) Apply(ctx context.Context, release Release) error {
	if err := release.Validate(); err != nil {
		return err
	}

	objects, err := render(release.Dir)
	if err != nil {
		return err
	}

	client, err := t.connection(release.Connection)
	if err != nil {
		return err
	}

	if err := tooler.Verify(ctx, t.Tool, release.Release, func(ctx context.Context) (domain.Ownership, error) {
		return t.read(ctx, release, client, objects)
	}); err != nil {
		return err
	}

	return t.apply(ctx, release, client, objects)
}

// Delete removes what this release declares, less the kinds that carry data:
// melt never removes data.
func (t *Tooler) Delete(ctx context.Context, release Release) error {
	if err := release.Validate(); err != nil {
		return err
	}

	objects, err := render(release.Dir)
	if err != nil {
		return err
	}

	client, err := t.connection(release.Connection)
	if err != nil {
		return err
	}

	if err := tooler.Verify(ctx, t.Tool, release.Release, func(ctx context.Context) (domain.Ownership, error) {
		return t.read(ctx, release, client, objects)
	}); err != nil {
		return err
	}

	// Background is what kubectl delete sends: the server's own default
	// orphans a Job's pods, leaving them running with nothing owning them.
	policy := metav1.DeletePropagationBackground

	for _, object := range objects {
		if undeletable(object.GetKind()) {
			continue
		}

		resource, err := client.resourceFor(object, release.Namespace)
		if err != nil {
			return err
		}

		t.Logger.DebugContext(ctx, "deleting object",
			slog.String("kind", object.GetKind()),
			slog.String("name", object.GetName()),
		)

		if err := resource.Delete(ctx, object.GetName(), metav1.DeleteOptions{PropagationPolicy: &policy}); err != nil && !apierrors.IsNotFound(err) {
			return errors.Wrapf(err, errors.TypeInternal, "failed to delete %s %q", object.GetKind(), object.GetName())
		}
	}

	return nil
}

// Owners reads the release's footprint back from the objects it declares;
// asking every kind the cluster serves is not affordable.
func (t *Tooler) Owners(ctx context.Context, release Release) (domain.Ownership, error) {
	if err := release.Release.Validate(); err != nil {
		return domain.Ownership{}, err
	}

	objects, err := render(release.Dir)
	if err != nil {
		return domain.Ownership{}, err
	}

	client, err := t.connection(release.Connection)
	if err != nil {
		return domain.Ownership{}, err
	}

	return t.read(ctx, release, client, objects)
}

func (t *Tooler) apply(ctx context.Context, release Release, client *client, objects []*unstructured.Unstructured) error {
	for _, object := range objects {
		resource, err := client.resourceFor(object, release.Namespace)
		if err != nil {
			return err
		}

		t.Logger.DebugContext(ctx, "applying object",
			slog.String("kind", object.GetKind()),
			slog.String("name", object.GetName()),
			slog.String("fieldManager", release.FieldManager),
		)

		// Force resolves conflicts toward the document: the pours are the
		// declared state, not the cluster.
		options := metav1.ApplyOptions{FieldManager: release.FieldManager, Force: true}

		if _, err := resource.Apply(ctx, object.GetName(), object, options); err != nil {
			return errors.Wrapf(err, errors.TypeInternal, "failed to apply %s %q", object.GetKind(), object.GetName())
		}
	}

	return nil
}

// The declared objects are read by identity, never by the caller's own labels:
// selecting on the owner would hide exactly the foreign owner the check exists
// to find.
func (t *Tooler) read(ctx context.Context, release Release, client *client, objects []*unstructured.Unstructured) (domain.Ownership, error) {
	owners := make([]domain.Owner, 0, len(objects))
	for _, object := range objects {
		resource, err := client.resourceFor(object, release.Namespace)
		if err != nil {
			// A kind the cluster does not serve yet holds nothing of ours.
			continue
		}

		found, err := resource.Get(ctx, object.GetName(), metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) || apierrors.IsForbidden(err) {
				continue
			}

			return domain.Ownership{}, errors.Wrapf(err, errors.TypeInternal, "failed to read %s %q", object.GetKind(), object.GetName())
		}

		owners = append(owners, release.Owner.Read(found.GetLabels()))
	}

	return domain.NewOwnership(owners...), nil
}

func render(dir string) ([]*unstructured.Unstructured, error) {
	resources, err := krusty.MakeKustomizer(krusty.MakeDefaultOptions()).Run(filesys.MakeFsOnDisk(), dir)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to render the kustomization at %q", dir)
	}

	rendered, err := resources.AsYaml()
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to render the kustomization at %q", dir)
	}

	decoder := utilyaml.NewYAMLToJSONDecoder(strings.NewReader(string(rendered)))

	objects := []*unstructured.Unstructured{}
	for {
		object := &unstructured.Unstructured{}

		if err := decoder.Decode(object); err != nil {
			if err == io.EOF {
				break
			}

			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to read the rendered kustomization at %q", dir)
		}

		if len(object.Object) == 0 {
			continue
		}

		objects = append(objects, object)
	}

	return objects, nil
}

// undeletable reports a kind Delete leaves standing whatever the casting
// declares: a Namespace or a PersistentVolumeClaim takes the data with it,
// and a CustomResourceDefinition is shared with every release in the cluster.
func undeletable(kind string) bool {
	switch kind {
	case "CustomResourceDefinition", "Namespace", "PersistentVolumeClaim":
		return true
	default:
		return false
	}
}

type client struct {
	dynamic   dynamic.Interface
	discovery discovery.CachedDiscoveryInterface
	mapper    *restmapper.DeferredDiscoveryRESTMapper
}

// The memo is unguarded; foundry runs single-threaded, and a lock would claim a
// concurrency contract toolers do not have.
func (t *Tooler) connection(connection tooler.Connection) (*client, error) {
	if t.client != nil {
		return t.client, nil
	}

	config, err := t.restConfig(connection)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to build the kubernetes client")
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to build the kubernetes client")
	}

	cached := memory.NewMemCacheClient(discoveryClient)

	t.client = &client{
		dynamic:   dynamicClient,
		discovery: cached,
		mapper:    restmapper.NewDeferredDiscoveryRESTMapper(cached),
	}

	return t.client, nil
}

// An unstated connection is the ambient kubeconfig; a stated one is built from
// scratch. The API server's warnings are the tool's own words either way, so
// they go where every other tool's output goes rather than to client-go's
// default logger.
func (t *Tooler) restConfig(connection tooler.Connection) (*rest.Config, error) {
	var config *rest.Config

	if connection.IsZero() {
		resolved, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{},
		).ClientConfig()
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeNotFound, "failed to reach a cluster: no kubeconfig resolved")
		}

		config = resolved
	} else {
		config = &rest.Config{Host: connection.Address().String()}
		config.CAData = connection.CA()

		// The token is minted per request, never once: an EKS token outlives
		// neither a slow apply nor a wait.
		config.Wrap(transport.TokenSourceWrapTransport(transport.NewCachedTokenSource(connection.TokenSource())))
	}

	config.WarningHandler = rest.NewWarningWriter(t.Settings.Sink(), rest.WarningWriterOptions{Deduplicate: true})

	return config, nil
}

// An object the render already placed is addressed where it says it lives: the
// path and the body must agree or the API server refuses the write.
func (c *client) resourceFor(object *unstructured.Unstructured, namespace string) (dynamic.ResourceInterface, error) {
	if placed := object.GetNamespace(); placed != "" {
		namespace = placed
	}

	gvk := object.GroupVersionKind()

	mapping, err := c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		// A definition applied moments ago is not in the caches yet; one
		// refresh tells staleness apart from a kind the cluster does not serve.
		c.discovery.Invalidate()
		c.mapper.Reset()

		if mapping, err = c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version); err != nil {
			return nil, errors.Wrapf(err, errors.TypeNotFound, "failed to find the kubernetes resource for %s", gvk.Kind)
		}
	}

	if mapping.Scope.Name() == meta.RESTScopeNameRoot {
		return c.dynamic.Resource(mapping.Resource), nil
	}

	return c.dynamic.Resource(mapping.Resource).Namespace(namespace), nil
}
