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
	"github.com/envaar/vaar/internal/mutations"
	"github.com/envaar/vaar/internal/scope"
	sourcedotenv "github.com/envaar/vaar/internal/source/dotenv"
)

type applicationRule struct {
	id       string
	idCalls  *int
	calls    *int
	findings []lintmodel.Finding
	err      error
}

func (r applicationRule) ID() string {
	if r.idCalls != nil {
		(*r.idCalls)++
	}
	return r.id
}
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

func TestServiceDoesNotReselectRulesDuringExecution(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	mustWrite(t, path, "KEY=value\n")

	idCalls := 0
	callsAtLoad := -1
	service := applicationlint.NewWithDependencies(applicationlint.Dependencies{
		LoadDotenv: func(paths, displayPaths []string) ([]sourcedotenv.Document, error) {
			callsAtLoad = idCalls
			return sourcedotenv.LoadMany(paths, displayPaths)
		},
	}, applicationRule{id: "rule", idCalls: &idCalls})

	result, err := service.Run(context.Background(), applicationlint.Options{Root: root})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if callsAtLoad < 0 {
		t.Fatal("loader did not observe rule selection")
	}
	if idCalls != callsAtLoad {
		t.Fatalf("rule ID calls after loading = %d, want %d", idCalls, callsAtLoad)
	}
	if len(result.SelectedRules) != 1 {
		t.Fatalf("selected rules = %#v, want one rule", result.SelectedRules)
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

func TestServiceRejectsLoadedDocumentsWithMismatchedProvenance(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, ".env.first")
	secondPath := filepath.Join(root, ".env.second")
	mustWrite(t, firstPath, "FIRST=value\n")
	mustWrite(t, secondPath, "SECOND=value\n")

	cases := []struct {
		name   string
		mutate func([]sourcedotenv.Document)
		want   string
	}{
		{
			name: "swapped documents",
			mutate: func(documents []sourcedotenv.Document) {
				documents[0], documents[1] = documents[1], documents[0]
			},
			want: "source path",
		},
		{
			name: "mismatched source path",
			mutate: func(documents []sourcedotenv.Document) {
				documents[0].SourcePath = filepath.Join(root, "unexpected.env")
			},
			want: "source path",
		},
		{
			name: "mismatched display path",
			mutate: func(documents []sourcedotenv.Document) {
				documents[0].Path = "unexpected.env"
			},
			want: "display path",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ruleCalls := 0
			service := applicationlint.NewWithDependencies(applicationlint.Dependencies{
				ResolveScope: func(scope.Options) (scope.Selection, error) {
					return scope.Selection{
						Root:  root,
						Paths: []string{firstPath, secondPath},
					}, nil
				},
				LoadDotenv: func(paths, displayPaths []string) ([]sourcedotenv.Document, error) {
					documents, err := sourcedotenv.LoadMany(paths, displayPaths)
					if err != nil {
						return nil, err
					}
					tc.mutate(documents)
					return documents, nil
				},
			}, applicationRule{id: "rule", calls: &ruleCalls})

			_, err := service.Run(context.Background(), applicationlint.Options{})
			if err == nil {
				t.Fatal("expected provenance validation error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
			if ruleCalls != 0 {
				t.Fatalf("rule calls = %d, want 0", ruleCalls)
			}
		})
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

func TestServiceDefaultDiscoveryLintsArbitraryDotenvSuffix(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env.preview-local")
	mustWrite(t, path, "KEY=first\nKEY=second\n")

	result, err := applicationlint.New(rules.NewDuplicateKey()).Run(
		context.Background(),
		applicationlint.Options{Root: root},
	)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if got, want := len(result.LoadedDocuments), 1; got != want {
		t.Fatalf("loaded document count = %d, want %d", got, want)
	}
	if got, want := result.LoadedDocuments[0].Path, ".env.preview-local"; got != want {
		t.Fatalf("display path = %q, want %q", got, want)
	}
	if got, want := len(result.Findings), 1; got != want {
		t.Fatalf("finding count = %d, want %d", got, want)
	}
	if got, want := result.Findings[0].File, ".env.preview-local"; got != want {
		t.Fatalf("finding file = %q, want %q", got, want)
	}
}

func TestServiceFixReanalyzesAndMarksFixedFindings(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	mustWrite(t, path, "TRAIL=one  \n")

	ruleCalls := 0
	service := applicationlint.New(
		applicationRule{id: "pass-counter", calls: &ruleCalls},
		rules.NewTrailingWhitespace(),
	)
	result, err := service.Run(context.Background(), applicationlint.Options{
		Root:      root,
		Fix:       true,
		OnlyRules: []string{"pass-counter", "trailing-whitespace"},
	})
	if err != nil {
		t.Fatalf("fix run failed: %v", err)
	}

	if got, want := ruleCalls, 2; got != want {
		t.Fatalf("rule calls = %d, want %d after reanalysis", got, want)
	}
	if !result.Changed {
		t.Fatal("expected fix run to report a change")
	}
	if len(result.Findings) != 1 || !result.Findings[0].Fixed {
		t.Fatalf("findings = %#v, want one fixed finding", result.Findings)
	}
	if got, want := string(readFile(t, path)), "TRAIL=one\n"; got != want {
		t.Fatalf("fixed content = %q, want %q", got, want)
	}
}

func TestServiceFixWithoutChangesSkipsReanalysis(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	mustWrite(t, path, "KEY=one\nKEY=two\n")

	ruleCalls := 0
	service := applicationlint.New(
		applicationRule{id: "pass-counter", calls: &ruleCalls},
		rules.NewDuplicateKey(),
	)
	result, err := service.Run(context.Background(), applicationlint.Options{
		Root:      root,
		Fix:       true,
		OnlyRules: []string{"pass-counter", "duplicate-key"},
	})
	if err != nil {
		t.Fatalf("fix run failed: %v", err)
	}

	if got, want := ruleCalls, 1; got != want {
		t.Fatalf("rule calls = %d, want %d without reanalysis", got, want)
	}
	if result.Changed {
		t.Fatal("expected no-change fix run to report no change")
	}
	if len(result.Findings) != 1 || result.Findings[0].Fixed {
		t.Fatalf("findings = %#v, want one remaining finding", result.Findings)
	}
}

func TestServiceFixPropagatesMutationFailure(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	original := "TRAIL=one  \n"
	mustWrite(t, path, original)
	applyErr := errors.New("mutation failed")

	service := applicationlint.NewWithDependencies(applicationlint.Dependencies{
		ApplyPlan: func(mutations.Plan) error { return applyErr },
	}, rules.NewTrailingWhitespace())
	_, err := service.Run(context.Background(), applicationlint.Options{Root: root, Fix: true})
	if !errors.Is(err, applyErr) {
		t.Fatalf("error = %v, want mutation error", err)
	}
	if got, want := string(readFile(t, path)), original; got != want {
		t.Fatalf("content after failed mutation = %q, want %q", got, want)
	}
}

func TestServiceFixPropagatesReloadFailure(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	mustWrite(t, path, "TRAIL=one  \n")
	reloadErr := errors.New("reload failed")
	loadCalls := 0

	service := applicationlint.NewWithDependencies(applicationlint.Dependencies{
		LoadDotenv: func(paths, displayPaths []string) ([]sourcedotenv.Document, error) {
			loadCalls++
			if loadCalls == 2 {
				return nil, reloadErr
			}
			return sourcedotenv.LoadMany(paths, displayPaths)
		},
	}, rules.NewTrailingWhitespace())
	_, err := service.Run(context.Background(), applicationlint.Options{Root: root, Fix: true})
	if !errors.Is(err, reloadErr) {
		t.Fatalf("error = %v, want reload error", err)
	}
	if got, want := loadCalls, 2; got != want {
		t.Fatalf("load calls = %d, want %d", got, want)
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

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s failed: %v", path, err)
	}
	return data
}
