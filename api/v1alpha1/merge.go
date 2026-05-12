package v1alpha1

import (
	"encoding/json"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// Merge applies an override onto a base via Kubernetes strategic merge patch
// and mutates base in place. Both arguments must be pointers to the same
// concrete struct type. Fields typed as any are replaced wholesale rather than
// recursively merged.
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

// MergeCasting layers an override Casting onto a base Casting and mutates
// base in place. APIVersion and Kind are overlaid when set on the override;
// Metadata, Spec, and Status are merged by handing the concrete pointer
// values to Merge, which dispatches on their reflected types.
func MergeCasting(base, override *Casting) error {
	if override.APIVersion != "" {
		base.APIVersion = override.APIVersion
	}
	if override.Kind != (Kind{}) {
		base.Kind = override.Kind
	}

	if err := Merge(&base.Metadata, &override.Metadata); err != nil {
		return fmt.Errorf("failed to merge casting met`adata: %w", err)
	}

	if override.Spec != nil {
		if err := Merge(base.Spec, override.Spec); err != nil {
			return fmt.Errorf("failed to merge casting spec: %w", err)
		}
	}

	if override.Status != nil {
		if err := Merge(base.Status, override.Status); err != nil {
			return fmt.Errorf("failed to merge casting status: %w", err)
		}
	}

	return nil
}

// MergeCastingSpecAndStatus folds each component's Status into its Spec in
// place. Dispatches on the concrete Spec kind; no-op for unknown kinds.
func MergeCastingSpecAndStatus(base *Casting) error {
	switch spec := base.Spec.(type) {
	case *SigNozCastingSpec:
		if err := spec.Signoz.Spec.MergeStatus(spec.Signoz.Status.MoldingStatus); err != nil {
			return err
		}
		if err := spec.TelemetryStore.Spec.MergeStatus(spec.TelemetryStore.Status.MoldingStatus); err != nil {
			return err
		}
		if err := spec.TelemetryKeeper.Spec.MergeStatus(spec.TelemetryKeeper.Status.MoldingStatus); err != nil {
			return err
		}
		if err := spec.MetaStore.Spec.MergeStatus(spec.MetaStore.Status.MoldingStatus); err != nil {
			return err
		}
		if err := spec.Ingester.Spec.MergeStatus(spec.Ingester.Status.MoldingStatus); err != nil {
			return err
		}
	}
	return nil
}
