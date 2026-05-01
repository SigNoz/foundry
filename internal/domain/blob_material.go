package domain

var _ Material = BlobMaterial{}

type BlobMaterial struct {
	contents []byte
	path     string
}

func NewBlobMaterial(contents []byte, path string) BlobMaterial {
	return BlobMaterial{
		contents: contents,
		path:     path,
	}
}

func (m BlobMaterial) Path() string {
	return m.path
}

func (m BlobMaterial) FmtContents() []byte {
	return m.contents
}
