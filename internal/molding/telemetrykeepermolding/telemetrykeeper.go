package telemetrykeepermolding

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/molding"
)

var _ molding.Molding = (*telemetrykeeper)(nil)

type telemetrykeeper struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *telemetrykeeper {
	return &telemetrykeeper{
		logger: logger,
	}
}

func (molding *telemetrykeeper) Kind() v1alpha1.MoldingKind {
	return v1alpha1.MoldingKindTelemetryKeeper
}

func (molding *telemetrykeeper) MoldV1Alpha1(ctx context.Context, config *installation.Casting) error {
	switch config.Spec.TelemetryKeeper.Kind {
	case installation.TelemetryKeeperKindClickhouseKeeper:
		return molding.moldClickhouseKeeper(ctx, config)
	case installation.TelemetryKeeperKindZookeeper:
		return molding.moldZookeeper(ctx, config)
	default:
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported telemetrykeeper kind %q", config.Spec.TelemetryKeeper.Kind)
	}
}

func (molding *telemetrykeeper) moldClickhouseKeeper(ctx context.Context, config *installation.Casting) error {
	data, err := newData(config)
	if err != nil {
		molding.logger.ErrorContext(ctx, "failed to get data", foundryerrors.LogAttr(err))
		return err
	}

	// Extract enricher config overrides (applies to all keeper nodes).
	overrides := config.Spec.TelemetryKeeper.Status.Extras["_overrides"]

	// Generate per-server configs (each keeper node needs its own server_id)
	configs := make(map[string]string, data.ServerCount)
	for i := 0; i < data.ServerCount; i++ {
		configBuf := bytes.NewBuffer(nil)
		data.ServerID = i // 0-indexed, used for array indexing in template
		if err := KeeperClickhousev25125YAML.Execute(configBuf, data); err != nil {
			return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to execute keeper template for server %d", data.ServerID)
		}

		key := fmt.Sprintf("keeper-%d.yaml", i)
		base := configBuf.String()

		if overrides != "" {
			merged, err := domain.MergeYAML(base, overrides)
			if err != nil {
				return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to merge config overrides for %s", key)
			}
			base = merged
		}

		configs[key] = base
	}

	config.Spec.TelemetryKeeper.Status.Config.Data = configs
	return nil
}

func (molding *telemetrykeeper) moldZookeeper(_ context.Context, config *installation.Casting) error {
	if config.Spec.TelemetryKeeper.Status.Env == nil {
		config.Spec.TelemetryKeeper.Status.Env = make(map[string]string)
	}

	// The production zookeeper settings, matching the signoz helm chart and the
	// deprecated deploy/ compose (which set the first five and left the rest at
	// the same bitnami defaults). Castings translate these to their platform's
	// form; bitnami images consume them directly.
	env := config.Spec.TelemetryKeeper.Status.Env
	env["ALLOW_ANONYMOUS_LOGIN"] = "yes"
	env["ZOO_AUTOPURGE_INTERVAL"] = "1"
	env["ZOO_AUTOPURGE_RETAIN_COUNT"] = "3"
	env["ZOO_ENABLE_PROMETHEUS_METRICS"] = "yes"
	env["ZOO_PROMETHEUS_METRICS_PORT_NUMBER"] = "9141"
	env["ZOO_TICK_TIME"] = "2000"
	env["ZOO_INIT_LIMIT"] = "10"
	env["ZOO_SYNC_LIMIT"] = "5"
	env["ZOO_PRE_ALLOC_SIZE"] = "65536"
	env["ZOO_SNAPCOUNT"] = "100000"
	env["ZOO_MAX_CLIENT_CNXNS"] = "60"
	env["ZOO_MAX_SESSION_TIMEOUT"] = "40000"
	env["ZOO_4LW_COMMANDS_WHITELIST"] = "srvr, mntr, ruok"
	env["ZOO_HEAP_SIZE"] = "1024"
	env["ZOO_LOG_LEVEL"] = "INFO"

	return nil
}
