// Copyright © 2026 envaar
// SPDX-License-Identifier: Apache-2.0

package lint_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	applicationlint "github.com/envaar/vaar/internal/application/lint"
	lintmodel "github.com/envaar/vaar/internal/lint"
	"github.com/envaar/vaar/internal/lint/rules"
	"github.com/envaar/vaar/internal/scope"
	sourcedotenv "github.com/envaar/vaar/internal/source/dotenv"
)

type applicationRule struct {
	id       string
	calls    *int
	findings []lintmodel.Finding
	err      error
}

func (r applicationRule) ID() string          { return r.id }
func (r applicationRule) Description() string { return "application test rule" }

func (r applicationRule) Run(lintmodel.Context) ([]lintmodel.Finding, error) {
	if r.calls != nil {
		(*r.calls)++
	}
	return r.findings, r.err
}

func TestServiceCleanRunReturnsLoadedAnalysisState(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	mustWrite(t, path, "KEY=value\n")

	service := applicationlint.New(rules.NewDuplicateKey())
	result, err := service.Run(context.Background(), applicationlint.Options{Root: root})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if result.Findings == nil {
		t.Fatal("findings must be non-nil for a clean run")
	}
	if len(result.Findings) != 0 {
		t.Fatalf("findings = %#v, want none", result.Findings)
	}
	if got, want := len(result.LoadedDocuments), 1; got != want {
		t.Fatalf("loaded document count = %d, want %d", got, want)
	}
	if got, want := len(result.Snapshot.Documents()), 1; got != want {
		t.Fatalf("analysis document count = %d, want %d", got, want)
	}
	if got, want := result.LoadedDocuments[0].Path, ".env"; got != want {
		t.Fatalf("display path = %q, want %q", got, want)
	}
	if got, want := len(result.SelectedRules), 1; got != want {
		t.Fatalf("selected rule count = %d, want %d", got, want)
	}
}

func TestServiceReturnsFindingsFromAnalysisEngine(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	mustWrite(t, path, "KEY=first\nKEY=second\n")

	service := applicationlint.New(rules.NewDuplicateKey())
	result, err := service.Run(context.Background(), applicationlint.Options{Root: root})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if got, want := len(result.Findings), 1; got != want {
		t.Fatalf("finding count = %d, want %d", got, want)
	}
	finding := result.Findings[0]
	if finding.Rule != "duplicate-key" || finding.File != ".env" || finding.Line != 2 {
		t.Fatalf("unexpected finding: %#v", finding)
	}
}

