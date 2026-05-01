// Package domain contains Foundry's shared domain primitives.
//
// This package is for value objects and pure helpers that are used across
// configuration loading, molding, casting, patching, and writing. Examples
// include structured materials, opaque blob materials, template wrappers,
// addresses, formats, pointer helpers, and serialization helpers.
//
// Keep this package free of orchestration and side effects. It must not decide
// which pipeline phase runs, select deployment implementations, execute tools,
// write files, read user configuration, or contain platform-specific rendering
// behavior. Put that logic in the package that owns the behavior instead.
//
// The only Foundry package this package may import is internal/errors. Do not
// import api/v1alpha1 or any internal orchestration, casting, molding,
// infrastructure, writer, tooler, config, or ledger package from here.
package domain
