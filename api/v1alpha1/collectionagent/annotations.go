package collectionagent

import "github.com/signoz/foundry/api/v1alpha1"

// Binary path annotation for the systemd/binary deployment of the
// CollectionAgent Kind. This is the single source for the key and default,
// consumed by the enricher, the unit template, and the binary check.
var (
	CollectorAgentBinaryPath = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/collector-binary-path",
		Default:     "/usr/local/bin/otelcol-contrib",
		Mode:        v1alpha1.ModeSystemd,
		Description: "Absolute path to the OpenTelemetry Collector Contrib binary.",
	}
)

// Annotations returns the CollectionAgent annotation catalog.
func Annotations() []v1alpha1.Annotation {
	return []v1alpha1.Annotation{
		CollectorAgentBinaryPath,
	}
}
