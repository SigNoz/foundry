package ledger

import "github.com/signoz/foundry/api/v1alpha1"

// key is the Segment write key, set via ldflags at build time.
// Example: -ldflags "-X github.com/signoz/foundry/internal/ledger.key=<key>".
var key string = "<unset>"

// Config holds ledger configuration.
type Config struct {
	Enabled bool
	Segment Segment
}

// Segment holds Segment-specific configuration.
type Segment struct {
	Key string
}

// NewConfig returns the default ledger configuration.
// The Segment write key is populated from the ldflags-injected value.
func NewConfig() Config {
	return Config{
		Enabled: true,
		Segment: Segment{
			Key: key,
		},
	}
}

// Provider returns the provider name based on the configuration.
func (c Config) Provider() string {
	if c.Enabled {
		return "segment"
	}
	return "noop"
}

// Event names for foundryctl commands.
const (
	EventGauge   = "gauge"
	EventForge   = "forge"
	EventCast    = "cast"
	EventCatalog = "catalog"
)

// Property keys for casting details.
const (
	PropPlatform              = "platform"
	PropMode                  = "mode"
	PropFlavor                = "flavor"
	PropPatchesConfigured     = "patches_configured"
	PropPatchCount            = "patch_count"
	PropInfrastructureEnabled = "infrastructure_enabled"
	PropMetaStoreKind         = "metastore_kind"
	PropTelemetryStoreKind    = "telemetry_store_kind"
	PropTelemetryKeeperKind   = "telemetry_keeper_kind"
	PropSuccess               = "success"
	PropError                 = "error"
)

// CastingProperties extracts trackable properties from a Casting config.
func CastingProperties(casting v1alpha1.Casting) map[string]any {
	return map[string]any{
		PropPlatform:              casting.Spec.Deployment.Platform,
		PropMode:                  casting.Spec.Deployment.Mode,
		PropFlavor:                casting.Spec.Deployment.Flavor,
		PropPatchesConfigured:     len(casting.Spec.Patches) > 0,
		PropPatchCount:            len(casting.Spec.Patches),
		PropInfrastructureEnabled: casting.Spec.Infrastructure.Enabled,
		PropMetaStoreKind:         casting.Spec.MetaStore.Kind.String(),
		PropTelemetryStoreKind:    casting.Spec.TelemetryStore.Kind.String(),
		PropTelemetryKeeperKind:   casting.Spec.TelemetryKeeper.Kind.String(),
	}
}

// WithSuccess adds success=true to the properties.
func WithSuccess(props map[string]any) map[string]any {
	if props == nil {
		props = make(map[string]any)
	}
	props[PropSuccess] = true
	return props
}

// WithError adds success=false and the error message to the properties.
func WithError(props map[string]any, err error) map[string]any {
	if props == nil {
		props = make(map[string]any)
	}
	props[PropSuccess] = false
	props[PropError] = err.Error()
	return props
}
