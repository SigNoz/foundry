package types

import "embed"

func MustNewFileFromFS(fs embed.FS, path string) []byte {
	contents, err := fs.ReadFile(path)
	if err != nil {
		panic(err)
	}

	return contents
}
