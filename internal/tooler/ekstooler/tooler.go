// Package awstooler speaks the AWS control plane.
package ekstooler

import (
	"context"
	"encoding/base64"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	ekstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go/logging"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/tooler"
)

var _ tooler.Tooler = (*Tooler)(nil)

const (
	// tokenPrefix and clusterIDHeader are the wire format the cluster's
	// authenticator expects; neither is ours to choose.
	tokenPrefix     = "k8s-aws-v1."
	clusterIDHeader = "x-k8s-aws-id"

	// tokenLifetime is what aws eks get-token reports: the presigned request is
	// valid for fifteen minutes, and the last minute is left unclaimed.
	tokenLifetime = 14 * time.Minute
)

// Release names the cluster to read. It carries no credential: the connection
// is the environment's, and it reads rather than owns, so no owner is stamped.
type Release struct {
	domain.Release

	Region string
}

func (r Release) Validate() error {
	if r.Name == "" {
		return errors.Newf(errors.TypeInvalidInput, "failed to validate release: no cluster is stated")
	}

	return nil
}

// Cluster carries no secret: its CA is public material.
type Cluster struct {
	Endpoint domain.Address

	CA []byte
}

type Tooler struct {
	tooler.Tool
}

func New(logger *slog.Logger) *Tooler {
	return &Tooler{Tool: tooler.NewTool("eks", logger)}
}

func Lookup(toolers []tooler.Tooler) (*Tooler, error) {
	for _, t := range toolers {
		if eks, ok := t.(*Tooler); ok {
			return eks, nil
		}
	}

	return nil, errors.Newf(errors.TypeNotFound, "failed to look up the eks tooler: it is not registered for this casting")
}

// Gauge has no binary to find: reach is whether the credential chain
// resolves at all, independent of any single cluster or region.
func (t *Tooler) Gauge(ctx context.Context) error {
	config, err := t.load(ctx, "")
	if err != nil {
		return err
	}

	if _, err := config.Credentials.Retrieve(ctx); err != nil {
		return errors.Wrapf(err, errors.TypeNotFound, "failed to find aws credentials: configure them from https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html")
	}

	return nil
}

// DescribeCluster refuses a cluster that is not yet serving: an endpoint from a
// cluster still creating would be a connection that cannot answer.
func (t *Tooler) DescribeCluster(ctx context.Context, release Release) (Cluster, error) {
	if err := release.Validate(); err != nil {
		return Cluster{}, err
	}

	config, err := t.load(ctx, release.Region)
	if err != nil {
		return Cluster{}, err
	}

	t.Logger.DebugContext(ctx, "describing eks cluster",
		slog.String("cluster", release.Name),
		slog.String("region", config.Region),
	)

	described, err := eks.NewFromConfig(config).DescribeCluster(ctx, &eks.DescribeClusterInput{Name: &release.Name})
	if err != nil {
		return Cluster{}, errors.Wrapf(err, errors.TypeNotFound, "failed to describe the eks cluster %q in region %q", release.Name, config.Region)
	}

	cluster := described.Cluster

	if cluster.Status != ekstypes.ClusterStatusActive {
		return Cluster{}, errors.Newf(errors.TypeInvalidInput, "failed to describe the eks cluster %q: it is %s, not ACTIVE", release.Name, cluster.Status)
	}

	if cluster.Endpoint == nil || cluster.CertificateAuthority == nil || cluster.CertificateAuthority.Data == nil {
		return Cluster{}, errors.Newf(errors.TypeInternal, "failed to describe the eks cluster %q: it reports no endpoint or certificate authority", release.Name)
	}

	authority, err := base64.StdEncoding.DecodeString(*cluster.CertificateAuthority.Data)
	if err != nil {
		return Cluster{}, errors.Wrapf(err, errors.TypeInternal, "failed to read the certificate authority of the eks cluster %q", release.Name)
	}

	endpoint, err := domain.ParseAddress(*cluster.Endpoint)
	if err != nil {
		return Cluster{}, errors.Wrapf(err, errors.TypeInternal, "failed to read the endpoint of the eks cluster %q", release.Name)
	}

	return Cluster{Endpoint: endpoint, CA: authority}, nil
}

// GetToken mints what aws eks get-token mints: a presigned GetCallerIdentity
// request the cluster runs to identify the caller. It refuses only when the
// credential chain fails, since presigning itself needs no permission.
func (t *Tooler) GetToken(ctx context.Context, release Release) (string, time.Time, error) {
	if err := release.Validate(); err != nil {
		return "", time.Time{}, err
	}

	config, err := t.load(ctx, release.Region)
	if err != nil {
		return "", time.Time{}, err
	}

	presigner := sts.NewPresignClient(sts.NewFromConfig(config), func(o *sts.PresignOptions) {
		o.ClientOptions = append(o.ClientOptions, func(client *sts.Options) {
			client.APIOptions = append(client.APIOptions, smithyhttp.SetHeaderValue(clusterIDHeader, release.Name))
		})
	})

	presigned, err := presigner.PresignGetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", time.Time{}, errors.Wrapf(err, errors.TypeInternal, "failed to mint a token for the eks cluster %q", release.Name)
	}

	return tokenPrefix + base64.RawURLEncoding.EncodeToString([]byte(presigned.URL)), time.Now().Add(tokenLifetime), nil
}

// load leaves an unstated region to the environment's own resolution chain.
func (t *Tooler) load(ctx context.Context, region string) (aws.Config, error) {
	loaded := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithLogger(logging.NewStandardLogger(t.Settings.Sink())),
	}
	if region != "" {
		loaded = append(loaded, awsconfig.WithRegion(region))
	}

	config, err := awsconfig.LoadDefaultConfig(ctx, loaded...)
	if err != nil {
		return aws.Config{}, errors.Wrapf(err, errors.TypeNotFound, "failed to find aws credentials: configure them from https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html")
	}

	return config, nil
}
