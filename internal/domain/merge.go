package domain

import "strings"

// ListMerge is how a list merges when base and override both define one at the
// same path, mirroring Kubernetes' listType. The variant carries its own merge
// func, so adding one is a single var entry. Set and Ordered assume scalar
// elements and degrade to Replace on a list of maps.
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

// Merge deep-merges override into base, mutating base. Maps merge recursively;
// a list is merged by the strategy whose dotted-gjson path matches (e.g.
// "service.pipelines.*.receivers"), defaulting to ListMergeReplace. A nil
// strategy is a plain deep-merge.
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
	return orderedScalars(base, override)
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

// orderedScalars keeps base's order and inserts override's non-base elements
// just before base's terminal element.
func orderedScalars(base, override []any) []any {
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
