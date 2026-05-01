package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/signoz/foundry/internal/errors"
	"gopkg.in/ini.v1"
)

var _ StructuredMaterial = INIMaterial{}

type INIMaterial struct {
	path     string
	contents []byte
}

func NewINIMaterial(contents []byte, path string) (INIMaterial, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{AllowShadows: true}, contents)
	if err != nil {
		return INIMaterial{}, err
	}

	data := make(map[string]map[string]any)

	for _, section := range cfg.Sections() {
		// Skip the default section if it's empty (common in systemd files)
		if section.Name() == ini.DefaultSection && len(section.Keys()) == 0 {
			continue
		}

		sectionData := make(map[string]any)
		for _, key := range section.Keys() {
			// Use ValueWithShadows() to get all values for this key
			vals := key.ValueWithShadows()

			if len(vals) > 1 {
				// If multiple values exist, store as an array
				sectionData[key.Name()] = vals
			} else {
				// If only one value exists, store as a string
				sectionData[key.Name()] = key.String()
			}
		}
		data[section.Name()] = sectionData
	}

	jsonContents, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return INIMaterial{}, errors.Wrapf(err, errors.TypeInvalidInput, "failed to create INI material for path %q, the contents are not valid INI", path)
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

func (m INIMaterial) IsMultiDoc() bool {
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

	ini.PrettyFormat = false
	// Process Sections
	for _, sName := range getSortedKeys(data) {
		section, _ := cfg.NewSection(sName)

		// Process Keys (Sorted)
		for _, kName := range getSortedKeys(data[sName]) {
			if err := writeEntry(section, kName, data[sName][kName]); err != nil {
				return nil
			}
		}
	}

	var buf bytes.Buffer
	_, err = cfg.WriteTo(&buf)

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

// Ensures generated files have stable output regardless of map iteration order.
func getSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
