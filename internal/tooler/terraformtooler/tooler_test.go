package terraformtooler

import (
	"context"
	"log/slog"
	"testing"

	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/tooler"
	"github.com/stretchr/testify/assert"
)

type otherTooler struct{}

func (otherTooler) Name() string                  { return "other" }
func (otherTooler) Gauge(_ context.Context) error { return nil }

func TestLookup(t *testing.T) {
	terraform := New(slog.New(slog.DiscardHandler))

	tests := []struct {
		name    string
		toolers []tooler.Tooler
		pass    bool
	}{
		{name: "Only_Valid", toolers: []tooler.Tooler{terraform}, pass: true},
		{name: "AmongOthers_Valid", toolers: []tooler.Tooler{otherTooler{}, terraform}, pass: true},
		{name: "Empty_Invalid", toolers: nil},
		{name: "OnlyOthers_Invalid", toolers: []tooler.Tooler{otherTooler{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := Lookup(tt.toolers)

			if !tt.pass {
				assert.Error(t, err)
				assert.Nil(t, found)

				return
			}

			assert.NoError(t, err)
			assert.Same(t, terraform, found)
		})
	}
}

func TestValidate(t *testing.T) {
	complete := Release{
		Release: domain.Release{Name: "signoz", Owner: domain.Owner{"foundry.signoz.io/name": "signoz"}},
		Root:    "pours/infrastructure",
	}

	without := func(mutate func(*Release)) Release {
		release := complete
		mutate(&release)

		return release
	}

	tests := []struct {
		name    string
		release Release
		pass    bool
	}{
		{name: "Complete_Valid", release: complete, pass: true},
		{name: "UnstatedName_Invalid", release: without(func(r *Release) { r.Name = "" })},
		{name: "UnstatedOwner_Invalid", release: without(func(r *Release) { r.Owner = nil })},
		{name: "UnstatedRoot_Invalid", release: without(func(r *Release) { r.Root = "" })},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.release.Validate()

			if !tt.pass {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

// The mutation verbs refuse an unapproved context before they invoke anything,
// which is the only place --yes is enforced. The release is deliberately
// unstated: reaching Validate at all would mean the gate was passed.
func TestVerbsRefuseWithoutApproval(t *testing.T) {
	terraform := New(slog.New(slog.DiscardHandler))

	tests := []struct {
		name string
		verb func(context.Context, Release) error
	}{
		{name: "Apply_Invalid", verb: terraform.Apply},
		{name: "Destroy_Invalid", verb: terraform.Destroy},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorContains(t, tt.verb(context.Background(), Release{}), "re-run with --yes")
		})
	}
}
