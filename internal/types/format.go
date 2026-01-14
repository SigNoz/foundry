package types

var (
	FormatYAML Format = Format{s: "yaml"}
	FormatJSON Format = Format{s: "json"}
	FormatINI  Format = Format{s: "ini"}
)

type Format struct{ s string }
