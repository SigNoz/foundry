package kuberneteskustomizecasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*/*.gotmpl templates/*/*/*.gotmpl templates/*.gotmpl
var templates embed.FS

var (
	// telemetrystore/clickhouse-operator.
	clickhouseOperatorClusterrole        = domain.MustNewTemplateFromFS(templates, "templates/clickhouse-operator/clusterrole.yaml.gotmpl", domain.FormatYAML)
	clickhouseOperatorClusterrolebinding = domain.MustNewTemplateFromFS(templates, "templates/clickhouse-operator/clusterrolebinding.yaml.gotmpl", domain.FormatYAML)
	clickhouseOperatorConfigmap          = domain.MustNewTemplateFromFS(templates, "templates/clickhouse-operator/configmap.yaml.gotmpl", domain.FormatYAML)
	clickhouseOperatorDeployment         = domain.MustNewTemplateFromFS(templates, "templates/clickhouse-operator/deployment.yaml.gotmpl", domain.FormatYAML)
	clickhouseOperatorService            = domain.MustNewTemplateFromFS(templates, "templates/clickhouse-operator/service.yaml.gotmpl", domain.FormatYAML)
	clickhouseOperatorServiceaccount     = domain.MustNewTemplateFromFS(templates, "templates/clickhouse-operator/serviceaccount.yaml.gotmpl", domain.FormatYAML)
	clickhouseOperatorKustomization      = domain.MustNewTemplateFromFS(templates, "templates/clickhouse-operator/kustomization.yaml.gotmpl", domain.FormatYAML)

	// telemetrystore/clickhouse.
	clickhouseInstanceInstallation      = domain.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse/clickhouseinstallation.yaml.gotmpl", domain.FormatYAML)
	clickhouseInstanceConfigmap         = domain.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse/configmap.yaml.gotmpl", domain.FormatYAML)
	clickhouseInstallationKustomization = domain.MustNewTemplateFromFS(templates, "templates/telemetrystore/clickhouse/kustomization.yaml.gotmpl", domain.FormatYAML)

	// telemetrykeeper.
	clickhouseKeeperInstallation  = domain.MustNewTemplateFromFS(templates, "templates/telemetrykeeper/clickhousekeeper/clickhousekeeperinstallation.yaml.gotmpl", domain.FormatYAML)
	clickhouseKeeperKustomization = domain.MustNewTemplateFromFS(templates, "templates/telemetrykeeper/clickhousekeeper/kustomization.yaml.gotmpl", domain.FormatYAML)

	// signoz.
	signozService        = domain.MustNewTemplateFromFS(templates, "templates/signoz/service.yaml.gotmpl", domain.FormatYAML)
	signozServiceaccount = domain.MustNewTemplateFromFS(templates, "templates/signoz/serviceaccount.yaml.gotmpl", domain.FormatYAML)
	signozStatefulset    = domain.MustNewTemplateFromFS(templates, "templates/signoz/statefulset.yaml.gotmpl", domain.FormatYAML)
	signozKustomization  = domain.MustNewTemplateFromFS(templates, "templates/signoz/kustomization.yaml.gotmpl", domain.FormatYAML)

	// ingester.
	ingesterConfigmap      = domain.MustNewTemplateFromFS(templates, "templates/ingester/configmap.yaml.gotmpl", domain.FormatYAML)
	ingesterDeployment     = domain.MustNewTemplateFromFS(templates, "templates/ingester/deployment.yaml.gotmpl", domain.FormatYAML)
	ingesterService        = domain.MustNewTemplateFromFS(templates, "templates/ingester/service.yaml.gotmpl", domain.FormatYAML)
	ingesterServiceaccount = domain.MustNewTemplateFromFS(templates, "templates/ingester/serviceaccount.yaml.gotmpl", domain.FormatYAML)
	ingesterKustomization  = domain.MustNewTemplateFromFS(templates, "templates/ingester/kustomization.yaml.gotmpl", domain.FormatYAML)

	// metastore/postgres.
	metastoreService        = domain.MustNewTemplateFromFS(templates, "templates/metastore/postgres/service.yaml.gotmpl", domain.FormatYAML)
	metastoreServiceaccount = domain.MustNewTemplateFromFS(templates, "templates/metastore/postgres/serviceaccount.yaml.gotmpl", domain.FormatYAML)
	metastoreStatefulset    = domain.MustNewTemplateFromFS(templates, "templates/metastore/postgres/statefulset.yaml.gotmpl", domain.FormatYAML)
	metastoreKustomization  = domain.MustNewTemplateFromFS(templates, "templates/metastore/postgres/kustomization.yaml.gotmpl", domain.FormatYAML)

	// telemetrystore-migrator.
	telemetrystoreMigratorJob           = domain.MustNewTemplateFromFS(templates, "templates/telemetrystore-migrator/job.yaml.gotmpl", domain.FormatYAML)
	telemetrystoreMigratorKustomization = domain.MustNewTemplateFromFS(templates, "templates/telemetrystore-migrator/kustomization.yaml.gotmpl", domain.FormatYAML)

	// deployment.
	deploymentNamespace     = domain.MustNewTemplateFromFS(templates, "templates/namespace.yaml.gotmpl", domain.FormatYAML)
	deploymentKustomization = domain.MustNewTemplateFromFS(templates, "templates/kustomization.yaml.gotmpl", domain.FormatYAML)

	// molding overrides.
	telemetryStoreOverrideTemplate = domain.MustNewTemplateFromFS(templates, "templates/overrides/telemetrystore.yaml.gotmpl", domain.FormatYAML)
)
