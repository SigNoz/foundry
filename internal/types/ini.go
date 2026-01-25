package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"gopkg.in/ini.v1"
)

// INI/Env bytes -> JSON bytes.
func INIToJSON(contents []byte) ([]byte, error) {
	cfg, err := ini.Load(contents)
	if err != nil {
		return nil, err
	}
	ini.DefaultHeader = false
	data := make(map[string]any)

	for _, section := range cfg.Sections() {
		hash := section.KeysHash()
		if len(hash) == 0 {
			continue
		}

		// If it's the default section and it's the only thing there,
		// we can choose to keep it flat or nested.
		// For consistency with Systemd, we'll keep the section names.
		data[section.Name()] = hash
	}

	return json.Marshal(data)
}

// JSON bytes -> INI/Env bytes.
func JSONToINI(contents []byte) ([]byte, error) {
	var data map[string]any
	if err := json.Unmarshal(contents, &data); err != nil {
		return nil, err
	}

	cfg := ini.Empty()

	ini.PrettyFormat = false
	ini.DefaultHeader = false

	var flatKeys []string

	for key, value := range data {
		// Check if the value is a nested map (a Section)
		// or a simple value (a flat key for DEFAULT)
		if sectionMap, ok := value.(map[string]any); ok {
			// It's a Section (e.g., "Service": {...})
			sec, _ := cfg.NewSection(key)

			// sort the keys alphabetically
			keys := make([]string, 0, len(sectionMap))
			for k := range sectionMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				if _, err := sec.NewKey(k, fmt.Sprint(sectionMap[k])); err != nil {
					return nil, err
				}
			}
		} else {
			flatKeys = append(flatKeys, key)
		}
	}

	sort.Strings(flatKeys)
	for _, k := range flatKeys {
		if _, err := cfg.Section("").NewKey(k, fmt.Sprint(data[k])); err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	_, err := cfg.WriteTo(&buf)
	return buf.Bytes(), err
}
