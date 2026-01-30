package v1alpha1

import (
	"encoding/json"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

type TypeVersion struct {
	// API Version of the casting configuration schema.
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty" description:"API Version of the casting configuration schema" example:"v1alpha1"`
}

type TypeMetadata struct {
	// The name of this installation. This name can be used to identify the installation.
	Name string `json:"name,omitempty" yaml:"name,omitempty" description:"The name of this installation" example:"signoz-dev"`

	// Annotations is an unstructured key-value map for arbitrary metadata.
	// Can be used to specify deployment-specific settings.
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty" description:"Unstructured key-value map for arbitrary metadata"`
}

type TypeCluster struct {
	// Number of replicas for the component
	Replicas *int `json:"replicas,omitempty" yaml:"replicas,omitempty" description:"Number of replicas for the component" example:"1"`

	// Number of shards for the component
	Shards *int `json:"shards,omitempty" yaml:"shards,omitempty" description:"Number of shards for the component" example:"1"`
}

type TypeConfig struct {
	// Data contains the configuration data.
	Data map[string]string `json:"data,omitempty" yaml:"data,omitempty" description:"Configuration data as key-value pairs"`
}

type TypeDeployment struct {
	// Platform: Provider where an installation runs on using various cloud vendors
	// Example values: aws|gcp|azure|digitalocean|railway
	Platform string `json:"platform,omitempty" yaml:"platform,omitempty" description:"Provider where an installation runs on" examples:"[\"aws\",\"gcp\",\"azure\",\"digitalocean\",\"railway\",\"docker\",\"linux\"]"`

	// Mode: Type of installation method that we support, currently identifies the engine or technology behind a deployment
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty" description:"Type of installation method" examples:"[\"binary\",\"docker\",\"kubernetes\",\"helm\",\"nomad\",\"windows\",\"systemctl\"]"`

	// Flavor: Defines the flavor of mode for the deployment, allows the user the pattern to deploy on
	Flavor string `json:"flavor,omitempty" yaml:"flavor,omitempty" description:"Flavor of mode for the deployment" examples:"[\"compose\",\"swarm\",\"helmfile\",\"helm\",\"kustomize\",\"binary\",\"rpm\",\"deb\",\"chocolatey\"]"`
}

func Merge(base, overrides any) error {
	if overrides == nil {
		return nil
	}

	baseBytes, err := json.Marshal(base)
	if err != nil {
		return fmt.Errorf("failed to convert current object to byte sequence: %w", err)
	}

	overrideBytes, err := json.Marshal(overrides)
	if err != nil {
		return fmt.Errorf("failed to convert current object to byte sequence: %w", err)
	}

	patchMeta, err := strategicpatch.NewPatchMetaFromStruct(base)
	if err != nil {
		return fmt.Errorf("failed to produce patch meta from struct: %w", err)
	}

	patch, err := strategicpatch.CreateThreeWayMergePatch(overrideBytes, overrideBytes, baseBytes, patchMeta, true)
	if err != nil {
		return fmt.Errorf("failed to create three way merge patch: %w", err)
	}

	merged, err := strategicpatch.StrategicMergePatchUsingLookupPatchMeta(baseBytes, patch, patchMeta)
	if err != nil {
		return fmt.Errorf("failed to apply patch: %w", err)
	}

	valueOfBase := reflect.Indirect(reflect.ValueOf(base))

	into := reflect.New(valueOfBase.Type())
	if err := json.Unmarshal(merged, into.Interface()); err != nil {
		return err
	}

	if !valueOfBase.CanSet() {
		return fmt.Errorf("unable to set unmarshalled value into base object")
	}

	valueOfBase.Set(reflect.Indirect(into))

	return nil
}
