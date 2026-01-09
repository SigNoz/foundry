package template

import (
	"embed"
	"io"
	"path/filepath"
	"text/template"
)

type Template struct {
	name string
	tmpl *template.Template
}

func New(fs embed.FS, path string) (*Template, error) {
	name := filepath.Base(path)
	tmpl, err := template.New(name).ParseFS(fs, path)
	if err != nil {
		return nil, err
	}

	return &Template{name: name, tmpl: tmpl}, nil
}

func MustNew(fs embed.FS, path string) *Template {
	tmpl, err := New(fs, path)
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

func funcMap() template.FuncMap {
	return template.FuncMap{
		"multiply": func(a, b int) int {
			return a * b
		},
	}
}
