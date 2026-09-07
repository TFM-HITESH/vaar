// Copyright © 2026 envaar
// SPDX-License-Identifier: Apache-2.0

// Package lint coordinates one read-only lint use case. It composes scope
// resolution, dotenv loading, analysis conversion and the analysis-backed lint
// engine, but does not render output, map exit codes or mutate files.
package lint

import (
	"context"
	"fmt"

	"github.com/envaar/vaar/internal/analysis"
	analysisdotenv "github.com/envaar/vaar/internal/analysis/dotenv"
	lintengine "github.com/envaar/vaar/internal/lint"
	"github.com/envaar/vaar/internal/scope"
	sourcedotenv "github.com/envaar/vaar/internal/source/dotenv"
)

// Options selects the repository inputs and rules for one read-only lint run.
// Output, exit-code and mutation options deliberately do not belong here.
type Options struct {
	Root      string
	Target    string
	TargetDir string
	OnlyRules []string
	SkipRules []string
}

// Dependencies provides narrow seams for the application boundary. Production
// callers should use New, which supplies the repository scope resolver and
// dotenv loader. The seams let tests prove that orchestration resolves and
// loads exactly once without introducing source interfaces into analysis.
type Dependencies struct {
	ResolveScope func(scope.Options) (scope.Selection, error)
	LoadDotenv   func(paths, displayPaths []string) ([]sourcedotenv.Document, error)
}

// Service is the read-only lint application service.
type Service struct {
	engine       *lintengine.Engine
	resolveScope func(scope.Options) (scope.Selection, error)
	loadDotenv   func(paths, displayPaths []string) ([]sourcedotenv.Document, error)
}

// Result contains the findings and the loaded state needed by the later
// mutation-planning flow. Reporters should consume Findings only;
// LoadedDocuments retain source-owned values for internal fix planning and are
// not output data.
type Result struct {
	Findings        []lintengine.Finding
	Selection       scope.Selection
	LoadedDocuments []sourcedotenv.Document
	Snapshot        analysis.Snapshot
	SelectedRules   []lintengine.Rule
}

// New constructs a read-only lint application service with the standard scope
// resolver and dotenv source loader.
func New(rules ...lintengine.Rule) *Service {
	return NewWithDependencies(Dependencies{}, rules...)
}

// NewWithDependencies constructs a service with explicit boundary functions.
// Nil dependencies use the production implementations.
func NewWithDependencies(dependencies Dependencies, rules ...lintengine.Rule) *Service {
	if dependencies.ResolveScope == nil {
		dependencies.ResolveScope = scope.Resolve
	}
	if dependencies.LoadDotenv == nil {
		dependencies.LoadDotenv = sourcedotenv.LoadMany
	}

	return &Service{
		engine:       lintengine.NewEngine(rules...),
		resolveScope: dependencies.ResolveScope,
		loadDotenv:   dependencies.LoadDotenv,
	}
}

// Run resolves the requested scope once, loads each selected dotenv source
// once, converts the loaded documents into one analysis snapshot, and runs the
// selected lint rules. It performs no writes, rendering or exit-code mapping.
func (s *Service) Run(ctx context.Context, opts Options) (Result, error) {
	selected, err := s.engine.SelectRules(lintengine.EngineOptions{
		OnlyRules: opts.OnlyRules,
		SkipRules: opts.SkipRules,
	})
	if err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	selection, err := s.resolveScope(scope.Options{
		Root:      opts.Root,
		Target:    opts.Target,
		TargetDir: opts.TargetDir,
	})
	if err != nil {
		return Result{}, fmt.Errorf("resolve lint scope: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	displayPaths := make([]string, len(selection.Paths))
	for i, path := range selection.Paths {
		displayPaths[i] = selection.DisplayPath(path)
	}

	documents, err := s.loadDotenv(selection.Paths, displayPaths)
	if err != nil {
		return Result{}, fmt.Errorf("load lint sources: %w", err)
	}
	if len(documents) != len(selection.Paths) {
		return Result{}, fmt.Errorf(
			"load lint sources: got %d documents for %d selected paths",
			len(documents), len(selection.Paths),
		)
	}
	for i, document := range documents {
		if document.SourcePath != selection.Paths[i] {
			return Result{}, fmt.Errorf(
				"load lint sources: document %d has source path %q, want %q",
				i, document.SourcePath, selection.Paths[i],
			)
		}
		if document.Path != displayPaths[i] {
			return Result{}, fmt.Errorf(
				"load lint sources: document %d has display path %q, want %q",
				i, document.Path, displayPaths[i],
			)
		}
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	inputs := make([]analysisdotenv.DocumentInput, len(documents))
	for i, document := range documents {
		inputs[i] = analysisdotenv.DocumentInput{
			ID:     analysis.DocumentID(selection.Paths[i]),
			Source: document,
		}
	}

	snapshot := analysis.NewSnapshot(analysisdotenv.FromDocuments(inputs))
	findings, err := s.engine.Run(ctx, snapshot, lintengine.EngineOptions{
		OnlyRules: opts.OnlyRules,
		SkipRules: opts.SkipRules,
	})
	if err != nil {
		return Result{}, fmt.Errorf("run lint engine: %w", err)
	}

	return Result{
		Findings:        findings,
		Selection:       selection,
		LoadedDocuments: documents,
		Snapshot:        snapshot,
		SelectedRules:   selected,
	}, nil
}
