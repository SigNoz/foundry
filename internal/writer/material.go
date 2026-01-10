package writer

type Material struct {
	contents []byte
	path     string
}

func NewMaterial(contents []byte, path string) Material {
	return Material{
		contents: contents,
		path:     path,
	}
}

func (m Material) Contents() []byte {
	return m.contents
}

func (m Material) Path() string {
	return m.path
}
