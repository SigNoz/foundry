package output

import (
	"errors"
	"os"
	"path/filepath"
)

// Manager handles writing output files.
type Manager struct {
	targetDir string // Output directory (./pours)
}

// NewManager creates a new output manager.
func NewManager(target string) (*Manager, error) {
	// Create output directory
	if err := os.MkdirAll(target, 0755); err != nil {
		return nil, errors.New("failed to create output directory: " + err.Error())
	}

	return &Manager{
		targetDir: target,
	}, nil
}

// WriteComponent writes component configuration files.
func (m *Manager) WriteComponent(name string, files map[string][]byte) error {
	// Create component directory
	componentDir := filepath.Join(m.targetDir, name)
	if err := os.MkdirAll(componentDir, 0755); err != nil {
		return errors.New("failed to create component directory " + name + ": " + err.Error())
	}

	// Write all files for this component
	for filename, content := range files {
		fullPath := filepath.Join(componentDir, filename)
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			return errors.New("failed to write file " + fullPath + ": " + err.Error())
		}
	}

	return nil
}

// WriteOrchestration writes orchestration files (docker-compose, etc.)
func (m *Manager) WriteOrchestration(files map[string][]byte) error {
	for filename, content := range files {
		fullPath := filepath.Join(m.targetDir, filename)
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			return errors.New("failed to write file " + fullPath + ": " + err.Error())
		}
	}

	return nil
}
