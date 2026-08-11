package domain

import "strings"

// Ownership captures which foundry Kinds own a group of platform workloads,
// derived from the foundry.signoz.io/kind label carried by each workload.
type Ownership struct {
	kinds     []string
	unlabeled bool
}

// ParseOwnership derives ownership from label values, one workload per line;
// an empty line is a workload without the label.
func ParseOwnership(labels string) Ownership {
	lines := strings.Split(labels, "\n")

	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}

	ownership := Ownership{}
	seen := map[string]bool{}

	for _, line := range lines {
		kind := strings.TrimSpace(line)

		if kind == "" {
			ownership.unlabeled = true
			continue
		}

		if seen[kind] {
			continue
		}

		seen[kind] = true
		ownership.kinds = append(ownership.kinds, kind)
	}

	return ownership
}

// Foreign returns the owning Kind that is not self, when one exists.
func (ownership Ownership) Foreign(self string) (string, bool) {
	for _, kind := range ownership.kinds {
		if kind != self {
			return kind, true
		}
	}

	return "", false
}

// HasUnlabeled reports workloads carrying no ownership label: either a
// pre-label foundry deployment or a foreign project sharing the name.
func (ownership Ownership) HasUnlabeled() bool {
	return ownership.unlabeled
}
