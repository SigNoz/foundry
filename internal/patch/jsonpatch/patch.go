package jsonpatch

import (
	"context"
	"encoding/json"

	jsonpatchv5 "github.com/evanphx/json-patch/v5"
	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
	"github.com/signoz/foundry/internal/patch"
)

var _ patch.Patch = (*jsonPatch)(nil)

type jsonPatch struct{}

func New() patch.Patch {
	return &jsonPatch{}
}

func (p *jsonPatch) Apply(ctx context.Context, materials []domain.Material, pe v1alpha1.PatchEntry) ([]domain.Material, error) {
	patchDoc, err := json.Marshal(pe.Operations)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to marshal patch operations for target %q", pe.Target)
	}

	result := make([]domain.Material, len(materials))
	copy(result, materials)

	matched := false
	for i, mat := range result {
		ok, err := patch.MatchTarget(pe.Target, mat.Path())
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInvalidInput, "invalid glob pattern %q", pe.Target)
		}
		if !ok {
			continue
		}

		structured, ok := mat.(domain.StructuredMaterial)
		if !ok {
			return nil, errors.Newf(errors.TypeUnsupported, "json patch on blob material %q is not supported", mat.Path())
		}

		if structured.HasMultipleDocuments() {
			return nil, errors.Newf(errors.TypeUnsupported, "json patch on multi-doc yaml material %q is not supported", mat.Path())
		}

		matched = true
		patched, err := applyToMaterial(structured, patchDoc)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeInternal, "failed to apply patch to %q", mat.Path())
		}
		result[i] = patched
	}

	if !matched {
		return nil, errors.Newf(errors.TypeNotFound, "patch target %q did not match any generated material", pe.Target)
	}

	return result, nil
}

func applyToMaterial(mat domain.StructuredMaterial, patchDoc []byte) (domain.StructuredMaterial, error) {
	decoded, err := jsonpatchv5.DecodePatch(patchDoc)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to decode json patch")
	}

	patched, err := decoded.Apply(mat.JSONContents())
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to apply json patch")
	}

	return mat.CloneWithJSONContents(patched), nil
}
