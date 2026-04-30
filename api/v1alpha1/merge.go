package v1alpha1

import (
	"encoding/json"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

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

func MergeCastingSpecAndStatus(base *Casting) error {
	if err := base.Spec.Signoz.Spec.MergeStatus(base.Spec.Signoz.Status.MoldingStatus); err != nil {
		return err
	}

	if err := base.Spec.TelemetryStore.Spec.MergeStatus(base.Spec.TelemetryStore.Status.MoldingStatus); err != nil {
		return err
	}

	if err := base.Spec.TelemetryKeeper.Spec.MergeStatus(base.Spec.TelemetryKeeper.Status.MoldingStatus); err != nil {
		return err
	}

	if err := base.Spec.MetaStore.Spec.MergeStatus(base.Spec.MetaStore.Status.MoldingStatus); err != nil {
		return err
	}

	if err := base.Spec.Ingester.Spec.MergeStatus(base.Spec.Ingester.Status.MoldingStatus); err != nil {
		return err
	}

	return nil
}
