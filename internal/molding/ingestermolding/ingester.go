package ingestermolding

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
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

func (molding *ingester) MoldV1Alpha1(ctx context.Context, config *v1alpha1.Casting) error {
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

	return nil
}

func (molding *ingester) getData(config *v1alpha1.Casting) (Data, error) {
	if len(config.Spec.Signoz.Status.Addresses) == 0 {
		return Data{}, fmt.Errorf("signoz address is not set")
	}

	signozAddress := config.Spec.Signoz.Status.Addresses[v1alpha1.SignozAPIAddresses][0]

	if len(config.Spec.TelemetryStore.Status.Addresses) == 0 {
		return Data{}, fmt.Errorf("telemetry store address is not set")
	}

	telemetryStoreAddresses := config.Spec.TelemetryStore.Status.Addresses[v1alpha1.TelemetryStoreClusterAddresses]
	var telemetryStoreTracesAddresses []string
	for _, address := range telemetryStoreAddresses {
		telemetryStoreTracesAddresses = append(telemetryStoreTracesAddresses, address+"/signoz_traces")
	}

	var telemetryStoreMetricsAddresses []string
	for _, address := range telemetryStoreAddresses {
		telemetryStoreMetricsAddresses = append(telemetryStoreMetricsAddresses, address+"/signoz_metrics")
	}

	var telemetryStoreLogsAddresses []string
	for _, address := range telemetryStoreAddresses {
		telemetryStoreLogsAddresses = append(telemetryStoreLogsAddresses, address+"/signoz_logs")
	}

	var telemetryStoreMeterAddresses []string
	for _, address := range telemetryStoreAddresses {
		telemetryStoreMeterAddresses = append(telemetryStoreMeterAddresses, address+"/signoz_meter")
	}

	return Data{
		SignozOpampAddress:           signozAddress,
		TelemetryStoreTracesAddress:  strings.Join(telemetryStoreTracesAddresses, ","),
		TelemetryStoreMetricsAddress: strings.Join(telemetryStoreMetricsAddresses, ","),
		TelemetryStoreLogsAddress:    strings.Join(telemetryStoreLogsAddresses, ","),
		TelemetryStoreMeterAddress:   strings.Join(telemetryStoreMeterAddresses, ","),
	}, nil
}
