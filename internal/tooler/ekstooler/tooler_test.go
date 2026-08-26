package ekstooler

import (
	"context"
	"encoding/base64"
	"log/slog"
	"strings"
	"testing"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/tooler"
)

func requireCredentials(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping aws test in short mode")
	}

	config, err := awsconfig.LoadDefaultConfig(context.Background())
	if err != nil {
		t.Skip("no aws configuration resolved")
	}

	if _, err := config.Credentials.Retrieve(context.Background()); err != nil {
		t.Skip("no aws credentials resolved")
	}
}

type otherTooler struct{}

func (otherTooler) Name() string                  { return "other" }
func (otherTooler) Gauge(_ context.Context) error { return nil }

func TestNew(t *testing.T) {
	eks := New(slog.New(slog.DiscardHandler))

	assert.NotNil(t, eks)
	assert.Equal(t, "eks", eks.Name())
}

func TestLookup(t *testing.T) {
	eks := New(slog.New(slog.DiscardHandler))

	tests := []struct {
		name    string
		toolers []tooler.Tooler
		pass    bool
	}{
		{name: "Only_Valid", toolers: []tooler.Tooler{eks}, pass: true},
		{name: "AmongOthers_Valid", toolers: []tooler.Tooler{otherTooler{}, eks}, pass: true},
		{name: "Empty_Invalid", toolers: nil, pass: false},
		{name: "OnlyOthers_Invalid", toolers: []tooler.Tooler{otherTooler{}}, pass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := Lookup(tt.toolers)

			if !tt.pass {
				assert.Error(t, err)
				assert.Nil(t, found)

				return
			}

			assert.NoError(t, err)
			assert.Same(t, eks, found)
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		release Release
		pass    bool
	}{
		{name: "Cluster_Valid", release: Release{Release: domain.Release{Name: "signoz-eks"}}, pass: true},
		{name: "ClusterAndRegion_Valid", release: Release{Release: domain.Release{Name: "signoz-eks"}, Region: "us-east-1"}, pass: true},
		{name: "UnstatedCluster_Invalid", release: Release{}},
		{name: "RegionAlone_Invalid", release: Release{Region: "us-east-1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.release.Validate()

			if !tt.pass {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestVerbsRequireCluster(t *testing.T) {
	eks := New(slog.New(slog.DiscardHandler))

	_, err := eks.DescribeCluster(context.Background(), Release{Region: "us-east-1"})
	assert.Error(t, err)

	_, _, err = eks.GetToken(context.Background(), Release{Region: "us-east-1"})
	assert.Error(t, err)
}

// The prefix, encoding, and signed header are the authenticator's wire
// format; the cluster name travels in the header, not the body.
func TestGetTokenShape(t *testing.T) {
	requireCredentials(t)

	eks := New(slog.New(slog.DiscardHandler))

	token, expiry, err := eks.GetToken(context.Background(), Release{Release: domain.Release{Name: "signoz-eks"}, Region: "us-east-1"})
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(token, tokenPrefix))
	assert.True(t, expiry.After(time.Now()))

	url, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(token, tokenPrefix))
	require.NoError(t, err)

	assert.Contains(t, string(url), "Action=GetCallerIdentity")
	assert.Contains(t, string(url), strings.ToLower(clusterIDHeader))
}

func TestGaugeReachesCredentials(t *testing.T) {
	requireCredentials(t)

	assert.NoError(t, New(slog.New(slog.DiscardHandler)).Gauge(context.Background()))
}
