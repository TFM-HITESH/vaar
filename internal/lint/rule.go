/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

// Package lint defines analysis-backed dotenv rules and their deterministic
// execution engine. Scope, mutation, output and command orchestration live in
// their respective boundary packages.
package lint

import "github.com/envaar/vaar/internal/analysis"

// Context gives each rule read-only access to the immutable analysis snapshot.
// Scope, filesystem, command-line and mutation concerns are intentionally not
// part of the rule-facing context.
type Context struct {
	// Snapshot contains the ordered, value-free documents being linted.
	Snapshot analysis.Snapshot
}

// Rule describes one lint check that can evaluate the current Context without
// mutating shared state. FixableRule remains a separate byte-transform
// contract consumed by the mutation planner.
type Rule interface {
	ID() string
	Description() string
	Run(Context) ([]Finding, error)
}
