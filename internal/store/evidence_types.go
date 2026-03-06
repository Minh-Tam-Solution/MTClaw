package store

// AllArtifactTypes is the SSOT (Single Source of Truth) list of expected
// artifact types in a complete evidence chain.
// CTO-49: extracted from evidence/chain.go to single source of truth.
// All consumers (chain builder, gate matrix, API responses) import from here.
var AllArtifactTypes = []string{"spec", "pr_gate", "test_run", "deploy"}
