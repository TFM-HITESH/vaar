// Copyright © 2026 envaar
// SPDX-License-Identifier: Apache-2.0

// Package diff coordinates the diff application use case. It composes source
// loading, analysis conversion and key-inventory comparison without rendering
// output or deciding command exit codes.
package diff

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/envaar/vaar/internal/analysis"
	analysisdotenv "github.com/envaar/vaar/internal/analysis/dotenv"
	diffengine "github.com/envaar/vaar/internal/diff"
	sourcedotenv "github.com/envaar/vaar/internal/source/dotenv"
)

// Options identifies the two user-selected dotenv operands. The values are
// retained as display labels so relative, absolute and space-containing paths
// remain unchanged in diff results.
type Options struct {
	LeftPath  string
	RightPath string
}

// Dependencies provides the source-loading seam used by application tests.
// Analysis and diff remain concrete boundaries: the service must always build
// an analysis snapshot and invoke the analysis-backed diff engine.
type Dependencies struct {
	LoadDotenv func(paths, displayPaths []string) ([]sourcedotenv.Document, error)
}

// Service coordinates one diff execution.
type Service struct {
	loadDotenv func(paths, displayPaths []string) ([]sourcedotenv.Document, error)
}

// New constructs a diff application service with the standard dotenv loader.
func New() *Service {
	return NewWithDependencies(Dependencies{})
}

// NewWithDependencies constructs a diff application service with explicit
// source-loading dependencies for focused orchestration tests.
func NewWithDependencies(dependencies Dependencies) *Service {
	if dependencies.LoadDotenv == nil {
		dependencies.LoadDotenv = sourcedotenv.LoadMany
	}

	return &Service{loadDotenv: dependencies.LoadDotenv}
}

// Run loads both operands through the shared dotenv source boundary, converts
// them into a value-free analysis snapshot, and compares their key inventories.
func (s *Service) Run(ctx context.Context, opts Options) (diffengine.Result, error) {
	if err := ctx.Err(); err != nil {
		return diffengine.Result{}, err
	}

	paths := []string{opts.LeftPath, opts.RightPath}
	documents, err := s.loadDotenv(paths, paths)
	if err != nil {
		return diffengine.Result{}, normalizeLoadError(err)
	}
	if len(documents) != len(paths) {
		return diffengine.Result{}, fmt.Errorf(
			"load diff sources: got %d documents for %d selected paths",
			len(documents), len(paths),
		)
	}
	for i, document := range documents {
		if document.SourcePath != paths[i] {
			return diffengine.Result{}, fmt.Errorf(
				"load diff sources: document %d has source path %q, want %q",
				i, document.SourcePath, paths[i],
			)
		}
		if document.Path != paths[i] {
			return diffengine.Result{}, fmt.Errorf(
				"load diff sources: document %d has display path %q, want %q",
				i, document.Path, paths[i],
			)
		}
	}
	if err := ctx.Err(); err != nil {
		return diffengine.Result{}, err
	}

	inputs := make([]analysisdotenv.DocumentInput, len(documents))
	for i, document := range documents {
		inputs[i] = analysisdotenv.DocumentInput{
			ID:     analysis.DocumentID(paths[i]),
			Source: document,
		}
	}

	snapshot := analysis.NewSnapshot(analysisdotenv.FromDocuments(inputs))
	analysisDocuments := snapshot.Documents()
	leftInventory := analysis.NewKeyInventory(analysisDocuments[0])
	rightInventory := analysis.NewKeyInventory(analysisDocuments[1])

	return diffengine.CompareInventories(
		paths[0], leftInventory,
		paths[1], rightInventory,
	), nil
}

func normalizeLoadError(err error) error {
	var loadErr *sourcedotenv.LoadError
	if !errors.As(err, &loadErr) {
		return err
	}

	switch {
	case errors.Is(loadErr.Err, sourcedotenv.ErrIsDirectory):
		return fmt.Errorf("%s is a directory, expected a dotenv file", loadErr.Path)
	case errors.Is(loadErr.Err, sourcedotenv.ErrNotRegularFile):
		return fmt.Errorf("%s is not a regular file, expected a dotenv file", loadErr.Path)
	case errors.Is(loadErr.Err, os.ErrNotExist):
		return fmt.Errorf("reading %s: file does not exist", loadErr.Path)
	}

	var pathErr *os.PathError
	if errors.As(loadErr.Err, &pathErr) {
		return fmt.Errorf("reading %s: %w", loadErr.Path, pathErr)
	}
	return fmt.Errorf("reading %s: %w", loadErr.Path, loadErr.Err)
}
