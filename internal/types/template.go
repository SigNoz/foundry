package types

import (
	"embed"
	"io"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"sigs.k8s.io/yaml"
)

type Template struct {
	name   string
	path   string
	format Format
	tmpl   *template.Template
}

// templateFuncMap returns the function map for templates (sprig + toYaml).
func templateFuncMap() template.FuncMap {
	fm := template.FuncMap(sprig.FuncMap())
	fm["toYaml"] = func(v any) (string, error) {
		if v == nil {
			return "", nil
		}
		b, err := yaml.Marshal(v)
		return string(b), err
	}
	return fm
}

func NewTemplateFromFS(fs embed.FS, path string, format Format) (*Template, error) {
	name := filepath.Base(path)
	tmpl, err := template.New(name).Funcs(templateFuncMap()).ParseFS(fs, path)
	if err != nil {
		return nil, err
	}

	return &Template{name: name, path: path, tmpl: tmpl, format: format}, nil
}

func MustNewTemplateFromFS(fs embed.FS, path string, format Format) *Template {
	tmpl, err := NewTemplateFromFS(fs, path, format)
	if err != nil {
		panic(err)
	}

	return tmpl
}

func NewTemplate(name string, contents []byte) (*Template, error) {
	tmpl, err := template.New(name).Funcs(templateFuncMap()).Parse(string(contents))
	if err != nil {
		return nil, err
	}

	return &Template{name: name, tmpl: tmpl}, nil
}

func MustNewTemplate(name string, contents []byte) *Template {
	tmpl, err := NewTemplate(name, contents)
	if err != nil {
		panic(err)
	}

	return tmpl
}

func (t *Template) Execute(w io.Writer, data any) error {
	newtmpl, err := t.tmpl.Clone()
	if err != nil {
		return err
	}

	return newtmpl.ExecuteTemplate(w, t.name, data)
}

func (t *Template) GetName() string {
	return t.name
}

func (t *Template) GetPath() string {
	return t.path
}
