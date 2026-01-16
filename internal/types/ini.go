package types

import (
	"bytes"
	"encoding/json"
	"fmt"

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

	for key, value := range data {
		// Check if the value is a nested map (a Section)
		// or a simple value (a flat key for DEFAULT)
		if sectionMap, ok := value.(map[string]any); ok {
			// It's a Section (e.g., "Service": {...})
			sec, _ := cfg.NewSection(key)
			for k, v := range sectionMap {
				if _, err := sec.NewKey(k, fmt.Sprint(v)); err != nil {
					return nil, err
				}
			}
		} else if sectionMapStr, ok := value.(map[string]string); ok {
			// Handle case where it might be map[string]string
			sec, _ := cfg.NewSection(key)
			for k, v := range sectionMapStr {
				if _, err := sec.NewKey(k, v); err != nil {
					return nil, err
				}
			}
		} else {
			// It's a flat key (e.g., "PORT": "8080"), put in DEFAULT section
			if _, err := cfg.Section("").NewKey(key, fmt.Sprint(value)); err != nil {
				return nil, err
			}
		}
	}

	var buf bytes.Buffer
	_, err := cfg.WriteTo(&buf)
	return buf.Bytes(), err
}
