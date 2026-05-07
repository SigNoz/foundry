package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplateRender(t *testing.T) {
	tests := []struct {
		name        string
		contents    []byte
		format      Format
		path        string
		data        any
		pass        bool
		expectedFmt []byte
	}{
		{
			name:        "YAML_Valid",
			contents:    []byte("greeting: hello {{ .Name }}\n"),
			format:      FormatYAML,
			path:        "out.yaml",
			data:        map[string]string{"Name": "world"},
			pass:        true,
			expectedFmt: []byte("greeting: hello world\n"),
		},
		{
			name:        "JSON_Valid",
			contents:    []byte(`{"greeting":"hello {{ .Name }}"}`),
			format:      FormatJSON,
			path:        "out.json",
			data:        map[string]string{"Name": "world"},
			pass:        true,
			expectedFmt: []byte(`{"greeting":"hello world"}`),
		},
		{
			name:        "INI_Valid",
			contents:    []byte("[greeting]\nname = {{ .Name }}\n"),
			format:      FormatINI,
			path:        "out.ini",
			data:        map[string]string{"Name": "world"},
			pass:        true,
			expectedFmt: []byte("[greeting]\nname=world\n"),
		},
		{
			name:        "Text_Valid",
			contents:    []byte("hello {{ .Name }}"),
			format:      FormatText,
			path:        "out.txt",
			data:        map[string]string{"Name": "world"},
			pass:        true,
			expectedFmt: []byte("hello world"),
		},
		{
			name:     "UnsupportedFormat_Invalid",
			contents: []byte("hello"),
			format:   Format{s: "bogus"},
			path:     "out",
			data:     nil,
			pass:     false,
		},
		{
			name:     "JSON_InvalidOutput",
			contents: []byte(`{"greeting": hello {{ .Name }}}`),
			format:   FormatJSON,
			path:     "out.json",
			data:     map[string]string{"Name": "world"},
			pass:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := MustNewTemplate(tt.name, tt.contents, tt.format)

			material, err := tmpl.Render(tt.data, tt.path)
			if !tt.pass {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.path, material.Path())
			assert.Equal(t, tt.expectedFmt, material.FmtContents())
		})
	}
}
