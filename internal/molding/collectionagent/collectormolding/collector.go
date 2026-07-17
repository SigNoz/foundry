package collectormolding

import (
	"bytes"
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	"github.com/signoz/foundry/internal/domain"
	foundryerrors "github.com/signoz/foundry/internal/errors"
	collectionagentmolding "github.com/signoz/foundry/internal/molding/collectionagent"
)

var _ collectionagentmolding.Molding = (*collector)(nil)

type collector struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *collector {
	return &collector{logger: logger}
}

func (m *collector) Kind() v1alpha1.MoldingKind {
	return v1alpha1.MoldingKindCollector
}

// listTypes declares how each pipeline list merges with the enricher's
// contribution: receivers/exporters/extensions union (order-insensitive);
// processors keep base order with contributed ones before the terminal batch.
var listTypes = domain.ListTypes{
	"service.pipelines.*.receivers":  domain.ListTypeSet,
	"service.pipelines.*.processors": domain.ListTypeOrdered,
	"service.pipelines.*.exporters":  domain.ListTypeSet,
	"service.extensions":             domain.ListTypeSet,
}

// MoldV1Alpha1 renders the collector's base config for the kind, merges the
// enricher's contribution onto it (see listTypes), and stores the result
// at the kind's config key.
func (m *collector) MoldV1Alpha1(ctx context.Context, config *collectionagent.Casting) error {
	kind := config.Spec.Collector.Kind
	spec := &config.Spec.Collector.Spec
	status := &config.Spec.Collector.Status

	var tmpl *domain.Template
	switch kind {
	case collectionagent.CollectorKindAgent:
		tmpl = agentConfig
	default:
		return foundryerrors.Newf(foundryerrors.TypeUnsupported, "unsupported collector kind %q", kind)
	}

	if spec.Env["SIGNOZ_INGESTION_ENDPOINT"] == "" {
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "collector molding requires SIGNOZ_INGESTION_ENDPOINT in spec.collector.spec.env")
	}

	base := bytes.NewBuffer(nil)
	if err := tmpl.Execute(base, struct{ IngestionKey bool }{IngestionKey: spec.Env["SIGNOZ_INGESTION_KEY"] != ""}); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to render base collector config")
	}

	key := kind.ConfigKey()
	final, err := domain.StrategicMergeYAML(base.String(), status.Config.Data[key], listTypes)
	if err != nil {
		return err
	}

	status.Config.Set(key, []byte(final))

	if status.Env == nil {
		status.Env = map[string]string{}
	}
	status.Env["OTEL_COLLECTOR_ROLE"] = kind.String()
	return nil
}
