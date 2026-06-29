package rendercasting

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl
var templates embed.FS

var (
	renderYAMLTemplate         *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/render.yaml.gotmpl", domain.FormatYAML)
	ingesterDockerfileTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/Dockerfile.ingester.gotmpl", domain.FormatText)

	// Version-stamped ClickHouse Dockerfiles, dispatched by the configured store/keeper version.
	telemetryKeeperDockerfilev2556     *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/Dockerfile.clickhousekeeper.telemetrykeeper.v2556.gotmpl", domain.FormatText)
	telemetryKeeperDockerfilev25125    *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/Dockerfile.clickhousekeeper.telemetrykeeper.v25125.gotmpl", domain.FormatText)
	telemetryKeeperDockerfileTemplates                  = domain.NewVersionedTemplates("25.12.5", map[string]*domain.Template{
		"25.5.6":  telemetryKeeperDockerfilev2556,
		"25.12.5": telemetryKeeperDockerfilev25125,
	})

	telemetryStoreDockerfilev2556     *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/Dockerfile.clickhouse.telemetrystore.v2556.gotmpl", domain.FormatText)
	telemetryStoreDockerfilev25125    *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/Dockerfile.clickhouse.telemetrystore.v25125.gotmpl", domain.FormatText)
	telemetryStoreDockerfileTemplates                  = domain.NewVersionedTemplates("25.12.5", map[string]*domain.Template{
		"25.5.6":  telemetryStoreDockerfilev2556,
		"25.12.5": telemetryStoreDockerfilev25125,
	})
)
