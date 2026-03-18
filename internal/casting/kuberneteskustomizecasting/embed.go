package kuberneteskustomizecasting

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*/*.gotmpl templates/*/*/*.gotmpl templates/*.gotmpl
var templates embed.FS

var (
	// telemetrystore/clickhouse-operator.
	clickhouseOperatorClusterrole        = types.MustNewTemplateFromFS(templates, "templates/clickhouse-operator/clusterrole.yaml.gotmpl", types.FormatYAML)
	clickhouseOperatorClusterrolebinding = types.MustNewTemplateFromFS(templates, "templates/clickhouse-operator/clusterrolebinding.yaml.gotmpl", types.FormatYAML)
	clickhouseOperatorConfigmap          = types.MustNewTemplateFromFS(templates, "templates/clickhouse-operator/configmap.yaml.gotmpl", types.FormatYAML)
	clickhouseOperatorDeployment         = types.MustNewTemplateFromFS(templates, "templates/clickhouse-operator/deployment.yaml.gotmpl", types.FormatYAML)
	clickhouseOperatorService            = types.MustNewTemplateFromFS(templates, "templates/clickhouse-operator/service.yaml.gotmpl", types.FormatYAML)
	clickhouseOperatorServiceaccount     = types.MustNewTemplateFromFS(templates, "templates/clickhouse-operator/serviceaccount.yaml.gotmpl", types.FormatYAML)
	clickhouseOperatorKustomization      = types.MustNewTemplateFromFS(templates, "templates/clickhouse-operator/kustomization.yaml.gotmpl", types.FormatYAML)

	// telemetrystore/clickhouse.
	clickhouseInstanceInstallation      = types.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse/clickhouseinstallation.yaml.gotmpl", types.FormatYAML)
	clickhouseInstanceConfigmap         = types.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse/configmap.yaml.gotmpl", types.FormatYAML)
	clickhouseInstallationKustomization = types.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse/kustomization.yaml.gotmpl", types.FormatYAML)

	// telemetrykeeper.
	clickhouseKeeperInstallation  = types.MustNewTemplateFromFS(templates, "templates/telemetrykeeper/clickhousekeeper/clickhousekeeperinstallation.yaml.gotmpl", types.FormatYAML)
	clickhouseKeeperKustomization = types.MustNewTemplateFromFS(templates, "templates/telemetrykeeper/clickhousekeeper/kustomization.yaml.gotmpl", types.FormatYAML)

	// signoz.
	signozService        = types.MustNewTemplateFromFS(templates, "templates/signoz/service.yaml.gotmpl", types.FormatYAML)
	signozServiceaccount = types.MustNewTemplateFromFS(templates, "templates/signoz/serviceaccount.yaml.gotmpl", types.FormatYAML)
	signozStatefulset    = types.MustNewTemplateFromFS(templates, "templates/signoz/statefulset.yaml.gotmpl", types.FormatYAML)
	signozKustomization  = types.MustNewTemplateFromFS(templates, "templates/signoz/kustomization.yaml.gotmpl", types.FormatYAML)

	// ingester.
	ingesterConfigmap      = types.MustNewTemplateFromFS(templates, "templates/ingester/configmap.yaml.gotmpl", types.FormatYAML)
	ingesterDeployment     = types.MustNewTemplateFromFS(templates, "templates/ingester/deployment.yaml.gotmpl", types.FormatYAML)
	ingesterService        = types.MustNewTemplateFromFS(templates, "templates/ingester/service.yaml.gotmpl", types.FormatYAML)
	ingesterServiceaccount = types.MustNewTemplateFromFS(templates, "templates/ingester/serviceaccount.yaml.gotmpl", types.FormatYAML)
	ingesterKustomization  = types.MustNewTemplateFromFS(templates, "templates/ingester/kustomization.yaml.gotmpl", types.FormatYAML)

	// metastore/postgres.
	metastoreService        = types.MustNewTemplateFromFS(templates, "templates/metastore/postgres/service.yaml.gotmpl", types.FormatYAML)
	metastoreServiceaccount = types.MustNewTemplateFromFS(templates, "templates/metastore/postgres/serviceaccount.yaml.gotmpl", types.FormatYAML)
	metastoreStatefulset    = types.MustNewTemplateFromFS(templates, "templates/metastore/postgres/statefulset.yaml.gotmpl", types.FormatYAML)
	metastoreKustomization  = types.MustNewTemplateFromFS(templates, "templates/metastore/postgres/kustomization.yaml.gotmpl", types.FormatYAML)

	// telemetrystore-migrator.
	telemetrystoreMigratorJob           = types.MustNewTemplateFromFS(templates, "templates/telemetrystore-migrator/job.yaml.gotmpl", types.FormatYAML)
	telemetrystoreMigratorKustomization = types.MustNewTemplateFromFS(templates, "templates/telemetrystore-migrator/kustomization.yaml.gotmpl", types.FormatYAML)

	// deployment.
	deploymentNamespace     = types.MustNewTemplateFromFS(templates, "templates/namespace.yaml.gotmpl", types.FormatYAML)
	deploymentKustomization = types.MustNewTemplateFromFS(templates, "templates/kustomization.yaml.gotmpl", types.FormatYAML)

	// molding overrides.
	telemetryStoreOverrideTemplate = types.MustNewTemplateFromFS(templates, "templates/overrides/telemetrystore.yaml.gotmpl", types.FormatYAML)
)
