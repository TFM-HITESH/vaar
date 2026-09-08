/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

// Package diff compares dotenv key inventories without exposing values.
package diff

import (
	"fmt"
	"sort"

	"github.com/envaar/vaar/internal/analysis"
	"github.com/envaar/vaar/internal/envfile"
)

// Result describes the key-presence difference between two dotenv files.
type Result struct {
	// Left is the label or path for the first compared file.
	Left string
	// Right is the label or path for the second compared file.
	Right string
	// MissingFromLeft contains keys present in Right but absent from Left.
	MissingFromLeft []string
	// MissingFromRight contains keys present in Left but absent from Right.
	MissingFromRight []string
}

// HasDifferences reports whether either file is missing keys from the other.
func (r Result) HasDifferences() bool {
	return len(r.MissingFromLeft) > 0 || len(r.MissingFromRight) > 0
}

// Compare parses two dotenv inputs and compares assignment key presence only.
// It remains a compatibility wrapper until the diff command migrates to the
// shared source and analysis pipeline.
func Compare(leftPath string, leftData []byte, rightPath string, rightData []byte) (Result, error) {
	left, err := envfile.Parse(leftPath, leftData)
	if err != nil {
		return Result{}, fmt.Errorf("parse %q: %w", leftPath, err)
	}

	right, err := envfile.Parse(rightPath, rightData)
	if err != nil {
		return Result{}, fmt.Errorf("parse %q: %w", rightPath, err)
	}

	return CompareFiles(left, right), nil
}

// CompareInventories compares two value-free analysis key inventories. Labels
// are supplied separately because an empty inventory has no declaration from
// which a display path could be recovered.
func CompareInventories(leftLabel string, left analysis.KeyInventory, rightLabel string, right analysis.KeyInventory) Result {
	leftKeys := left.Keys()
	rightKeys := right.Keys()

	return Result{
		Left:             leftLabel,
		Right:            rightLabel,
		MissingFromLeft:  missingInventoryKeys(leftKeys, rightKeys),
		MissingFromRight: missingInventoryKeys(rightKeys, leftKeys),
	}
}

// CompareFiles compares key presence between two already parsed dotenv files.
// It is a temporary compatibility adapter for callers that have not yet
// migrated to analysis key inventories.
func CompareFiles(left envfile.File, right envfile.File) Result {
	return CompareInventories(
		left.Path,
		inventoryFromFile(left),
		right.Path,
		inventoryFromFile(right),
	)
}

func inventoryFromFile(file envfile.File) analysis.KeyInventory {
	lines := make([]analysis.Line, 0, len(file.Lines))
	for _, line := range file.Lines {
		lines = append(lines, analysis.Line{
			Number:        line.Number,
			Key:           line.Key,
			HasKey:        line.HasKey,
			HasAssignment: line.HasAssignment,
		})
	}

	return analysis.NewKeyInventory(analysis.Document{
		DisplayPath: file.Path,
		Lines:       lines,
	})
}

func missingInventoryKeys(have, want []string) []string {
	haveSet := make(map[string]struct{}, len(have))
	for _, key := range have {
		haveSet[key] = struct{}{}
	}

	missing := make([]string, 0, len(want))
	for _, key := range want {
		if _, ok := haveSet[key]; !ok {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return missing
}
