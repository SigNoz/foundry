package domain

import "strings"

// ListMerge describes how a list is merged when both the base and the override
// define one at the same path. It mirrors Kubernetes' listType
// (structured-merge-diff's ElementRelationship). The variant carries its own
// merge function, so adding one is a single var entry, not a switch arm.
//
//   - ListMergeReplace ("atomic"): the override's list wins wholesale. The
//     default for any list whose path matches no strategy.
//   - ListMergeSet ("set"): scalar elements are unioned, base first, deduped.
//   - ListMergeOrdered ("ordered"): the base order is kept and the override's
//     extra elements are inserted before the base's terminal element, for lists
//     like an OTel pipeline's processors where order matters but the base
//     members must survive.
//
// ListMergeSet and ListMergeOrdered assume scalar (comparable) elements; a list
// of maps under either degrades to ListMergeReplace rather than panicking.
// Strategy is supplied per path (see Merge), so Foundry never models the
// upstream component's schema.
type ListMerge struct {
	name  string
	merge func(base, override []any) []any
}

func (l ListMerge) String() string { return l.name }

var (
	ListMergeReplace = ListMerge{name: "atomic", merge: func(_, override []any) []any { return override }}
	ListMergeSet     = ListMerge{name: "set", merge: mergeSet}
	ListMergeOrdered = ListMerge{name: "ordered", merge: mergeOrdered}
)

// Merge deep-merges override into base, applying the per-path list strategy.
// Nested maps merge recursively; scalars and unmatched lists are taken from
// override; a list whose path matches a strategy is merged by it. Paths are
// dotted gjson-style keys with "*" matching any single segment, e.g.
// "service.pipelines.*.receivers". A nil strategy merges every list by
// ListMergeReplace — a plain deep-merge. base is mutated.
func Merge(base, override map[string]any, strategy map[string]ListMerge) {
	mergeMap(base, override, nil, strategy)
}

func mergeMap(base, override map[string]any, path []string, strategy map[string]ListMerge) {
	for key, overrideVal := range override {
		// Cap the slice so the recursive append can't alias the parent's path.
		keyPath := append(path[:len(path):len(path)], key)

		baseVal, ok := base[key]
		if !ok {
			base[key] = overrideVal
			continue
		}

		if baseMap, ok := baseVal.(map[string]any); ok {
			if overrideMap, ok := overrideVal.(map[string]any); ok {
				mergeMap(baseMap, overrideMap, keyPath, strategy)
				continue
			}
		}

		if baseList, ok := baseVal.([]any); ok {
			if overrideList, ok := overrideVal.([]any); ok {
				base[key] = strategyAt(keyPath, strategy).merge(baseList, overrideList)
				continue
			}
		}

		base[key] = overrideVal
	}
}

// strategyAt returns the strategy whose dotted, "*"-wildcard pattern matches
// path, or ListMergeReplace if none do.
func strategyAt(path []string, strategy map[string]ListMerge) ListMerge {
	for pattern, l := range strategy {
		if matchPath(strings.Split(pattern, "."), path) {
			return l
		}
	}
	return ListMergeReplace
}

func matchPath(pattern, path []string) bool {
	if len(pattern) != len(path) {
		return false
	}
	for i := range pattern {
		if pattern[i] != "*" && pattern[i] != path[i] {
			return false
		}
	}
	return true
}

// mergeSet unions scalar elements, base first, deduped. A list of maps degrades
// to atomic replacement (the override wins) rather than hashing an unhashable.
func mergeSet(base, override []any) []any {
	if !scalarList(base) || !scalarList(override) {
		return override
	}
	return unionScalars(base, override)
}

// mergeOrdered keeps base's order and inserts the override's non-base elements
// just before base's terminal element. A list of maps degrades to atomic.
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

// scalarList reports whether every element is a comparable scalar, i.e. safe to
// use as a map key. Lists of maps or lists are not.
func scalarList(list []any) bool {
	for _, v := range list {
		switch v.(type) {
		case map[string]any, []any:
			return false
		}
	}
	return true
}

// unionScalars returns base followed by override's elements not already present,
// de-duplicated and order-preserving.
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
