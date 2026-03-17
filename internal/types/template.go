package types

import (
	"embed"
	"io"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

type Template struct {
	name   string
	path   string
	format Format
	tmpl   *template.Template
}

// customFuncMap returns a template.FuncMap with custom functions merged with sprig functions.
func customFuncMap() template.FuncMap {
	funcMap := sprig.FuncMap()

	// Add custom functions
	funcMap["derefInt"] = func(p *int) int {
		if p == nil {
			return 0
		}
		return *p
	}

	funcMap["derefIntDefault"] = func(p *int, defaultVal int) int {
		if p == nil {
			return defaultVal
		}
		return *p
	}

	return funcMap
}

func NewTemplateFromFS(fs embed.FS, path string, format Format) (*Template, error) {
	name := filepath.Base(path)
	tmpl, err := template.New(name).Funcs(customFuncMap()).ParseFS(fs, path)
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
	tmpl, err := template.New(name).Funcs(customFuncMap()).Parse(string(contents))
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
