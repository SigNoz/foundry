package mechanic

import (
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
)

// Source records where an Endpoint's value came from so logs and downstream
// callers can tell an operator-supplied override apart from a lock-derived
// default.
type Source string

const (
	// SourceUnset means neither an override nor the lock file provided a value.
	SourceUnset Source = ""
	// SourceOverride means the value came from a flag or environment override.
	SourceOverride Source = "override"
	// SourceLock means the value was resolved from the casting lock file.
	SourceLock Source = "lock"
)

// Endpoint is a single resolved connection address paired with its origin.
type Endpoint struct {
	Value  string
	Source Source
}

// Connection holds the effective endpoints mechanic will dial. Each field is
// resolved by taking the override when set and otherwise falling back to the
// lock file's resolved status addresses.
type Connection struct {
	Signoz     Endpoint
	Clickhouse Endpoint
	Metastore  Endpoint
}

// UsesOverride reports whether any endpoint was sourced from an override rather
// than the lock file.
func (c Connection) UsesOverride() bool {
	return c.Signoz.Source == SourceOverride ||
		c.Clickhouse.Source == SourceOverride ||
		c.Metastore.Source == SourceOverride
}

// ResolveConnection derives the effective connection details from the lock file
// machinery, letting overrides win over the lock-derived addresses. Kinds that
// carry no telemetry/meta store (e.g. CollectionAgent) contribute no lock
// addresses; overrides still apply.
func ResolveConnection(machinery v1alpha1.Machinery, overrides Overrides) Connection {
	var signoz, clickhouse, metastore string

	if c, ok := machinery.(*installation.Casting); ok {
		signoz = strings.Join(c.Spec.Signoz.Status.Addresses.APIServer, ",")
		clickhouse = strings.Join(c.Spec.TelemetryStore.Status.Addresses.TCP, ",")
		metastore = strings.Join(c.Spec.MetaStore.Status.Addresses.DSN, ",")
	}

	return Connection{
		Signoz:     resolveEndpoint(overrides.Signoz, signoz),
		Clickhouse: resolveEndpoint(overrides.ClickhouseDSN, clickhouse),
		Metastore:  resolveEndpoint(overrides.MetastoreDSN, metastore),
	}
}

// resolveEndpoint applies override-wins precedence and tags the resulting value
// with its source.
func resolveEndpoint(override, fromLock string) Endpoint {
	if override != "" {
		return Endpoint{Value: override, Source: SourceOverride}
	}
	if fromLock != "" {
		return Endpoint{Value: fromLock, Source: SourceLock}
	}
	return Endpoint{Source: SourceUnset}
}