func TestServicePreservesOnlyAndSkipRuleSelection(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	mustWrite(t, path, "KEY=value\n")

	firstCalls, secondCalls := 0, 0
	service := applicationlint.New(
		applicationRule{id: "first", calls: &firstCalls},
		applicationRule{id: "second", calls: &secondCalls},
	)

	_, err := service.Run(context.Background(), applicationlint.Options{
		Root:      root,
		OnlyRules: []string{"first", "second"},
		SkipRules: []string{"second"},
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if firstCalls != 1 || secondCalls != 0 {
		t.Fatalf("rule calls = (%d, %d), want (1, 0)", firstCalls, secondCalls)
	}
}

func TestServicePreservesDefaultTargetAndTargetDirectoryScopes(t *testing.T) {
	root := t.TempDir()
	rootFile := filepath.Join(root, ".env")
	nestedFile := filepath.Join(root, "service", ".env.local")
	mustWrite(t, rootFile, "ROOT=value\n")
	mustWrite(t, nestedFile, "NESTED=value\n")

	service := applicationlint.New()
	cases := []struct {
		name        string
		options     applicationlint.Options
		wantPaths   []string
		wantDisplay []string
	}{
		{
			name:        "default",
			options:     applicationlint.Options{Root: root},
			wantPaths:   []string{rootFile, nestedFile},
			wantDisplay: []string{".env", filepath.Join("service", ".env.local")},
		},
		{
			name:        "target file",
			options:     applicationlint.Options{Root: root, Target: ".env"},
			wantPaths:   []string{rootFile},
			wantDisplay: []string{".env"},
		},
		{
			name:        "target directory",
			options:     applicationlint.Options{Root: root, TargetDir: "service"},
			wantPaths:   []string{nestedFile},
			wantDisplay: []string{filepath.Join("service", ".env.local")},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.Run(context.Background(), tc.options)
			if err != nil {
				t.Fatalf("run failed: %v", err)
			}
			if got := result.Selection.Paths; !reflect.DeepEqual(got, tc.wantPaths) {
				t.Fatalf("selected paths = %#v, want %#v", got, tc.wantPaths)
			}
			gotDisplay := make([]string, len(result.LoadedDocuments))
			for i, document := range result.LoadedDocuments {
				gotDisplay[i] = document.Path
			}
			if !reflect.DeepEqual(gotDisplay, tc.wantDisplay) {
				t.Fatalf("display paths = %#v, want %#v", gotDisplay, tc.wantDisplay)
			}
		})
	}
}

func TestServiceSupportsEmptyDiscovery(t *testing.T) {
	service := applicationlint.New()
	result, err := service.Run(context.Background(), applicationlint.Options{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if result.LoadedDocuments == nil {
		t.Fatal("loaded documents must be non-nil for empty discovery")
	}
	if result.Findings == nil {
		t.Fatal("findings must be non-nil for empty discovery")
	}
	if len(result.LoadedDocuments) != 0 || len(result.Findings) != 0 {
		t.Fatalf("empty discovery result = %#v, want no documents or findings", result)
	}
}

func TestServiceResolvesScopeAndLoadsSelectedFilesOnce(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, ".env.first")
	secondPath := filepath.Join(root, ".env.second")
	mustWrite(t, firstPath, "FIRST=value\n")
	mustWrite(t, secondPath, "SECOND=value\n")

	resolveCalls := 0
	loadCalls := 0
	loadedPathCount := 0
	service := applicationlint.NewWithDependencies(applicationlint.Dependencies{
		ResolveScope: func(options scope.Options) (scope.Selection, error) {
			resolveCalls++
			return scope.Resolve(options)
		},
		LoadDotenv: func(paths, displayPaths []string) ([]sourcedotenv.Document, error) {
			loadCalls++
			loadedPathCount += len(paths)
			return sourcedotenv.LoadMany(paths, displayPaths)
		},
	}, rules.NewDuplicateKey())

	result, err := service.Run(context.Background(), applicationlint.Options{Root: root})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if resolveCalls != 1 {
		t.Fatalf("scope resolve calls = %d, want 1", resolveCalls)
	}
	if loadCalls != 1 {
		t.Fatalf("source load calls = %d, want 1", loadCalls)
	}
	if loadedPathCount != 2 {
		t.Fatalf("loaded path count = %d, want 2", loadedPathCount)
	}
	if got, want := len(result.LoadedDocuments), 2; got != want {
		t.Fatalf("loaded document count = %d, want %d", got, want)
	}
}

func TestServiceReturnsLoadingFailureWithoutRunningRules(t *testing.T) {
	root := t.TempDir()
	loadErr := errors.New("source load failed")
	ruleCalls := 0
	service := applicationlint.NewWithDependencies(applicationlint.Dependencies{
		ResolveScope: func(scope.Options) (scope.Selection, error) {
			return scope.Selection{Root: root}, nil
		},
		LoadDotenv: func([]string, []string) ([]sourcedotenv.Document, error) {
			return nil, loadErr
		},
	}, applicationRule{id: "rule", calls: &ruleCalls})

	_, err := service.Run(context.Background(), applicationlint.Options{Root: root})
	if !errors.Is(err, loadErr) {
		t.Fatalf("error = %v, want source load error", err)
	}
	if ruleCalls != 0 {
		t.Fatalf("rule calls = %d, want 0", ruleCalls)
	}
}

func TestServiceReturnsRuleExecutionFailure(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	mustWrite(t, path, "KEY=value\n")
	ruleErr := errors.New("rule failed")
	service := applicationlint.New(applicationRule{id: "broken-rule", err: ruleErr})

	_, err := service.Run(context.Background(), applicationlint.Options{Root: root})
	if !errors.Is(err, ruleErr) {
		t.Fatalf("error = %v, want rule error", err)
	}
	if !strings.Contains(err.Error(), "broken-rule") {
		t.Fatalf("error = %v, want rule ID", err)
	}
}

func TestServiceStopsBeforeScopeResolutionWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	resolveCalls := 0
	loadCalls := 0
	service := applicationlint.NewWithDependencies(applicationlint.Dependencies{
		ResolveScope: func(scope.Options) (scope.Selection, error) {
			resolveCalls++
			return scope.Selection{}, nil
		},
		LoadDotenv: func([]string, []string) ([]sourcedotenv.Document, error) {
			loadCalls++
			return nil, nil
		},
	}, applicationRule{id: "rule"})

	_, err := service.Run(ctx, applicationlint.Options{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
	if resolveCalls != 0 || loadCalls != 0 {
		t.Fatalf("dependency calls = (%d, %d), want (0, 0)", resolveCalls, loadCalls)
	}
}

func TestServiceValidatesRuleSelectionBeforeCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := applicationlint.New(applicationRule{id: "known-rule"})

	_, err := service.Run(ctx, applicationlint.Options{
		OnlyRules: []string{"unknown-rule"},
	})
	if err == nil {
		t.Fatal("expected rule-selection error")
	}
	if !strings.Contains(err.Error(), `unknown lint rule "unknown-rule"`) {
		t.Fatalf("error = %v, want unknown-rule selection error", err)
	}
}

func TestServiceConvertsDocumentsThroughAnalysisAdapter(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	mustWrite(t, path, "KEY=value with spaces\n")

	service := applicationlint.New()
	result, err := service.Run(context.Background(), applicationlint.Options{Root: root})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	documents := result.Snapshot.Documents()
	if len(documents) != 1 || len(documents[0].Lines) != 1 {
		t.Fatalf("unexpected snapshot: %#v", documents)
	}
	line := documents[0].Lines[0]
	if !line.ValueContainsWhitespace || line.Key != "KEY" || !line.HasAssignment {
		t.Fatalf("analysis line facts = %#v", line)
	}
	if line.Number != 1 || documents[0].DisplayPath != ".env" {
		t.Fatalf("analysis provenance = %#v", documents[0])
	}
}

func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent directory failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s failed: %v", path, err)
	}
}
