package domain

import "fmt"

var _ StructuredMaterial = INIMaterial{}

type INIMaterial struct {
	path     string
	contents []byte
}

func NewINIMaterial(contents []byte, path string) (INIMaterial, error) {
	jsonContents, err := INIToJSON(contents)
	if err != nil {
		return INIMaterial{}, fmt.Errorf("invalid ini: %w", err)
	}

	return INIMaterial{
		contents: jsonContents,
		path:     path,
	}, nil
}

func (m INIMaterial) Path() string {
	return m.path
}

func (m INIMaterial) Contents() []byte {
	return m.contents
}

func (m INIMaterial) IsMultiDoc() bool {
	return false
}

func (m INIMaterial) FmtContents() []byte {
	fmtContents, err := JSONToINI(m.contents)
	if err != nil {
		return nil
	}
	return fmtContents
}

func (m INIMaterial) WithContents(contents []byte) StructuredMaterial {
	return INIMaterial{
		contents: contents,
		path:     m.path,
	}
}

func (m INIMaterial) GetBytes(path string) ([]byte, error) {
	return getBytes(m.contents, path)
}

func (m INIMaterial) GetStringSlice(path string) ([]string, error) {
	return getStringSlice(m.contents, path)
}
