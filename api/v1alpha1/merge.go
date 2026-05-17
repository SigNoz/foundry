package v1alpha1

import (
	"encoding/json"
	"reflect"

	"github.com/signoz/foundry/internal/errors"
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
		return errors.Wrapf(err, errors.TypeInternal, "failed to convert current object to byte sequence")
	}

	overrideBytes, err := json.Marshal(overrides)
	if err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to convert current object to byte sequence")
	}

	patchMeta, err := strategicpatch.NewPatchMetaFromStruct(base)
	if err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to produce patch meta from struct")
	}

	patch, err := strategicpatch.CreateThreeWayMergePatch(overrideBytes, overrideBytes, baseBytes, patchMeta, true)
	if err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to create three way merge patch")
	}

	merged, err := strategicpatch.StrategicMergePatchUsingLookupPatchMeta(baseBytes, patch, patchMeta)
	if err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to apply patch")
	}

	valueOfBase := reflect.Indirect(reflect.ValueOf(base))

	into := reflect.New(valueOfBase.Type())
	if err := json.Unmarshal(merged, into.Interface()); err != nil {
		return err
	}

	if !valueOfBase.CanSet() {
		return errors.Newf(errors.TypeInternal, "unable to set unmarshalled value into base object")
	}

	valueOfBase.Set(reflect.Indirect(into))

	return nil
}
