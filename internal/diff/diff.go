/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

// Package diff compares dotenv key inventories without exposing values.
package diff

import (
	"sort"

	"github.com/envaar/vaar/internal/analysis"
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
