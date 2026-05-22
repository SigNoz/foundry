package collectormolding

import (
	"bytes"
	"path/filepath"
	"sort"

	"github.com/signoz/foundry/api/v1alpha1/collectionagent"
	foundryerrors "github.com/signoz/foundry/internal/errors"
)

// AgentConfig is one rendered configuration the agent collector molding
// produces. A molding may produce multiple AgentConfigs to support
// multi-instance deployments (e.g., per shard or replica). The casting
// reads these via AgentConfigsOf and stages them into its pourer.
type AgentConfig struct {
	// Path is the relative path within the casting's pour area
	// (e.g. "otel-collector-config.yaml").
	Path string

	// Content is the rendered configuration bytes.
	Content []byte
}

// AgentConfigsOf reads the configurations the agent molding stamped on the
// casting's CollectorStatus. Empty slice means the molding didn't run or
// didn't stamp. Results are sorted by Path for deterministic ordering.
func AgentConfigsOf(c *collectionagent.Casting) []AgentConfig {
	data := c.Spec.Collector.Spec.Config.Data
	paths := make([]string, 0, len(data))
	for p := range data {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	configs := make([]AgentConfig, 0, len(paths))
	for _, p := range paths {
		configs = append(configs, AgentConfig{Path: p, Content: []byte(data[p])})
	}
	return configs
}

// moldAgent renders the agent OTel collector config and stamps it on the
// casting's CollectorStatus.
func (m *collector) moldAgent(config *collectionagent.Casting) error {
	spec := &config.Spec.Collector.Spec
	status := &config.Spec.Collector.Status

	if spec.Env["SIGNOZ_INGESTION_ENDPOINT"] == "" {
		return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "collector molding requires SIGNOZ_INGESTION_ENDPOINT in spec.collector.spec.env")
	}

	if err := validateAgentReferences(status); err != nil {
		return err
	}

	data := buildAgentTemplateData(status)
	data.IngestionKey = spec.Env["SIGNOZ_INGESTION_KEY"] != ""

	buf := bytes.NewBuffer(nil)
	if err := AgentYAMLTemplate.Execute(buf, data); err != nil {
		return foundryerrors.Wrapf(err, foundryerrors.TypeInternal, "failed to render agent collector config")
	}

	if status.Config.Data == nil {
		status.Config.Data = map[string]string{}
	}
	role := collectionagent.CollectorKindAgent.String()
	status.Config.Data[filepath.Join(m.Kind().String(), role, role+".yaml")] = buf.String()

	if status.Env == nil {
		status.Env = map[string]string{}
	}
	status.Env["OTEL_COLLECTOR_ROLE"] = role
	return nil
}

// agentTemplateData is the per-pipeline view the agent template consumes.
// buildAgentTemplateData inverts the colocated CollectorStatus.Receivers /
// .Processors / .Exporters maps (each entry carries Component.Pipelines) into
// the pipeline-keyed lists the template renders directly.
type agentTemplateData struct {
	Receivers          map[string]map[string]any
	Processors         map[string]map[string]any
	Exporters          map[string]map[string]any
	Extensions         map[string]map[string]any
	ResourceDetectors  []string
	PipelineReceivers  map[string][]string
	PipelineProcessors map[string][]string
	PipelineExporters  map[string][]string
	ServiceExtensions  []string
	// IngestionKey is true when spec.collector.spec.env carries a non-empty
	// SIGNOZ_INGESTION_KEY (cloud-only); the template emits the otlphttp
	// header block only when set.
	IngestionKey bool
}

func buildAgentTemplateData(s *collectionagent.CollectorStatus) agentTemplateData {
	d := agentTemplateData{
		Receivers:          map[string]map[string]any{},
		Processors:         map[string]map[string]any{},
		Exporters:          map[string]map[string]any{},
		Extensions:         map[string]map[string]any{},
		ResourceDetectors:  s.ResourceDetectors,
		PipelineReceivers:  map[string][]string{},
		PipelineProcessors: map[string][]string{},
		PipelineExporters:  map[string][]string{},
	}
	for name, c := range s.Receivers {
		d.Receivers[name] = c.Body
		for _, p := range c.Pipelines {
			d.PipelineReceivers[p] = append(d.PipelineReceivers[p], name)
		}
	}
	for name, c := range s.Processors {
		d.Processors[name] = c.Body
		for _, p := range c.Pipelines {
			d.PipelineProcessors[p] = append(d.PipelineProcessors[p], name)
		}
	}
	for name, c := range s.Exporters {
		d.Exporters[name] = c.Body
		for _, p := range c.Pipelines {
			d.PipelineExporters[p] = append(d.PipelineExporters[p], name)
		}
	}
	for name, body := range s.Extensions {
		d.Extensions[name] = body
		d.ServiceExtensions = append(d.ServiceExtensions, name)
	}
	sort.Strings(d.ServiceExtensions)
	for _, names := range d.PipelineReceivers {
		sort.Strings(names)
	}
	for _, names := range d.PipelineProcessors {
		sort.Strings(names)
	}
	for _, names := range d.PipelineExporters {
		sort.Strings(names)
	}
	return d
}

// validateAgentReferences checks that each Component's Pipelines entry names
// one of the three pipelines the mold renders. With wiring colocated on the
// Component, cross-map dangling references can no longer occur.
func validateAgentReferences(s *collectionagent.CollectorStatus) error {
	valid := map[string]bool{"traces": true, "metrics": true, "logs": true}
	check := func(kind, name string, pipelines []string) error {
		for _, p := range pipelines {
			if !valid[p] {
				return foundryerrors.Newf(foundryerrors.TypeInvalidInput, "%s %q references unknown pipeline %q", kind, name, p)
			}
		}
		return nil
	}
	for n, c := range s.Receivers {
		if err := check("receiver", n, c.Pipelines); err != nil {
			return err
		}
	}
	for n, c := range s.Processors {
		if err := check("processor", n, c.Pipelines); err != nil {
			return err
		}
	}
	for n, c := range s.Exporters {
		if err := check("exporter", n, c.Pipelines); err != nil {
			return err
		}
	}
	return nil
}
