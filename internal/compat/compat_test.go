package compat

import (
	"io"
	"log/slog"
	"testing"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestCheck(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rules := []Rule{
		{Subject: v1alpha1.MoldingKindIngester, When: ">0.144.5", Target: v1alpha1.MoldingKindTelemetryStore, Requires: "=25.12.5", Advice: "advice"},
	}

	enabled := func(tag string) Resolved { return NewResolved("component:"+tag, true) }

	tests := []struct {
		name      string
		collector Resolved
		store     Resolved
		pass      bool
	}{
		{"NewCollector_OldStore_Fails", enabled("0.144.6"), enabled("25.5.6"), false},
		{"NewCollector_NewStore_OK", enabled("0.144.6"), enabled("25.12.5"), true},
		{"NewCollector_NewStoreAlpine_OK", enabled("0.144.6"), enabled("25.12.5-alpine"), true},
		{"FloorCollector_OldStore_OK", enabled("0.144.5"), enabled("25.5.6"), true},
		{"FloatingCollector_OldStore_Warns", enabled("latest"), enabled("25.5.6"), true},
		{"FloatingCollector_NewStore_OK", enabled("latest"), enabled("25.12.5"), true},
		{"DisabledCollector_OldStore_OK", NewResolved("component:0.144.6", false), enabled("25.5.6"), true},
		{"UnknownStore_Skips", enabled("0.144.6"), enabled("latest"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			versions := map[v1alpha1.MoldingKind]Resolved{
				v1alpha1.MoldingKindIngester:       tt.collector,
				v1alpha1.MoldingKindTelemetryStore: tt.store,
			}

			err := Check(versions, rules, logger)
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
