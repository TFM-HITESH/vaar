/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package lint_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	model "github.com/envaar/vaar/internal/lint"
	lintoutput "github.com/envaar/vaar/internal/output/lint"
)

func TestTextReturnsEmptyForNoFindings(t *testing.T) {
	if got := lintoutput.Text(nil); got != "" {
		t.Fatalf("unexpected text output for no findings: got %q", got)
	}
}

func TestLintCLIUsesOutputLintPackage(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate the output package test file")
	}

	repositoryRoot := filepath.Join(filepath.Dir(testFile), "..", "..", "..")
	cliPath := filepath.Join(repositoryRoot, "internal", "cli", "lint.go")
	file, err := parser.ParseFile(token.NewFileSet(), cliPath, nil, 0)
	if err != nil {
		t.Fatalf("parse lint CLI imports: %v", err)
	}

	imports := make(map[string]bool, len(file.Imports))
	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			t.Fatalf("unquote lint CLI import %q: %v", imported.Path.Value, err)
		}
		imports[path] = true
	}

	const outputLintPath = "github.com/envaar/vaar/internal/output/lint"
	if !imports[outputLintPath] {
		t.Fatalf("lint CLI must import %q", outputLintPath)
	}
	if imports["github.com/envaar/vaar/internal/report"] {
		t.Fatal("lint CLI must not import internal/report")
	}
}

func TestTextRendersFindingExactly(t *testing.T) {
	findings := []model.Finding{{
		Rule:     "duplicate-key",
		Severity: model.SeverityError,
		File:     ".env",
		Line:     4,
		Message:  "DATABASE_URL is defined more than once",
	}}

	want := "error duplicate-key .env:4 DATABASE_URL is defined more than once\n"
	if got := lintoutput.Text(findings); got != want {
		t.Fatalf("unexpected text output: got %q want %q", got, want)
	}
}

func TestTextMarksFixedFindingExactly(t *testing.T) {
	findings := []model.Finding{{
		Rule:     "trailing-whitespace",
		Severity: model.SeverityWarn,
		File:     ".env",
		Line:     1,
		Message:  "line has trailing whitespace",
		Fixed:    true,
	}}

	want := "[fixed] warn trailing-whitespace .env:1 line has trailing whitespace\n"
	if got := lintoutput.Text(findings); got != want {
		t.Fatalf("unexpected fixed text output: got %q want %q", got, want)
	}
}

func TestJSONRendersFindingExactly(t *testing.T) {
	findings := []model.Finding{{
		Rule:     "duplicate-key",
		Severity: model.SeverityError,
		File:     ".env",
		Line:     4,
		Message:  "DATABASE_URL is defined more than once",
	}}

	want := "{\n  \"findings\": [\n    {\n      \"rule\": \"duplicate-key\",\n      \"severity\": \"error\",\n      \"file\": \".env\",\n      \"line\": 4,\n      \"message\": \"DATABASE_URL is defined more than once\"\n    }\n  ]\n}"
	got, err := lintoutput.JSON(findings)
	if err != nil {
		t.Fatalf("json render failed: %v", err)
	}
	if string(got) != want {
		t.Fatalf("unexpected JSON output: got %q want %q", got, want)
	}
}

func TestJSONRendersNilFindingsAsEmptyArray(t *testing.T) {
	want := "{\n  \"findings\": []\n}"
	got, err := lintoutput.JSON(nil)
	if err != nil {
		t.Fatalf("json render failed: %v", err)
	}
	if string(got) != want {
		t.Fatalf("unexpected empty JSON: got %q want %q", got, want)
	}
}
