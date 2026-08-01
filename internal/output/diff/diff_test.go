/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package diff_test

import (
	"strings"
	"testing"

	model "github.com/envaar/vaar/internal/diff"
	diffoutput "github.com/envaar/vaar/internal/output/diff"
)

func TestTextRendersCleanResultExactly(t *testing.T) {
	result := model.Result{
		Left:  "left.env",
		Right: "right.env",
	}

	if got, want := diffoutput.Text(result), "No key differences found"; got != want {
		t.Fatalf("unexpected clean text: got %q want %q", got, want)
	}
}

func TestTextRendersSingularMissingKeyExactly(t *testing.T) {
	result := model.Result{
		Left:            "left.env",
		MissingFromLeft: []string{"BAR"},
	}

	if got, want := diffoutput.Text(result), "left.env is missing key: BAR"; got != want {
		t.Fatalf("unexpected singular text: got %q want %q", got, want)
	}
}

func TestTextRendersPluralMissingKeysExactly(t *testing.T) {
	result := model.Result{
		Right:            "right.env",
		MissingFromRight: []string{"BAR", "BAZ", "DATABASE_URL"},
	}

	if got, want := diffoutput.Text(result), "right.env is missing keys: BAR, BAZ, DATABASE_URL"; got != want {
		t.Fatalf("unexpected plural text: got %q want %q", got, want)
	}
}

func TestTextRendersBothDirectionsInStableOrder(t *testing.T) {
	result := model.Result{
		Left:             "left.env",
		Right:            "right.env",
		MissingFromLeft:  []string{"BAR"},
		MissingFromRight: []string{"FOO"},
	}

	if got, want := diffoutput.Text(result), "left.env is missing key: BAR\nright.env is missing key: FOO"; got != want {
		t.Fatalf("unexpected two-sided text: got %q want %q", got, want)
	}
}

func TestJSONRendersDifferencesExactly(t *testing.T) {
	result := model.Result{
		Left:             "left.env",
		Right:            "right.env",
		MissingFromLeft:  []string{"BAR"},
		MissingFromRight: []string{"FOO"},
	}

	want := "{\n  \"left\": \"left.env\",\n  \"right\": \"right.env\",\n  \"missing_from_left\": [\n    \"BAR\"\n  ],\n  \"missing_from_right\": [\n    \"FOO\"\n  ],\n  \"different\": true\n}"
	got, err := diffoutput.JSON(result)
	if err != nil {
		t.Fatalf("json render failed: %v", err)
	}
	if string(got) != want {
		t.Fatalf("unexpected JSON: got %q want %q", got, want)
	}
}

func TestJSONRendersMatchingResultWithEmptyArrays(t *testing.T) {
	result := model.Result{
		Left:  "left.env",
		Right: "right.env",
	}

	want := "{\n  \"left\": \"left.env\",\n  \"right\": \"right.env\",\n  \"missing_from_left\": [],\n  \"missing_from_right\": [],\n  \"different\": false\n}"
	got, err := diffoutput.JSON(result)
	if err != nil {
		t.Fatalf("json render failed: %v", err)
	}
	if string(got) != want {
		t.Fatalf("unexpected matching JSON: got %q want %q", got, want)
	}
}

func TestJSONNormalizesNilMissingKeySlices(t *testing.T) {
	got, err := diffoutput.JSON(model.Result{
		Left:  "left.env",
		Right: "right.env",
	})
	if err != nil {
		t.Fatalf("json render failed: %v", err)
	}
	if strings.Contains(string(got), "null") {
		t.Fatalf("JSON must not contain null arrays: %s", got)
	}
}

func TestRenderersDoNotExposeDotenvValues(t *testing.T) {
	result := model.Result{
		Left:             "left.env",
		Right:            "right.env",
		MissingFromLeft:  []string{"DATABASE_URL"},
		MissingFromRight: []string{"DATABASE_PASSWORD"},
	}

	text := diffoutput.Text(result)
	jsonOutput, err := diffoutput.JSON(result)
	if err != nil {
		t.Fatalf("json render failed: %v", err)
	}
	for _, value := range []string{"left-secret", "right-secret"} {
		if strings.Contains(text, value) {
			t.Fatalf("text output leaked value %q: %q", value, text)
		}
		if strings.Contains(string(jsonOutput), value) {
			t.Fatalf("JSON output leaked value %q: %q", value, jsonOutput)
		}
	}
}
