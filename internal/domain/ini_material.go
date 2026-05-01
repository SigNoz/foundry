package domain

import "fmt"

var _ StructuredMaterial = INIMaterial{}

type INIMaterial struct {
	structuredData
}

func NewINIMaterial(contents []byte, path string) (INIMaterial, error) {
	jsonContents, err := INIToJSON(contents)
	if err != nil {
		return INIMaterial{}, fmt.Errorf("invalid ini: %w", err)
	}

	return INIMaterial{
		structuredData: structuredData{
			contents: jsonContents,
			path:     path,
		},
	}, nil
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
		structuredData: structuredData{
			contents: contents,
			path:     m.path,
		},
	}
}
