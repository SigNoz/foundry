package output

import (
	"fmt"
	"os"
	"path/filepath"
)

// Manager handles writing output files
type Manager struct {
	targetDir string // Output directory (./pours)
}

// NewManager creates a new output manager
func NewManager(target string) (*Manager, error) {
	// Create output directory
	if err := os.MkdirAll(target, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &Manager{
		targetDir: target,
	}, nil
}

// WriteComponent writes component configuration files
func (m *Manager) WriteComponent(name string, files map[string][]byte) error {
	// Create component directory
	componentDir := filepath.Join(m.targetDir, name)
	if err := os.MkdirAll(componentDir, 0755); err != nil {
		return fmt.Errorf("failed to create component directory %s: %w", name, err)
	}

	// Write all files for this component
	for filename, content := range files {
		fullPath := filepath.Join(componentDir, filename)
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}

	return nil
}

// WriteOrchestration writes orchestration files (docker-compose, etc.)
func (m *Manager) WriteOrchestration(files map[string][]byte) error {
	// Write all orchestration files to root of output directory
	for filename, content := range files {
		fullPath := filepath.Join(m.targetDir, filename)
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}

	return nil
}
