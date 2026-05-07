package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/signoz/foundry/internal/errors"
	"gopkg.in/ini.v1"
)

func init() {
	ini.PrettyFormat = false
}

var _ StructuredMaterial = INIMaterial{}

type INIMaterial struct {
	path     string
	contents []byte
}

func NewINIMaterial(contents []byte, path string) (INIMaterial, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{AllowShadows: true}, contents)
	if err != nil {
		return INIMaterial{}, errors.Wrapf(err, errors.TypeInvalidInput, "failed to create INI material for path %q: contents are not valid INI", path)
	}

	data := make(map[string]map[string]any)

	for _, section := range cfg.Sections() {
		if section.Name() == ini.DefaultSection && len(section.Keys()) == 0 {
			continue
		}

		sectionData := make(map[string]any)
		for _, key := range section.Keys() {
			vals := key.ValueWithShadows()

			if len(vals) > 1 {
				sectionData[key.Name()] = vals
			} else {
				sectionData[key.Name()] = key.String()
			}
		}
		data[section.Name()] = sectionData
	}

	jsonContents, err := json.Marshal(data)
	if err != nil {
		return INIMaterial{}, errors.Wrapf(err, errors.TypeInternal, "failed to convert INI material to canonical JSON for path %q", path)
	}

	return INIMaterial{
		path:     path,
		contents: jsonContents,
	}, nil
}

func MustNewINIMaterial(contents []byte, path string) INIMaterial {
	material, err := NewINIMaterial(contents, path)
	if err != nil {
		panic(err)
	}

	return material
}

func (m INIMaterial) Path() string {
	return m.path
}

func (m INIMaterial) JSONContents() []byte {
	return m.contents
}

func (m INIMaterial) HasMultipleDocuments() bool {
	return false
}

func (m INIMaterial) FmtContents() []byte {
	var data map[string]map[string]any
	if err := json.Unmarshal(m.contents, &data); err != nil {
		return nil
	}

	cfg, err := ini.LoadSources(ini.LoadOptions{AllowShadows: true, PreserveSurroundedQuote: true}, []byte(""))
	if err != nil {
		return nil
	}

	for _, sName := range getSortedKeys(data) {
		section, _ := cfg.NewSection(sName)

		for _, kName := range getSortedKeys(data[sName]) {
			if err := writeEntry(section, kName, data[sName][kName]); err != nil {
				return nil
			}
		}
	}

	var buf bytes.Buffer
	if _, err := cfg.WriteTo(&buf); err != nil {
		return nil
	}

	return buf.Bytes()
}

func (m INIMaterial) CloneWithJSONContents(contents []byte) StructuredMaterial {
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

func writeEntry(sec *ini.Section, key string, value any) error {
	if vals, ok := value.([]any); ok {
		for i, v := range vals {
			strVal := fmt.Sprint(v)
			if i == 0 {
				if _, err := sec.NewKey(key, strVal); err != nil {
					return err
				}
			} else {
				if err := sec.Key(key).AddShadow(strVal); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if _, err := sec.NewKey(key, fmt.Sprint(value)); err != nil {
		return err
	}

	return nil
}

func getSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
