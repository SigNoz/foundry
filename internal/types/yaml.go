package types

import (
	goyaml "gopkg.in/yaml.v3"
)

func MustMarshalYAML(v any) string {
	yaml, err := goyaml.Marshal(v)
	if err != nil {
		panic(err)
	}

	return string(yaml)
}
