// Copyright © 2026 envaar
// SPDX-License-Identifier: Apache-2.0

package diff_test

import (
	"context"
	"reflect"
	"testing"

	applicationdiff "github.com/envaar/vaar/internal/application/diff"
	"github.com/envaar/vaar/internal/envfile"
	sourcedotenv "github.com/envaar/vaar/internal/source/dotenv"
)

func TestServiceLoadsBothOperandsThroughSharedSourceLoader(t *testing.T) {
	var loadCalls int
	var gotPaths, gotDisplayPaths []string

	service := applicationdiff.NewWithDependencies(applicationdiff.Dependencies{
		LoadDotenv: func(paths, displayPaths []string) ([]sourcedotenv.Document, error) {
			loadCalls++
			gotPaths = append([]string(nil), paths...)
			gotDisplayPaths = append([]string(nil), displayPaths...)

			return []sourcedotenv.Document{
				{
					File: envfile.File{
						Path: displayPaths[0],
						Lines: []envfile.Line{
							{Number: 1, Key: "COMMON", HasKey: true, HasAssignment: true},
							{Number: 2, Key: "LEFT_ONLY", HasKey: true, HasAssignment: true},
						},
					},
					SourcePath: paths[0],
				},
				{
					File: envfile.File{
						Path: displayPaths[1],
						Lines: []envfile.Line{
							{Number: 1, Key: "COMMON", HasKey: true, HasAssignment: true},
							{Number: 2, Key: "RIGHT_ONLY", HasKey: true, HasAssignment: true},
						},
					},
					SourcePath: paths[1],
				},
			}, nil
		},
	})

	result, err := service.Run(context.Background(), applicationdiff.Options{
		LeftPath:  "left env",
		RightPath: "right env",
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if loadCalls != 1 {
		t.Fatalf("source loader calls = %d, want 1", loadCalls)
	}
	if want := []string{"left env", "right env"}; !reflect.DeepEqual(gotPaths, want) {
		t.Fatalf("loaded paths = %#v, want %#v", gotPaths, want)
	}
	if want := []string{"left env", "right env"}; !reflect.DeepEqual(gotDisplayPaths, want) {
		t.Fatalf("display paths = %#v, want %#v", gotDisplayPaths, want)
	}

	if result.Left != "left env" || result.Right != "right env" {
		t.Fatalf("result labels = (%q, %q), want (%q, %q)", result.Left, result.Right, "left env", "right env")
	}
	if want := []string{"RIGHT_ONLY"}; !reflect.DeepEqual(result.MissingFromLeft, want) {
		t.Fatalf("missing from left = %#v, want %#v", result.MissingFromLeft, want)
	}
	if want := []string{"LEFT_ONLY"}; !reflect.DeepEqual(result.MissingFromRight, want) {
		t.Fatalf("missing from right = %#v, want %#v", result.MissingFromRight, want)
	}
}

func TestServiceStopsBeforeLoadingWhenContextIsCanceled(t *testing.T) {
	loadCalls := 0
	service := applicationdiff.NewWithDependencies(applicationdiff.Dependencies{
		LoadDotenv: func([]string, []string) ([]sourcedotenv.Document, error) {
			loadCalls++
			return nil, nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.Run(ctx, applicationdiff.Options{LeftPath: "left", RightPath: "right"})
	if err != context.Canceled {
		t.Fatalf("run error = %v, want %v", err, context.Canceled)
	}
	if loadCalls != 0 {
		t.Fatalf("source loader calls = %d, want 0", loadCalls)
	}
}
