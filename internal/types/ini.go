package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/ini.v1"
	"slices"
)

// INI/Env bytes -> JSON bytes.
func INIToJSON(contents []byte) ([]byte, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		AllowShadows: true,
	}, contents)
	if err != nil {
		return nil, err
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

	return json.MarshalIndent(data, "", "  ")
}

// JSONToINI converts JSON to Systemd INI format.
func JSONToINI(contents []byte) ([]byte, error) {
	var data map[string]map[string]any
	if err := json.Unmarshal(contents, &data); err != nil {
		return nil, err
	}

	cfg, err := ini.LoadSources(ini.LoadOptions{
		AllowShadows:            true,
		PreserveSurroundedQuote: true,
	}, []byte(""))

	if err != nil {
		return nil, err
	}
	ini.PrettyFormat = false
	// Process Sections
	for _, sName := range getSortedKeys(data) {
		section, _ := cfg.NewSection(sName)

		// Process Keys (Sorted)
		for _, kName := range getSortedKeys(data[sName]) {
			if err := writeEntry(section, kName, data[sName][kName]); err != nil {
				return nil, err
			}
		}
	}

	var buf bytes.Buffer
	_, err = cfg.WriteTo(&buf)
	return buf.Bytes(), err
}

// writeEntry adds the "Shadow" (multiple keys).
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

// getSortedKeys is a generic helper that returns keys in consistent order.
// This ensures generated files have stable output regardless of map iteration order.
func getSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
