package common

import (
	"errors"
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"gopkg.in/ini.v1"
)

// KeyTransformer defines how to transform keys.
type KeyTransformer func(key string) string

// IdentityTransformer keeps keys as-is.
func IdentityTransformer(key string) string {
	return key
}

// UpperTransformer converts keys to uppercase.
func UpperTransformer(key string) string {
	return strings.ToUpper(key)
}

// valueToString converts any CUE value to string using Decode.
func valueToString(value cue.Value) (string, error) {
	var result interface{}
	if err := value.Decode(&result); err != nil {
		return "", errors.New("failed to decode CUE value to interface{}: " + err.Error())
	}
	return fmt.Sprintf("%v", result), nil
}

func MapToINI(data cue.Value) ([]byte, error) {
	return mapToFormat(data, IdentityTransformer, "ini")
}

func MapToEnv(data cue.Value) ([]byte, error) {
	fields, err := data.Fields()
	if err != nil {
		return nil, errors.New("failed to get fields from CUE value: " + err.Error())
	}

	var buf strings.Builder
	for fields.Next() {
		key := fields.Selector().String()
		value := fields.Value()

		strValue, err := valueToString(value)
		if err != nil {
			return nil, errors.New("failed to convert value to string for key '" + key + "': " + err.Error())
		}

		transformedKey := UpperTransformer(key)
		buf.WriteString(fmt.Sprintf("%s=%s\n", transformedKey, strValue))
	}

	return []byte(buf.String()), nil
}

// PathToEnvKey converts a config path to an environment variable key.
func PathToEnvKey(path, prefix string) string {
	key := strings.ToUpper(strings.ReplaceAll(path, ".", "_"))
	if prefix != "" {
		return prefix + "_" + key
	}
	return key
}

// GenerateEnvFromPaths generates an environment file from a CUE config.
func GenerateEnvFromPaths(config cue.Value, paths []string, prefix string) ([]byte, error) {
	if len(paths) == 0 {
		return nil, errors.New("no paths provided for environment variable generation")
	}

	var buf strings.Builder
	extractedCount := 0

	for _, path := range paths {
		value := config.LookupPath(cue.ParsePath(path))
		if !value.Exists() {
			continue
		}

		strValue, err := valueToString(value)
		if err != nil {
			return nil, errors.New("failed to convert value to string for path '" + path + "' (prefix: '" + prefix + "'): " + err.Error())
		}

		if strValue == "" {
			continue
		}

		envKey := PathToEnvKey(path, prefix)
		buf.WriteString(fmt.Sprintf("%s=%s\n", envKey, strValue))
		extractedCount++
	}

	if extractedCount == 0 {
		return nil, errors.New("no valid values found for any of the " + fmt.Sprintf("%d", len(paths)) + " provided paths (prefix: '" + prefix + "')")
	}

	return []byte(buf.String()), nil
}

// GenerateSigNozEnv generates an environment file from the signoz config.
func GenerateSigNozEnv(signozConfig cue.Value) ([]byte, error) {
	paths := []string{
		"telemetrystore.clickhouse.dsn",
		"sqlstore.postgres.dsn",
		"sqlstore.provider",
		"web.enabled",
	}
	return GenerateEnvFromPaths(signozConfig, paths, "SIGNOZ")
}

// Core function that handles both INI and ENV generation.
func mapToFormat(data cue.Value, transformer KeyTransformer, formatType string) ([]byte, error) {
	cfg := ini.Empty()

	fields, err := data.Fields()
	if err != nil {
		return nil, errors.New("failed to get fields from CUE value: " + err.Error())
	}

	for fields.Next() {
		key := fields.Selector().String()
		value := fields.Value()

		strValue, err := valueToString(value)
		if err != nil {
			return nil, errors.New("failed to convert value to string for key '" + key + "': " + err.Error())
		}

		// Handle nested structures by creating sections
		if _, err := value.Fields(); err == nil {
			section, err := cfg.NewSection(key)
			if err != nil {
				return nil, errors.New("failed to create section '" + key + "': " + err.Error())
			}

			nestedFields, err := value.Fields()
			if err != nil {
				return nil, errors.New("failed to get nested fields for section '" + key + "': " + err.Error())
			}

			for nestedFields.Next() {
				nestedKey := nestedFields.Selector().String()
				nestedValue := nestedFields.Value()

				nestedStrValue, err := valueToString(nestedValue)
				if err != nil {
					return nil, errors.New("failed to convert nested value to string for key '" + nestedKey + "' in section '" + key + "': " + err.Error())
				}

				transformedKey := transformer(nestedKey)
				_, err = section.NewKey(transformedKey, nestedStrValue)
				if err != nil {
					return nil, errors.New("failed to create key '" + transformedKey + "' in section '" + key + "': " + err.Error())
				}
			}
		} else {
			// Simple key-value pair
			transformedKey := transformer(key)
			_, err := cfg.Section("").NewKey(transformedKey, strValue)
			if err != nil {
				return nil, errors.New("failed to create key '" + transformedKey + "': " + err.Error())
			}
		}
	}

	// Convert to bytes
	var buf strings.Builder
	_, err = cfg.WriteTo(&buf)
	if err != nil {
		return nil, errors.New("failed to write " + formatType + " format: " + err.Error())
	}

	return []byte(buf.String()), nil
}
