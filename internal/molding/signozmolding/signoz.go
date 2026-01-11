package signozmolding

import (
	"context"
	"log/slog"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/molding"
)

var _ molding.Molding = (*signoz)(nil)

type signoz struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *signoz {
	return &signoz{
		logger: logger,
	}
}

func (molding *signoz) Kind() v1alpha1.MoldingKind {
	return v1alpha1.MoldingKindSignoz
}

func (molding *signoz) MoldV1Alpha1(ctx context.Context, config *v1alpha1.Casting) error {
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

// 	// Navigate to components.signoz.config in the CUE value
// 	signozConfig := config.LookupPath(cue.ParsePath("components.signoz.config"))
// 	if !signozConfig.Exists() {
// 		// Config is optional - generate minimal default
// 		files["config.yaml"] = []byte("{}\n")
// 		return files, nil
// 	}

// 	// Export CUE value to YAML
// 	configYAML, err := yaml.Encode(signozConfig)
// 	if err != nil {
// 		return nil, errors.New("failed to encode config: " + err.Error())
// 	}
// 	files["config.yaml"] = configYAML

// 	return files, nil
// }
