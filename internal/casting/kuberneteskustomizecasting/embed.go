package kuberneteskustomizecasting

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*/*.gotmpl templates/*/*/*.gotmpl templates/*.gotmpl
var templates embed.FS

var (
	// telemetrystore/clickhouse-operator.
	clickhouseOperatorClusterrole        = types.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse-operator/clusterrole.yaml.gotmpl", types.FormatYAML)
	clickhouseOperatorClusterrolebinding = types.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse-operator/clusterrolebinding.yaml.gotmpl", types.FormatYAML)
	clickhouseOperatorConfigmap          = types.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse-operator/configmap.yaml.gotmpl", types.FormatYAML)
	clickhouseOperatorDeployment         = types.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse-operator/deployment.yaml.gotmpl", types.FormatYAML)
	clickhouseOperatorService            = types.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse-operator/service.yaml.gotmpl", types.FormatYAML)
	clickhouseOperatorServiceaccount     = types.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse-operator/serviceaccount.yaml.gotmpl", types.FormatYAML)
	clickhouseOperatorKustomization      = types.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse-operator/kustomization.yaml.gotmpl", types.FormatYAML)

	// telemetrystore/clickhouse-instance.
	clickhouseInstanceInstallation      = types.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse-instance/clickhouseinstallation.yaml.gotmpl", types.FormatYAML)
	clickhouseInstanceConfigmap         = types.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse-instance/configmap.yaml.gotmpl", types.FormatYAML)
	clickhouseInstallationKustomization = types.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse-instance/kustomization.yaml.gotmpl", types.FormatYAML)

	// telemetrykeeper.
	clickhouseKeeperInstallation  = types.MustNewTemplateFromFS(templates, "templates/telemetrykeeper/clickhousekeeperinstallation.yaml.gotmpl", types.FormatYAML)
	clickhouseKeeperKustomization = types.MustNewTemplateFromFS(templates, "templates/telemetrykeeper/kustomization.yaml.gotmpl", types.FormatYAML)

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

	// metastore/postgresql.
	metastoreService        = types.MustNewTemplateFromFS(templates, "templates/metastore/postgresql/service.yaml.gotmpl", types.FormatYAML)
	metastoreServiceaccount = types.MustNewTemplateFromFS(templates, "templates/metastore/postgresql/serviceaccount.yaml.gotmpl", types.FormatYAML)
	metastoreStatefulset    = types.MustNewTemplateFromFS(templates, "templates/metastore/postgresql/statefulset.yaml.gotmpl", types.FormatYAML)
	metastoreKustomization  = types.MustNewTemplateFromFS(templates, "templates/metastore/postgresql/kustomization.yaml.gotmpl", types.FormatYAML)

	// telemetrystore-migrator.
	telemetrystoreMigratorJob           = types.MustNewTemplateFromFS(templates, "templates/telemetrystore-migrator/job.yaml.gotmpl", types.FormatYAML)
	telemetrystoreMigratorKustomization = types.MustNewTemplateFromFS(templates, "templates/telemetrystore-migrator/kustomization.yaml.gotmpl", types.FormatYAML)

	// deployment.
	deploymentKustomization = types.MustNewTemplateFromFS(templates, "templates/kustomization.yaml.gotmpl", types.FormatYAML)
)
