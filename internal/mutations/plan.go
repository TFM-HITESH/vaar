/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package mutations

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/envaar/vaar/internal/fs"
	"github.com/envaar/vaar/internal/lint"
	sourcedotenv "github.com/envaar/vaar/internal/source/dotenv"
)

// Change describes one planned replacement. Original and Replacement are
// owned by the plan and are never exposed through the plan's backing storage.
// Mode is the permission state captured when the source document was loaded.
type Change struct {
	SourcePath  string
	DisplayPath string
	Original    []byte
	Replacement []byte
	Mode        os.FileMode
}

// Plan is an immutable, ordered collection of file replacements. A plan is
// created from already-loaded source documents and performs no filesystem I/O
// until Apply is called.
type Plan struct {
	changes []Change
}

// BuildPlan composes the selected rule fixes over each loaded document and
// returns only files whose replacement bytes differ from their original bytes.
// Documents remain in caller-provided order, and all planned byte slices are
// copied into plan-owned storage.
func BuildPlan(documents []sourcedotenv.Document, rules []lint.Rule) (Plan, error) {
	changes := make([]Change, 0, len(documents))
	seen := make(map[string]struct{}, len(documents))

	for _, document := range documents {
		if document.SourcePath == "" {
			return Plan{}, fmt.Errorf("plan lint fix: document source path is empty")
		}
		if _, exists := seen[document.SourcePath]; exists {
			return Plan{}, fmt.Errorf("plan lint fix: duplicate source path %q", document.SourcePath)
		}
		seen[document.SourcePath] = struct{}{}

		original := cloneBytes(document.Original)
		replacement := lint.FixData(rules, cloneBytes(original))
		if bytes.Equal(replacement, original) {
			continue
		}

		changes = append(changes, Change{
			SourcePath:  document.SourcePath,
			DisplayPath: document.Path,
			Original:    original,
			Replacement: cloneBytes(replacement),
			Mode:        document.Mode.Perm(),
		})
	}

	return Plan{changes: changes}, nil
}

// Changes returns a deep copy of the ordered planned replacements. An empty
// plan returns a non-nil empty slice.
func (p Plan) Changes() []Change {
	changes := make([]Change, len(p.changes))
	for i, change := range p.changes {
		changes[i] = Change{
			SourcePath:  change.SourcePath,
			DisplayPath: change.DisplayPath,
			Original:    cloneBytes(change.Original),
			Replacement: cloneBytes(change.Replacement),
			Mode:        change.Mode,
		}
	}
	return changes
}

// Apply resolves source paths through symlinks, validates every planned
// destination before creating any replacement file, and then revalidates each
// destination immediately before replacing it. Replacements occur in plan
// order using same-directory atomic files, with each captured permission mode
// applied to its temporary file before finalization. It intentionally does not
// roll back earlier successful replacements if a later replacement fails.
func (p Plan) Apply() error {
	for _, change := range p.changes {
		if err := validateChange(change); err != nil {
			return fmt.Errorf("validate mutation %q: %w", changeLabel(change), err)
		}
	}

	for _, change := range p.changes {
		destination, err := resolveDestination(change.SourcePath)
		if err != nil {
			return fmt.Errorf("validate mutation %q: %w", changeLabel(change), err)
		}
		if err := validateChangeAt(change, destination); err != nil {
			return fmt.Errorf("validate mutation %q: %w", changeLabel(change), err)
		}
		if err := applyChange(change, destination); err != nil {
			return fmt.Errorf("apply mutation %q: %w", changeLabel(change), err)
		}
	}

	return nil
}

func validateChange(change Change) error {
	destination, err := resolveDestination(change.SourcePath)
	if err != nil {
		return err
	}
	return validateChangeAt(change, destination)
}

func validateChangeAt(change Change, destination string) error {
	current, err := fs.ReadFile(destination)
	if err != nil {
		return fmt.Errorf("read destination: %w", err)
	}
	if !bytes.Equal(current, change.Original) {
		return fmt.Errorf("destination changed since planning")
	}

	info, err := os.Stat(destination)
	if err != nil {
		return fmt.Errorf("stat destination: %w", err)
	}
	if got, want := info.Mode().Perm(), change.Mode.Perm(); got != want {
		return fmt.Errorf("destination permissions changed since planning: got %o, want %o", got, want)
	}
	return nil
}

func resolveDestination(sourcePath string) (string, error) {
	destination, err := fs.CanonicalPath(sourcePath)
	if err != nil {
		return "", fmt.Errorf("resolve destination: %w", err)
	}
	return destination, nil
}

func applyChange(change Change, destination string) (err error) {
	file, err := fs.NewAtomicFile(destination)
	if err != nil {
		return fmt.Errorf("create atomic replacement: %w", err)
	}
	defer func() {
		if cleanupErr := file.Cleanup(); cleanupErr != nil {
			err = errors.Join(err, fmt.Errorf("cleanup temporary replacement: %w", cleanupErr))
		}
	}()

	if _, err := file.Write(change.Replacement); err != nil {
		return fmt.Errorf("write replacement: %w", err)
	}
	if err := file.Chmod(change.Mode); err != nil {
		return fmt.Errorf("preserve permissions: %w", err)
	}
	if err := file.Finalize(); err != nil {
		return fmt.Errorf("finalize replacement: %w", err)
	}
	return nil
}

func changeLabel(change Change) string {
	if change.DisplayPath != "" {
		return change.DisplayPath
	}
	return change.SourcePath
}

func cloneBytes(data []byte) []byte {
	if data == nil {
		return nil
	}
	cloned := make([]byte, len(data))
	copy(cloned, data)
	return cloned
}
