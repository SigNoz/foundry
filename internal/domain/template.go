package domain

import (
	"bytes"
	"embed"
	"io"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/signoz/foundry/internal/errors"
	"sigs.k8s.io/yaml"
)

// Template is a parsed text/template paired with the on-disk path it was loaded
// from and the Format its rendered output is expected to take. Format drives
// Render's choice of concrete Material type.
type Template struct {
	name   string
	path   string
	format Format
	tmpl   *template.Template
}

func NewTemplateFromFS(fs embed.FS, path string, format Format) (*Template, error) {
	name := filepath.Base(path)
	tmpl, err := template.New(name).Funcs(templateFuncMap()).ParseFS(fs, path)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to create template from %q: contents are not a valid template", path)
	}

	return &Template{name: name, path: path, format: format, tmpl: tmpl}, nil
}

func NewTemplate(name string, contents []byte, format Format) (*Template, error) {
	tmpl, err := template.New(name).Funcs(templateFuncMap()).Parse(string(contents))
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInvalidInput, "failed to create template from %q: contents are not a valid template", name)
	}

	return &Template{name: name, format: format, tmpl: tmpl}, nil
}

func MustNewTemplateFromFS(fs embed.FS, path string, format Format) *Template {
	tmpl, err := NewTemplateFromFS(fs, path, format)
	if err != nil {
		panic(err)
	}

	return tmpl
}

func MustNewTemplate(name string, contents []byte, format Format) *Template {
	tmpl, err := NewTemplate(name, contents, format)
	if err != nil {
		panic(err)
	}

	return tmpl
}

// Execute renders the template into w. Each call clones the underlying
// *template.Template so concurrent calls don't share parse state.
func (t *Template) Execute(w io.Writer, data any) error {
	newtmpl, err := t.tmpl.Clone()
	if err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to execute template %q", t.name)
	}

	if err := newtmpl.ExecuteTemplate(w, t.name, data); err != nil {
		return errors.Wrapf(err, errors.TypeInternal, "failed to execute template %q", t.name)
	}

	return nil
}

// Render executes the template against data and wraps the output in the
// concrete Material type indicated by the template's Format.
func (t *Template) Render(data any, path string) (Material, error) {
	buf := bytes.NewBuffer(nil)
	if err := t.Execute(buf, data); err != nil {
		return nil, err
	}

	m, err := t.format.NewMaterial(buf.Bytes(), path)
	if err != nil {
		return nil, errors.Wrapf(err, errors.TypeInternal, "failed to render template %q", t.name)
	}
	return m, nil
}

func (t *Template) Name() string {
	return t.name
}

func (t *Template) Path() string {
	return t.path
}

func (t *Template) Format() Format {
	return t.format
}

// templateFuncMap layers the project's helpers on top of sprig:
//   - derefInt / derefIntDefault: nil-safe int pointer access for spec fields
//     that distinguish "unset" from zero.
//   - toYaml / fromYaml: round-trip a value through YAML inside templates.
//   - flattenKeys: flatten a nested map into "/"-joined leaf keys (used to
//     project hierarchical config into flat env-var-style maps).
func templateFuncMap() template.FuncMap {
	fm := template.FuncMap(sprig.FuncMap())
	fm["derefInt"] = func(p *int) int {
		if p == nil {
			return 0
		}
		return *p
	}
	fm["derefIntDefault"] = func(p *int, defaultVal int) int {
		if p == nil {
			return defaultVal
		}
		return *p
	}
	fm["toYaml"] = func(v any) (string, error) {
		if v == nil {
			return "", nil
		}
		b, err := yaml.Marshal(v)
		return string(b), err
	}
	fm["fromYaml"] = func(s string) (map[string]any, error) {
		var m map[string]any
		err := yaml.Unmarshal([]byte(s), &m)
		return m, err
	}
	fm["flattenKeys"] = func(m map[string]any) map[string]any {
		result := make(map[string]any)
		flattenMapKeys("", m, result)
		return result
	}
	return fm
}

func flattenMapKeys(prefix string, m map[string]any, result map[string]any) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "/" + k
		}
		if nested, ok := v.(map[string]any); ok {
			flattenMapKeys(key, nested, result)
		} else {
			result[key] = v
		}
	}
}
