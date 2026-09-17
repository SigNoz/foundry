package foundry

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/mechanic"
	"github.com/signoz/foundry/internal/writer"
)

// entityKindAlert is the entity-kind segment that targets a SigNoz alert rule.
const entityKindAlert = "alert"

// probeQuery is the placeholder diagnostic mechanic runs against the telemetry
// store. The per-signal queries that decide whether an alert is a false alarm
// are defined in a later phase; for now this proves mechanic can reach and
// query ClickHouse directly.
const probeQuery = "SELECT version()"

// alertInspection is the result mechanic emits for an inspected alert: the rule
// metadata, the ClickHouse tables its signals map to, and the output of the
// diagnostic probe run against the telemetry store.
type alertInspection struct {
	Alert             mechanic.Alert `json:"alert"`
	Tables            []string       `json:"tables"`
	ClickhouseVersion string         `json:"clickhouseVersion"`
}

// MarshalJSON satisfies json.Marshaler so the inspection can stream to stdout
// via writer.WriteOutput. The alias breaks the method recursion.
func (a alertInspection) MarshalJSON() ([]byte, error) {
	type alias alertInspection
	return json.Marshal(alias(a))
}

// Inspect resolves a mechanic resource path against the loaded casting and runs the targeted inspection.
func (foundry *Foundry) Inspect(ctx context.Context, machinery v1alpha1.Machinery, resource mechanic.Resource, overrides mechanic.Overrides) error {
	resolved, err := resource.Resolve(machinery.Kind())
	if err != nil {
		return err
	}

	if resolved.EntityKind == "" || resolved.EntityID == "" {
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "inspect requires <molding> <entity-kind> <entity-id>, got %q", resolved.String())
	}

	foundry.Logger.InfoContext(
		ctx, "mechanic inspect target resolved",
		slog.String("kind", resolved.Kind.String()),
		slog.String("molding", resolved.Molding.String()),
		slog.String("entity.kind", resolved.EntityKind),
		slog.String("entity.id", resolved.EntityID),
	)

	conn := mechanic.ResolveConnection(machinery, overrides)
	foundry.Logger.InfoContext(
		ctx, "mechanic connection resolved",
		slog.String("clickhouse", conn.Clickhouse.Value),
		slog.String("clickhouse.source", string(conn.Clickhouse.Source)),
		slog.String("metastore", conn.Metastore.Value),
		slog.String("metastore.source", string(conn.Metastore.Source)),
		slog.String("signoz", conn.Signoz.Value),
		slog.String("signoz.source", string(conn.Signoz.Source)),
	)

	// The current reachers exec into the running containers named in the lock,
	// so connection overrides do not change where mechanic connects. They are
	// reserved for the direct-driver path; warn rather than silently ignore them.
	if conn.UsesOverride() {
		foundry.Logger.WarnContext(ctx, "connection overrides are not applied in exec mode; reaching containers from the lock file")
	}

	switch resolved.EntityKind {
	case entityKindAlert:
		return foundry.inspectAlert(ctx, machinery, resolved.EntityID)
	default:
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "inspect does not support entity kind %q yet", resolved.EntityKind)
	}
}

// inspectAlert looks the alert rule up in the deployment's metastore, maps the
// signals its queries target to ClickHouse tables, probes the telemetry store,
// and writes the inspection to stdout.
func (foundry *Foundry) inspectAlert(ctx context.Context, machinery v1alpha1.Machinery, id string) error {
	executor := mechanic.NewExecExecutor()

	metastore, err := mechanic.NewMetaStore(executor, machinery)
	if err != nil {
		return err
	}

	alert, err := metastore.Rule(ctx, id)
	if err != nil {
		return err
	}

	foundry.Logger.InfoContext(ctx, "alert metadata resolved",
		slog.String("alert.id", alert.ID),
		slog.String("alert.name", alert.Name),
	)

	// Map the alert's query signals to their backing ClickHouse tables. This is
	// best-effort context for the diagnostics to come; a rule whose signal we
	// can't map should not block reaching the telemetry store. Keep tables a
	// non-nil slice so the emitted JSON carries [] rather than null.
	tables := []string{}
	mapped, err := mechanic.SignalTables(alert.Data)
	if err != nil {
		foundry.Logger.WarnContext(ctx, "could not map alert signals to clickhouse tables", foundryerrors.LogAttr(err))
	} else {
		tables = append(tables, mapped...)
		foundry.Logger.InfoContext(ctx, "alert signals mapped to clickhouse tables",
			slog.String("tables", strings.Join(tables, ",")))
	}

	telemetrystore, err := mechanic.NewTelemetryStore(executor, machinery)
	if err != nil {
		return err
	}

	version, err := telemetrystore.Query(ctx, probeQuery)
	if err != nil {
		return err
	}

	foundry.Logger.InfoContext(ctx, "telemetrystore reached",
		slog.String("clickhouse.version", version))

	return writer.WriteOutput(os.Stdout, alertInspection{
		Alert:             alert,
		Tables:            tables,
		ClickhouseVersion: version,
	})
}
