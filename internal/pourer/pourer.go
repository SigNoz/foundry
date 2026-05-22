// Package pourer is the staging buffer a casting fills during Forge. Each
// Kind's Planner constructs a fresh Pourer for its output subdirectory, hands
// it to the casting, then pours its contents into domain.Materials after the
// casting returns.
//
// Castings interact only with Pourer methods (AddYAML, AddJSON, ...). They do
// not import domain.Material, domain.Format, or compute Kind-aware paths —
// those concerns live behind this package.
package pourer

import (
	"path/filepath"

	"github.com/signoz/foundry/internal/domain"
)

// Pourer stages a casting's outputs. The casting calls Add* methods to
// deposit content under relative path names; the Planner calls Pour to
// convert the staged entries into domain.Materials anchored at the Kind's
// output subdirectory.
type Pourer struct {
	kindDir string
	entries []entry
}

type entry struct {
	path    string
	format  domain.Format
	content []byte
}

// New constructs a Pourer staged at kindDir. Each Kind's Planner builds one
// per Forge / Cast call (e.g. pourer.New("agent") for CollectionAgent).
func New(kindDir string) *Pourer {
	return &Pourer{kindDir: kindDir}
}

// Dir returns the Kind's output subdirectory. Castings use this in Cast to
// locate already-written pours on disk by joining the user-supplied output
// target with the Kind dir.
func (p *Pourer) Dir() string {
	return p.kindDir
}

// AddYAML stages a YAML pour. The resulting material is structurally
// introspectable and targetable by spec.patches.
func (p *Pourer) AddYAML(content []byte, parts ...string) {
	p.add(domain.FormatYAML, content, parts...)
}

// AddJSON stages a JSON pour. The resulting material is structurally
// introspectable and targetable by spec.patches.
func (p *Pourer) AddJSON(content []byte, parts ...string) {
	p.add(domain.FormatJSON, content, parts...)
}

// AddINI stages an INI pour. The resulting material is structurally
// introspectable and targetable by spec.patches.
func (p *Pourer) AddINI(content []byte, parts ...string) {
	p.add(domain.FormatINI, content, parts...)
}

// AddBlob stages an opaque, byte-exact pour. The content is written
// verbatim; foundry does not introspect it. A spec.patches entry whose
// target matches this pour's path will cause forge to fail with an
// "unsupported" error — use a structured method (AddYAML, AddJSON,
// AddINI) for content that should remain patchable.
func (p *Pourer) AddBlob(content []byte, parts ...string) {
	p.add(domain.FormatText, content, parts...)
}

func (p *Pourer) add(format domain.Format, content []byte, parts ...string) {
	p.entries = append(p.entries, entry{
		path:    filepath.Join(parts...),
		format:  format,
		content: content,
	})
}

// Pour converts staged entries to domain.Materials whose paths are anchored
// at the Kind's output subdirectory. Called by the Planner after the
// casting's Forge returns.
func (p *Pourer) Pour() ([]domain.Material, error) {
	materials := make([]domain.Material, 0, len(p.entries))
	for _, e := range p.entries {
		mat, err := e.format.NewMaterial(e.content, filepath.Join(p.kindDir, e.path))
		if err != nil {
			return nil, err
		}
		materials = append(materials, mat)
	}
	return materials, nil
}
