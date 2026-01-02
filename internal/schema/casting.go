// Package schema provides access to embedded CUE schema files.
package schema

import "embed"

// Content holds the embedded CUE schema files.
//
//go:embed all:*
var Content embed.FS
