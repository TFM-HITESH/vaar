// Copyright © 2026 envaar
// SPDX-License-Identifier: Apache-2.0

// Package lint coordinates the lint application use case. It composes scope
// resolution, dotenv loading, analysis conversion, rule execution and the
// optional mutation lifecycle, but does not render output or map exit codes.
package lint

import (
	"context"
	"fmt"

	"github.com/envaar/vaar/internal/analysis"
	analysisdotenv "github.com/envaar/vaar/internal/analysis/dotenv"
	lintengine "github.com/envaar/vaar/internal/lint"
	"github.com/envaar/vaar/internal/mutations"
	"github.com/envaar/vaar/internal/scope"
	sourcedotenv "github.com/envaar/vaar/internal/source/dotenv"
)

// Options selects the repository inputs, rules and optional safe-fix lifecycle
// for one lint run. Output and exit-code options deliberately do not belong
// here.
type Options struct {
	Root      string
	Target    string
	TargetDir string
	OnlyRules []string
	SkipRules []string
	Fix       bool
}

// Dependencies provides narrow seams for the application boundary. Production
// callers should use New, which supplies the repository scope resolver, dotenv
// loader and mutation planner. The seams let tests prove lifecycle behavior
// without introducing source interfaces into analysis.
type Dependencies struct {
	ResolveScope func(scope.Options) (scope.Selection, error)
	LoadDotenv   func(paths, displayPaths []string) ([]sourcedotenv.Document, error)
	BuildPlan    func([]sourcedotenv.Document, []lintengine.Rule) (mutations.Plan, error)
	ApplyPlan    func(mutations.Plan) error
}

// Service is the lint application service.
type Service struct {
	engine       *lintengine.Engine
	resolveScope func(scope.Options) (scope.Selection, error)
	loadDotenv   func(paths, displayPaths []string) ([]sourcedotenv.Document, error)
	buildPlan    func([]sourcedotenv.Document, []lintengine.Rule) (mutations.Plan, error)
	applyPlan    func(mutations.Plan) error
}

// Result contains findings and the loaded state produced by one application
// run. Reporters should consume Findings only; LoadedDocuments retain
// source-owned values for internal fix planning and are not output data.
type Result struct {
	Findings        []lintengine.Finding
	Selection       scope.Selection
	LoadedDocuments []sourcedotenv.Document
	Snapshot        analysis.Snapshot
	SelectedRules   []lintengine.Rule
	Changed         bool
}

// HasUnfixedFindings reports whether the final lint snapshot still contains a
// finding that was not repaired by the optional fix lifecycle.
func (r Result) HasUnfixedFindings() bool {
	for _, finding := range r.Findings {
		if !finding.Fixed {
			return true
		}
	}
	return false
}

// New constructs a lint application service with the standard scope resolver,
// dotenv source loader and mutation planner.
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
	if dependencies.BuildPlan == nil {
		dependencies.BuildPlan = mutations.BuildPlan
	}
	if dependencies.ApplyPlan == nil {
		dependencies.ApplyPlan = func(plan mutations.Plan) error {
			return plan.Apply()
		}
	}

	return &Service{
		engine:       lintengine.NewEngine(rules...),
		resolveScope: dependencies.ResolveScope,
		loadDotenv:   dependencies.LoadDotenv,
		buildPlan:    dependencies.BuildPlan,
		applyPlan:    dependencies.ApplyPlan,
	}
}

// Run resolves the requested scope once and executes the selected lint rules.
// With Fix enabled, it plans and applies safe changes, reloads the same scope,
// and reruns the same selected rule plan before returning the final result.
func (s *Service) Run(ctx context.Context, opts Options) (Result, error) {
	plan, err := s.engine.SelectRulePlan(lintengine.EngineOptions{
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

	return s.runWithSelection(ctx, opts, selection, plan)
}

// RunWithSelection executes the configured rules against a pre-resolved scope
// selection. Callers that already resolved scope, such as the CLI preflight for
// output-path validation, can reuse the same selection without a second scope
// walk.
func (s *Service) RunWithSelection(ctx context.Context, opts Options, selection scope.Selection) (Result, error) {
	plan, err := s.engine.SelectRulePlan(lintengine.EngineOptions{
		OnlyRules: opts.OnlyRules,
		SkipRules: opts.SkipRules,
	})
	if err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	return s.runWithSelection(ctx, opts, selection, plan)
}

func (s *Service) runWithSelection(ctx context.Context, opts Options, selection scope.Selection, plan lintengine.RulePlan) (Result, error) {
	selected := plan.Rules()
	documents, snapshot, findings, err := s.loadAndRun(ctx, selection, plan)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		Findings:        findings,
		Selection:       selection,
		LoadedDocuments: documents,
		Snapshot:        snapshot,
		SelectedRules:   selected,
	}
	if !opts.Fix {
		return result, nil
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	mutationPlan, err := s.buildPlan(documents, selected)
	if err != nil {
		return Result{}, fmt.Errorf("plan lint fixes: %w", err)
	}
	if changes := mutationPlan.Changes(); len(changes) == 0 {
		return result, nil
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := s.applyPlan(mutationPlan); err != nil {
		return Result{}, fmt.Errorf("apply lint fixes: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	updatedDocuments, updatedSnapshot, remaining, err := s.loadAndRun(ctx, selection, plan)
	if err != nil {
		return Result{}, err
	}

	findings = markFixedFindings(findings, remaining)
	lintengine.SortFindings(findings)
	result.Findings = findings
	result.LoadedDocuments = updatedDocuments
	result.Snapshot = updatedSnapshot
	result.Changed = true
	return result, nil
}

func (s *Service) loadAndRun(ctx context.Context, selection scope.Selection, plan lintengine.RulePlan) ([]sourcedotenv.Document, analysis.Snapshot, []lintengine.Finding, error) {
	displayPaths := make([]string, len(selection.Paths))
	for i, path := range selection.Paths {
		displayPaths[i] = selection.DisplayPath(path)
	}

	documents, err := s.loadDotenv(selection.Paths, displayPaths)
	if err != nil {
		return nil, analysis.Snapshot{}, nil, fmt.Errorf("load lint sources: %w", err)
	}
	if len(documents) != len(selection.Paths) {
		return nil, analysis.Snapshot{}, nil, fmt.Errorf(
			"load lint sources: got %d documents for %d selected paths",
			len(documents), len(selection.Paths),
		)
	}
	for i, document := range documents {
		if document.SourcePath != selection.Paths[i] {
			return nil, analysis.Snapshot{}, nil, fmt.Errorf(
				"load lint sources: document %d has source path %q, want %q",
				i, document.SourcePath, selection.Paths[i],
			)
		}
		if document.Path != displayPaths[i] {
			return nil, analysis.Snapshot{}, nil, fmt.Errorf(
				"load lint sources: document %d has display path %q, want %q",
				i, document.Path, displayPaths[i],
			)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, analysis.Snapshot{}, nil, err
	}

	inputs := make([]analysisdotenv.DocumentInput, len(documents))
	for i, document := range documents {
		inputs[i] = analysisdotenv.DocumentInput{
			ID:     analysis.DocumentID(selection.Paths[i]),
			Source: document,
		}
	}

	snapshot := analysis.NewSnapshot(analysisdotenv.FromDocuments(inputs))
	findings, err := s.engine.RunPlan(ctx, snapshot, plan)
	if err != nil {
		return nil, analysis.Snapshot{}, nil, fmt.Errorf("run lint engine: %w", err)
	}

	return documents, snapshot, findings, nil
}
