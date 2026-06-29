package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionedTemplatesResolve(t *testing.T) {
	old := MustNewTemplate("old", []byte("old"), FormatText)
	latest := MustNewTemplate("latest", []byte("latest"), FormatText)
	vt := MustNewVersionedTemplates("25.12.5", map[string]*Template{
		"25.5.6":  old,
		"25.12.5": latest,
	})

	tests := []struct {
		name     string
		version  string
		expected *Template
		exact    bool
	}{
		{"KnownOldVersion_ResolvesExact", "25.5.6", old, true},
		{"KnownLatestVersion_ResolvesExact", "25.12.5", latest, true},
		{"UnknownVersion_FallsBackToLatest", "26.0.0", latest, false},
		{"EmptyVersion_FallsBackToLatest", "", latest, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, exact := vt.Resolve(tt.version)
			assert.Equal(t, tt.expected, got)
			assert.Equal(t, tt.exact, exact)
		})
	}
}

func TestNewVersionedTemplates(t *testing.T) {
	tmpl := MustNewTemplate("t", []byte("t"), FormatText)

	tests := []struct {
		name      string
		latest    string
		templates map[string]*Template
		pass      bool
	}{
		{"LatestPresent_Valid", "25.12.5", map[string]*Template{"25.12.5": tmpl}, true},
		{"LatestMissing_Invalid", "25.12.5", map[string]*Template{"25.5.6": tmpl}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt, err := NewVersionedTemplates(tt.latest, tt.templates)
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, vt)
		})
	}
}

func TestMustNewVersionedTemplates_PanicsOnMissingLatest(t *testing.T) {
	tmpl := MustNewTemplate("t", []byte("t"), FormatText)
	assert.Panics(t, func() {
		MustNewVersionedTemplates("25.12.5", map[string]*Template{"25.5.6": tmpl})
	})
}
