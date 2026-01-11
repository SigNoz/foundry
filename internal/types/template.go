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
	format Format
	tmpl   *template.Template
}

func NewTemplateFromFS(fs embed.FS, path string, format Format) (*Template, error) {
	name := filepath.Base(path)
	tmpl, err := template.New(name).Funcs(sprig.FuncMap()).ParseFS(fs, path)
	if err != nil {
		return nil, err
	}

	return &Template{name: name, tmpl: tmpl, format: format}, nil
}

func MustNewTemplateFromFS(fs embed.FS, path string, format Format) *Template {
	tmpl, err := NewTemplateFromFS(fs, path, format)
	if err != nil {
		panic(err)
	}

	return tmpl
}

func NewTemplate(name string, contents []byte) (*Template, error) {
	tmpl, err := template.New(name).Funcs(sprig.FuncMap()).Parse(string(contents))
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
