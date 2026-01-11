package ingestermolding

import (
	"context"
	"log/slog"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
)

var _ molding.Molding = (*ingester)(nil)

type ingester struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *ingester {
	return &ingester{
		logger: logger,
	}
}

func (molding *ingester) Kind() v1alpha1.MoldingKind {
	return v1alpha1.MoldingKindIngester
}

func (molding *ingester) MoldV1Alpha1(ctx context.Context, config *v1alpha1.Casting) error {
	if config.Spec.Signoz.Spec.Env == nil {
		config.Spec.Signoz.Spec.Env = make(map[string]string)
	}

	if config.Spec.Signoz.Status.Env == nil {
		config.Spec.Signoz.Status.Env = make(map[string]string)
	}

	// Add telemetry store addresses
	if val, ok := config.Spec.Signoz.Spec.Env["SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN"]; ok {
		molding.logger.WarnContext(ctx, "SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN is set and is going to be ignored", slog.String("value", val))
	}

	config.Spec.Signoz.Status.Env["SIGNOZ_TELEMETRYSTORE_PROVIDER"] = config.Spec.TelemetryStore.Kind.String()
	config.Spec.Signoz.Status.Env["SIGNOZ_TELEMETRYSTORE_CLICKHOUSE_DSN"] = strings.Join(config.Spec.TelemetryStore.Status.Addresses, ",")

	return nil
}

// type Generator struct{}

// func (g *Generator) GenerateComponent(config cue.Value) (map[string][]byte, error) {
// 	files := make(map[string][]byte)

// 	// Navigate to components.signozOtelCollector.config in the CUE value
// 	collectorConfig := config.LookupPath(cue.ParsePath("components.signozOtelCollector.config"))

// 	if collectorConfig.Exists() {
// 		// Export CUE value to YAML
// 		configYAML, err := yaml.Encode(collectorConfig)
// 		if err != nil {
// 			return nil, errors.New("failed to encode config: " + err.Error())
// 		}
// 		files["config.yaml"] = configYAML
// 	} else {
// 		return nil, errors.New("signozOtelCollector config not found in the provided CUE value")
// 	}

// 	return files, nil
// }
