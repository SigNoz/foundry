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

	var sectionKeys []string
	var flatKeys []string

	for key, value := range data {
		// Check if the value is a nested map (a Section)
		// or a simple value (a flat key for DEFAULT)
		if _, ok := value.(map[string]any); ok {
			sectionKeys = append(sectionKeys, key)
		} else {
			flatKeys = append(flatKeys, key)
		}
	}

	sort.Strings(sectionKeys)
	for _, k := range sectionKeys {
		sec, _ := cfg.NewSection(k)

		m := data[k].(map[string]any)
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}

		sort.Strings(keys)
		for _, k := range keys {
			if _, err := sec.NewKey(k, fmt.Sprint(m[k])); err != nil {
				return nil, err
			}
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
