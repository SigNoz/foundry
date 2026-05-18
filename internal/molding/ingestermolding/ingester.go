package ingestermolding

import (
	"bytes"
	"context"
	"log/slog"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	foundryerrors "github.com/signoz/foundry/internal/errors"
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

func (molding *ingester) MoldV1Alpha1(ctx context.Context, config *installation.Casting) error {
	// render the template for config.yaml
	data, err := molding.getData(config)
	if err != nil {
		molding.logger.ErrorContext(ctx, "failed to get data", foundryerrors.LogAttr(err))
		return err
	}

	configBuf := bytes.NewBuffer(nil)
	if err := ConfigV0129xTemplate.Execute(configBuf, data); err != nil {
		return err
	}

	opampBuf := bytes.NewBuffer(nil)
	if err := OpampV0129xTemplate.Execute(opampBuf, data); err != nil {
		return err
	}

	config.Spec.Ingester.Status.Config.Data = map[string]string{
		"ingester.yaml": configBuf.String(),
		"opamp.yaml":    opampBuf.String(),
	}

	if config.Spec.Ingester.Status.Env == nil {
		config.Spec.Ingester.Status.Env = make(map[string]string)
	}
	config.Spec.Ingester.Status.Env["SIGNOZ_OTEL_COLLECTOR_TIMEOUT"] = "10m"

	return nil
}

func (molding *ingester) getData(config *installation.Casting) (Data, error) {
	if len(config.Spec.Signoz.Status.Addresses.Opamp) == 0 {
		return Data{}, foundryerrors.Newf(foundryerrors.TypeInternal, "signoz address is not set")
	}

	signozAddress := config.Spec.Signoz.Status.Addresses.Opamp[0]

	if len(config.Spec.TelemetryStore.Status.Addresses.TCP) == 0 {
		return Data{}, foundryerrors.Newf(foundryerrors.TypeInternal, "telemetry store address is not set")
	}

	telemetryStoreAddress := config.Spec.TelemetryStore.Status.Addresses.TCP[0]

	return Data{
		SignozOpampAddress:            signozAddress,
		TelemetryStoreTracesAddress:   telemetryStoreAddress + "/signoz_traces",
		TelemetryStoreMetricsAddress:  telemetryStoreAddress + "/signoz_metrics",
		TelemetryStoreLogsAddress:     telemetryStoreAddress + "/signoz_logs",
		TelemetryStoreMeterAddress:    telemetryStoreAddress + "/signoz_meter",
		TelemetryStoreMetadataAddress: telemetryStoreAddress + "/signoz_metadata",
	}, nil
}
