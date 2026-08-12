package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRole(t *testing.T) {
	tests := []struct {
		name         string
		role         Role
		expectedWord string
	}{
		{name: "Node_Rendered", role: RoleNode, expectedWord: "node"},
		{name: "Task_Rendered", role: RoleTask, expectedWord: "task"},
		{name: "Exec_Rendered", role: RoleExec, expectedWord: "exec"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedWord, tt.role.String())
		})
	}
}

// Two roles sharing a rendering would collide in a derived name.
func TestRolesAreDistinct(t *testing.T) {
	roles := []Role{RoleNode, RoleTask, RoleExec}

	seen := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		assert.NotContains(t, seen, role.String())
		seen[role.String()] = struct{}{}
	}
}
