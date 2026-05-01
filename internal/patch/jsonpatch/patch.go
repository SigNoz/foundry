package jsonpatch

import (
	"context"
	"encoding/json"
	"fmt"

	jsonpatchv5 "github.com/evanphx/json-patch/v5"
	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/domain"
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
		return nil, fmt.Errorf("failed to marshal patch operations for target %q: %w", pe.Target, err)
	}

	result := make([]domain.Material, len(materials))
	copy(result, materials)

	matched := false
	for i, mat := range result {
		ok, err := patch.MatchTarget(pe.Target, mat.Path())
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", pe.Target, err)
		}
		if !ok {
			continue
		}

		if mat.IsMultiDoc() {
			return nil, fmt.Errorf("json patch on multi-doc yaml material %q is not supported", mat.Path())
		}

		matched = true
		patched, err := applyToMaterial(mat, patchDoc)
		if err != nil {
			return nil, fmt.Errorf("failed to apply patch to %q: %w", mat.Path(), err)
		}
		result[i] = patched
	}

	if !matched {
		return nil, fmt.Errorf("patch target %q did not match any generated material", pe.Target)
	}

	return result, nil
}

func applyToMaterial(mat domain.Material, patchDoc []byte) (domain.Material, error) {
	decoded, err := jsonpatchv5.DecodePatch(patchDoc)
	if err != nil {
		return domain.Material{}, fmt.Errorf("failed to decode json patch: %w", err)
	}

	patched, err := decoded.Apply(mat.Contents())
	if err != nil {
		return domain.Material{}, fmt.Errorf("failed to apply json patch: %w", err)
	}

	return mat.WithContents(patched), nil
}
