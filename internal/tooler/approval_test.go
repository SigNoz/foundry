package tooler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApproved(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		pass bool
	}{
		{name: "Stamped_Valid", ctx: WithApproval(context.Background()), pass: true},
		{name: "Bare_Invalid", ctx: context.Background()},
		{name: "OtherValue_Invalid", ctx: context.WithValue(context.Background(), approvalKey{}, "yes")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.pass, Approved(tt.ctx))
		})
	}
}
