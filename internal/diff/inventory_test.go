// Copyright © 2026 envaar
// SPDX-License-Identifier: Apache-2.0

package diff_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/envaar/vaar/internal/analysis"
	"github.com/envaar/vaar/internal/diff"
)

func TestCompareInventoriesPreservesKeyPresenceSemantics(t *testing.T) {
	tests := []struct {
		name                 string
		left                 analysis.KeyInventory
		right                analysis.KeyInventory
		wantLeft             string
		wantRight            string
		wantMissingFromLeft  []string
		wantMissingFromRight []string
	}{
		{
			name: "matching inventories including empty assignments",
			left: inventory("left.env",
				assignment("COMMON"),
				assignment("EMPTY"),
			),
			right: inventory("right.env",
				assignment("COMMON"),
				assignment("EMPTY"),
			),
			wantLeft:             "left.env",
			wantRight:            "right.env",
			wantMissingFromLeft:  []string{},
			wantMissingFromRight: []string{},
		},
		{
			name: "missing keys are sorted independently",
			left: inventory("left.env",
				assignment("ZED"),
				assignment("COMMON"),
			),
			right: inventory("right.env",
				assignment("BETA"),
				assignment("ALPHA"),
			),
			wantLeft:             "left.env",
			wantRight:            "right.env",
			wantMissingFromLeft:  []string{"ALPHA", "BETA"},
			wantMissingFromRight: []string{"COMMON", "ZED"},
		},
		{
			name: "duplicates collapse and comparison is case sensitive",
			left: inventory("left.env",
				assignment("TOKEN"),
				assignment("TOKEN"),
			),
			right: inventory("right.env",
				assignment("token"),
			),
			wantLeft:             "left.env",
			wantRight:            "right.env",
			wantMissingFromLeft:  []string{"token"},
			wantMissingFromRight: []string{"TOKEN"},
		},
		{
			name: "bare and malformed declarations are excluded",
			left: inventory("left.env",
				bareKey("BARE"),
				malformedKey("COLON"),
				assignment("VALID"),
			),
			right: inventory("right.env",
				assignment("VALID"),
			),
			wantLeft:             "left.env",
			wantRight:            "right.env",
			wantMissingFromLeft:  []string{},
			wantMissingFromRight: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := diff.CompareInventories(tc.wantLeft, tc.left, tc.wantRight, tc.right)

			if result.Left != tc.wantLeft || result.Right != tc.wantRight {
				t.Fatalf("labels = (%q, %q), want (%q, %q)", result.Left, result.Right, tc.wantLeft, tc.wantRight)
			}
			if !reflect.DeepEqual(result.MissingFromLeft, tc.wantMissingFromLeft) {
				t.Fatalf("MissingFromLeft = %#v, want %#v", result.MissingFromLeft, tc.wantMissingFromLeft)
			}
			if !reflect.DeepEqual(result.MissingFromRight, tc.wantMissingFromRight) {
				t.Fatalf("MissingFromRight = %#v, want %#v", result.MissingFromRight, tc.wantMissingFromRight)
			}
		})
	}
}

func TestCompareInventoriesDoesNotExposeValues(t *testing.T) {
	left := inventory("left.env", assignment("DATABASE_PASSWORD"))
	right := inventory("right.env", assignment("DATABASE_URL"))

	result := diff.CompareInventories("left.env", left, "right.env", right)
	rendered := fmt.Sprintf("%#v", result)
	if rendered != "diff.Result{Left:\"left.env\", Right:\"right.env\", MissingFromLeft:[]string{\"DATABASE_URL\"}, MissingFromRight:[]string{\"DATABASE_PASSWORD\"}}" {
		t.Fatalf("unexpected value-bearing or unstable result rendering: %s", rendered)
	}
}

func TestCompareInventoriesHasDifferencesMatchesResult(t *testing.T) {
	matching := diff.CompareInventories(
		"left.env",
		inventory("left.env", assignment("COMMON")),
		"right.env",
		inventory("right.env", assignment("COMMON")),
	)
	if matching.HasDifferences() {
		t.Fatal("matching inventories reported differences")
	}

	different := diff.CompareInventories(
		"left.env",
		inventory("left.env", assignment("LEFT_ONLY")),
		"right.env",
		inventory("right.env", assignment("RIGHT_ONLY")),
	)
	if !different.HasDifferences() {
		t.Fatal("different inventories reported no differences")
	}
}

func inventory(displayPath string, lines ...analysis.Line) analysis.KeyInventory {
	return analysis.NewKeyInventory(analysis.Document{
		DisplayPath: displayPath,
		Lines:       lines,
	})
}

func assignment(key string) analysis.Line {
	return analysis.Line{Key: key, HasKey: true, HasAssignment: true}
}

func bareKey(key string) analysis.Line {
	return analysis.Line{Key: key, HasKey: true, DelimiterState: analysis.DelimiterMissing}
}

func malformedKey(key string) analysis.Line {
	return analysis.Line{Key: key, HasKey: true, DelimiterState: analysis.DelimiterColon}
}
