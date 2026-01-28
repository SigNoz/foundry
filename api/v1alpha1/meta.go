package v1alpha1

import (
	"encoding/json"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

type TypeVersion struct {
	// API Version of the casting configuration schema.
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
}

type TypeMetadata struct {
	// The name of this installation. This name can be used to identify the installation.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// Annotations is an unstructured key-value map for arbitrary metadata.
	// Can be used to specify deployment-specific settings.
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

type TypeCluster struct {
	// Number of replicas for the component
	Replicas *int `json:"replicas,omitempty" yaml:"replicas,omitempty"`

	// Number of shards for the component
	Shards *int `json:"shards,omitempty" yaml:"shards,omitempty"`
}

type TypeConfig struct {
	// Data contains the configuration data.
	Data map[string]string `json:"data,omitempty" yaml:"data,omitempty"`
}

type TypeDeployment struct {
	// Platform: Provider where an installation runs on using various cloud vendors
	// Example values: aws|gcp|azure|digitalocean|railway
	Platform string `json:"platform,omitempty" yaml:"platform,omitempty"`

	// Mode: Type of installation method that we support, currently identifies the engine or technology behind a deployment
	// Example values: binary|docker|kubernetes|helm|nomad|windows|systemctl
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty"`

	// Flavor: Defines the flavor of mode for the deployment, allows the user the pattern to deploy on
	// Example values: compose|swarm|helmfile|helm|kustomize|binary|rpm|deb|chocolatey
	Flavor string `json:"flavor,omitempty" yaml:"flavor,omitempty"`
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
