package common

import (
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
		return "", err
	}
	return fmt.Sprintf("%v", result), nil
}

func MapToINI(data cue.Value) ([]byte, error) {
	return mapToFormat(data, IdentityTransformer, "ini")
}

func MapToEnv(data cue.Value) ([]byte, error) {
	return mapToFormat(data, UpperTransformer, "env")
}

// Core function that handles both INI and ENV generation.
func mapToFormat(data cue.Value, transformer KeyTransformer, formatType string) ([]byte, error) {
	cfg := ini.Empty()
	
	fields, err := data.Fields()
	if err != nil {
		return nil, fmt.Errorf("failed to get fields: %w", err)
	}
	
	for fields.Next() {
		key := fields.Selector().String()
		value := fields.Value()
		
		strValue, err := valueToString(value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert value to string for key '%s': %w", key, err)
		}
		
		// Handle nested structures by creating sections
		if _, err := value.Fields(); err == nil {
			section, err := cfg.NewSection(key)
			if err != nil {
				return nil, fmt.Errorf("failed to create section '%s': %w", key, err)
			}
			
			nestedFields, err := value.Fields()
			if err != nil {
				return nil, fmt.Errorf("failed to get nested fields for section '%s': %w", key, err)
			}
			
			for nestedFields.Next() {
				nestedKey := nestedFields.Selector().String()
				nestedValue := nestedFields.Value()
				
				nestedStrValue, err := valueToString(nestedValue)
				if err != nil {
					return nil, fmt.Errorf("failed to convert nested value to string for key '%s' in section '%s': %w", nestedKey, key, err)
				}
				
				transformedKey := transformer(nestedKey)
				_, err = section.NewKey(transformedKey, nestedStrValue)
				if err != nil {
					return nil, fmt.Errorf("failed to create key '%s' in section '%s': %w", transformedKey, key, err)
				}
			}
		} else {
			// Simple key-value pair
			transformedKey := transformer(key)
			_, err := cfg.Section("").NewKey(transformedKey, strValue)
			if err != nil {
				return nil, fmt.Errorf("failed to create key '%s': %w", transformedKey, err)
			}
		}
	}
	
	// Convert to bytes
	var buf strings.Builder
	_, err = cfg.WriteTo(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to write %s: %w", formatType, err)
	}
	
	return []byte(buf.String()), nil
}