package types

var (
	FormatYAML Format = Format{s: "yaml"}
	FormatJSON Format = Format{s: "json"}
)

type Format struct{ s string }
