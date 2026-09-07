/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

// Package mutations owns the bounded plan-and-apply pipeline for safe lint
// fixes. Planning consumes already-loaded source documents and selected rule
// fix functions without reading or writing the filesystem. Application
// validates every planned original state before replacing any destination,
// coordinates destination writers through the shared filesystem lock, preserves
// captured permission bits through atomic replacement, and cleans temporary
// files.
//
// Mutation state may contain source bytes because it belongs to the mutation
// boundary, not the value-free analysis boundary. This package does not own
// rule semantics, analysis facts, CLI behavior, output rendering, exit-code
// mapping, persistent backups, or multi-file rollback.
package mutations
