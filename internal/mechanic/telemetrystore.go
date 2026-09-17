package mechanic

import (
	"context"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/errors"
	"github.com/tidwall/gjson"
)

// signalTables maps a SigNoz query-builder signal to the ClickHouse table that
// backs it. Per-signal diagnostic queries (added later) run against these.
var signalTables = map[string]string{
	"traces":  "signoz_traces.distributed_signoz_index_v3",
	"logs":    "signoz_logs.distributed_logs_v2",
	"metrics": "signoz_metrics.distributed_samples_v4",
}

// signalPaths are the gjson paths to a rule's builder-query signals within the
// alert data payload. The compositeQuery sits at the root in some rule versions
// and under condition in others; both are tried.
var signalPaths = []string{
	"compositeQuery.queries.#.spec.signal",
	"condition.compositeQuery.queries.#.spec.signal",
}

// TableForSignal returns the ClickHouse table backing a query-builder signal.
func TableForSignal(signal string) (string, error) {
	table, ok := signalTables[signal]
	if !ok {
		return "", errors.Newf(errors.TypeUnsupported, "no clickhouse table mapping for signal %q", signal)
	}
	return table, nil
}

// SignalTables extracts the distinct signals an alert's builder queries target
// (from the data payload's compositeQuery) and maps each to its ClickHouse
// table, preserving first-seen order.
func SignalTables(data []byte) ([]string, error) {
	seen := make(map[string]struct{})
	var tables []string

	for _, path := range signalPaths {
		for _, result := range gjson.GetBytes(data, path).Array() {
			signal := result.String()
			if signal == "" {
				continue
			}

			table, err := TableForSignal(signal)
			if err != nil {
				return nil, err
			}

			if _, ok := seen[table]; ok {
				continue
			}
			seen[table] = struct{}{}
			tables = append(tables, table)
		}
	}

	return tables, nil
}

// TelemetryStore runs read-only queries against a deployment's telemetry store
// (ClickHouse).
type TelemetryStore interface {
	// Query runs sql against the telemetry store and returns its trimmed output.
	Query(ctx context.Context, sql string) (string, error)
}

// NewTelemetryStore selects the reach strategy for the deployment's telemetry
// store. Phase 1 supports docker/compose only, executing clickhouse-client
// inside the running ClickHouse container.
func NewTelemetryStore(executor Executor, machinery v1alpha1.Machinery) (TelemetryStore, error) {
	c, err := dockerComposeCasting(machinery)
	if err != nil {
		return nil, err
	}

	container, err := firstHost(c.Spec.TelemetryStore.Status.Addresses.TCP)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeNotFound, "telemetrystore address missing from lock, run forge first")
	}

	return &dockerClickhouseTelemetryStore{executor: executor, container: container}, nil
}

// dockerClickhouseTelemetryStore reaches ClickHouse by exec-ing clickhouse-client
// inside the running container. SigNoz provisions the default user with an empty
// password, so no credentials are passed.
type dockerClickhouseTelemetryStore struct {
	executor  Executor
	container string
}

func (t *dockerClickhouseTelemetryStore) Query(ctx context.Context, sql string) (string, error) {
	out, err := t.executor.Output(ctx, "docker", "exec", t.container,
		"clickhouse-client", "--query", sql,
	)
	if err != nil {
		return "", errors.Wrapf(err, errors.TypeInternal, "failed to query telemetrystore via container %q", t.container)
	}

	return strings.TrimSpace(string(out)), nil
}

// dockerComposeCasting asserts the machinery is an Installation deployed via
// docker/compose, the only target mechanic inspect reaches today.
func dockerComposeCasting(machinery v1alpha1.Machinery) (*installation.Casting, error) {
	c, ok := machinery.(*installation.Casting)
	if !ok {
		return nil, errors.Newf(errors.TypeUnsupported, "mechanic inspect supports the Installation kind only, got %q", machinery.Kind())
	}

	deployment := c.Spec.Deployment
	if deployment.Mode != v1alpha1.ModeDocker || deployment.Flavor != v1alpha1.FlavorCompose {
		return nil, errors.Newf(errors.TypeUnsupported, "mechanic inspect supports docker/compose only, got %s/%s", deployment.Mode, deployment.Flavor)
	}

	return c, nil
}
