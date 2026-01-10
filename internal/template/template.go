package template

import (
	"embed"
	"io"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

type Template struct {
	name string
	tmpl *template.Template
}

func NewFromFS(fs embed.FS, path string) (*Template, error) {
	name := filepath.Base(path)
	tmpl, err := template.New(name).Funcs(sprig.FuncMap()).ParseFS(fs, path)
	if err != nil {
		return nil, err
	}

	return &Template{name: name, tmpl: tmpl}, nil
}

func MustNewFromFS(fs embed.FS, path string) *Template {
	tmpl, err := NewFromFS(fs, path)
	if err != nil {
		panic(err)
	}

	return tmpl
}

func New(name string, contents []byte) (*Template, error) {
	tmpl, err := template.New(name).Funcs(sprig.FuncMap()).Parse(string(contents))
	if err != nil {
		return nil, err
	}

	return &Template{name: name, tmpl: tmpl}, nil
}

func MustNew(name string, contents []byte) *Template {
	tmpl, err := New(name, contents)
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
