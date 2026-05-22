// Package planner defines the per-Kind orchestrator contract that Foundry
// iterates against.
//
// Each Kind (Installation, CollectionAgent, ...) ships a Planner implementation
// in its own casting package. Foundry resolves a Kind to a Planner at runtime
// via the dispatch map on the Foundry struct and then drives the same set of
// verbs (Machinery, Patches, Toolers, MoldingKinds, EnrichStatus, Mold,
// MergeStatusIntoSpec, Forge, Cast) regardless of which Kind it is.
//
// Keep this package free of orchestration and side effects. It must not
// construct planners, decide which Kind runs, dispatch by Kind, execute
// tools, write files, or hold registries. The dispatch and lifecycle live
// in internal/foundry; the implementations live in internal/casting/<kind>.
//
// This package is a leaf: it is imported by both internal/foundry and every
// internal/casting/<kind> package, so it must not import any of them. The
// only Foundry packages this package may import are api/v1alpha1,
// internal/domain, and internal/tooler.
package planner
