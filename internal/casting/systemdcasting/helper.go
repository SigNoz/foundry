package systemdcasting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/types"
)

// --- Material Helpers ---

func (c *linuxCasting) renderTemplate(tmpl *types.Template, cfg *v1alpha1.Casting, path string) (types.Material, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return types.Material{}, fmt.Errorf("execute template %s: %w", path, err)
	}
	return types.NewINIMaterial(buf.Bytes(), path)
}

func (c *linuxCasting) envMaterial(envs map[string]string, prefix string) (types.Material, error) {
	if envs == nil {
		return types.Material{}, fmt.Errorf("envs not enriched for %s", prefix)
	}
	jb, _ := json.Marshal(envs)
	ib, err := types.JSONToINI(jb)
	if err != nil {
		return types.Material{}, fmt.Errorf("failed to convert env to INI: %w", err)
	}
	return types.NewINIMaterial(ib, fmt.Sprintf("%s/%s.env", prefix, prefix))
}

func (c *linuxCasting) configMaterials(data map[string]string, path string) ([]types.Material, error) {
	mats := make([]types.Material, 0, len(data))
	for file, content := range data {
		m, err := types.NewYAMLMaterial([]byte(content), filepath.Join(path, file))
		if err != nil {
			return nil, fmt.Errorf("failed to create config material %s: %w", file, err)
		}
		mats = append(mats, m)
	}
	return mats, nil
}
