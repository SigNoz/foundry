package yaml

import (
	"embed"

	goyaml "gopkg.in/yaml.v3"
)

func MustFile(fs embed.FS, name string) string {
	data, err := fs.ReadFile(name)
	if err != nil {
		panic(err)
	}

	return string(data)
}

func MustMarshal(v any) string {
	yaml, err := goyaml.Marshal(v)
	if err != nil {
		panic(err)
	}

	return string(yaml)
}
