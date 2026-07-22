package domain

import (
	"encoding/json"
	"strings"

	jsonpatchv5 "github.com/evanphx/json-patch/v5"
	"github.com/signoz/foundry/internal/errors"
	kyaml "sigs.k8s.io/yaml"
)

// ListType mirrors Kubernetes' listType marker: how a list merges when base
// and override both define one at the same path. Set and Ordered assume
// scalar elements and degrade to Atomic on a list of maps.
type ListType struct {
	name  string
	merge func(base, override []any) []any
}

func (l ListType) String() string { return l.name }

var (
	// ListTypeAtomic replaces the list wholesale: RFC 7386's rule and the
	// default for any path without a declared type.
	ListTypeAtomic = ListType{name: "atomic", merge: func(_, override []any) []any { return override }}

	// ListTypeSet unions scalar elements, base first, deduped.
	ListTypeSet = ListType{name: "set", merge: mergeSet}

	// ListTypeOrdered keeps base order and inserts override's new elements
	// before the terminal one; a foundry extension beyond Kubernetes' listType.
	ListTypeOrdered = ListType{name: "ordered", merge: mergeOrdered}
)

// ListTypes declares the list types of a document's paths: dotted keys with
// "*" matching any single segment, e.g. "service.pipelines.*.receivers".
// Undeclared paths are ListTypeAtomic.
type ListTypes map[string]ListType

// StrategicMergeYAML deep-merges the override YAML onto the base YAML the way
// Kubernetes' strategic merge patch works: the effective patch is computed
// first (lists at declared paths merge with the base's per their list type),
// then applied with plain RFC 7386 semantics (override wins at each leaf,
// base-only keys survive, a null in override deletes the key, undeclared
// lists are replaced). A nil listTypes applies the override as-is.
func StrategicMergeYAML(base, override string, listTypes ListTypes) (string, error) {
	var baseMap map[string]any
	if err := kyaml.Unmarshal([]byte(base), &baseMap); err != nil {
		return "", errors.Wrapf(err, errors.TypeInternal, "failed to unmarshal base yaml")
	}

	var patch map[string]any
	if err := kyaml.Unmarshal([]byte(override), &patch); err != nil {
		return "", errors.Wrapf(err, errors.TypeInvalidInput, "failed to unmarshal override yaml")
	}

	// An empty override is an empty merge patch: the document is unchanged.
	if len(patch) == 0 {
		return base, nil
	}

	resolvePatch(baseMap, patch, listTypes)

	baseJSON, err := json.Marshal(baseMap)
	if err != nil {
		return "", errors.Wrapf(err, errors.TypeInternal, "failed to marshal base json")
	}

	patchJSON, err := json.Marshal(patch)
	if err != nil {
		return "", errors.Wrapf(err, errors.TypeInternal, "failed to marshal merge patch")
	}

	mergedJSON, err := jsonpatchv5.MergePatch(baseJSON, patchJSON)
	if err != nil {
		return "", errors.Wrapf(err, errors.TypeInternal, "failed to apply json merge patch")
	}

	merged, err := kyaml.JSONToYAML(mergedJSON)
	if err != nil {
		return "", errors.Wrapf(err, errors.TypeInternal, "failed to convert merged json to yaml")
	}

	return string(merged), nil
}

// resolvePatch computes the effective merge patch in place: where the base and
// the patch both hold a list at a declared path, the patch gets the list
// merged per its declared type, so the RFC 7386 replace that follows lands the
// resolved list.
func resolvePatch(base, patch map[string]any, listTypes ListTypes) {
	for pattern, listType := range listTypes {
		segments := strings.Split(pattern, ".")
		leaf := len(segments) - 1

		pairs := []nodePair{{base: base, patch: patch}}
		for _, segment := range segments[:leaf] {
			next := make([]nodePair, 0, len(pairs))
			for _, pair := range pairs {
				next = append(next, pair.children(segment)...)
			}
			pairs = next
		}

		for _, pair := range pairs {
			pair.resolve(segments[leaf], listType)
		}
	}
}

// nodePair is the same node in the base and patch documents.
type nodePair struct {
	base  map[string]any
	patch map[string]any
}

// children returns the pairs one segment deeper; "*" spans every key held as a
// map by both documents.
func (p nodePair) children(segment string) []nodePair {
	var out []nodePair
	for key, patchVal := range p.patch {
		if segment != "*" && segment != key {
			continue
		}

		patchMap, ok := patchVal.(map[string]any)
		if !ok {
			continue
		}

		baseMap, ok := p.base[key].(map[string]any)
		if !ok {
			continue
		}

		out = append(out, nodePair{base: baseMap, patch: patchMap})
	}
	return out
}

// resolve merges the lists both documents hold at the segment into the patch.
func (p nodePair) resolve(segment string, listType ListType) {
	for key, patchVal := range p.patch {
		if segment != "*" && segment != key {
			continue
		}

		patchList, ok := patchVal.([]any)
		if !ok {
			continue
		}

		baseList, ok := p.base[key].([]any)
		if !ok {
			continue
		}

		p.patch[key] = listType.merge(baseList, patchList)
	}
}

// mergeSet unions scalar elements, base first, deduped.
func mergeSet(base, override []any) []any {
	if !scalarList(base) || !scalarList(override) {
		return override
	}
	return unionScalars(base, override)
}

// mergeOrdered keeps base's order, inserting override's new elements before the last.
func mergeOrdered(base, override []any) []any {
	if !scalarList(base) || !scalarList(override) {
		return override
	}
	if len(base) == 0 {
		return unionScalars(nil, override)
	}
	inBase := make(map[any]struct{}, len(base))
	for _, v := range base {
		inBase[v] = struct{}{}
	}
	extra := make([]any, 0, len(override))
	for _, v := range override {
		if _, ok := inBase[v]; !ok {
			extra = append(extra, v)
		}
	}
	out := make([]any, 0, len(base)+len(extra))
	out = append(out, base[:len(base)-1]...)
	out = append(out, extra...)
	out = append(out, base[len(base)-1])
	return unionScalars(out, nil)
}

// scalarList reports whether every element is a comparable scalar.
func scalarList(list []any) bool {
	for _, v := range list {
		switch v.(type) {
		case map[string]any, []any:
			return false
		}
	}
	return true
}

// unionScalars returns base then override's new elements, order-preserving and deduped.
func unionScalars(base, override []any) []any {
	out := make([]any, 0, len(base)+len(override))
	seen := make(map[any]struct{}, len(base)+len(override))
	for _, list := range [][]any{base, override} {
		for _, v := range list {
			if _, dup := seen[v]; dup {
				continue
			}
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}
