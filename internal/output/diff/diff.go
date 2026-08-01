/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

// Package diff renders typed dotenv key-difference results for humans and machines.
package diff

import (
	"encoding/json"
	"fmt"
	"strings"

	internaldiff "github.com/envaar/vaar/internal/diff"
)

// Text renders a diff result without a trailing newline. The CLI owns writing
// the rendered representation to its output stream.
func Text(result internaldiff.Result) string {
	if !result.HasDifferences() {
		return "No key differences found"
	}

	lines := make([]string, 0, 2)
	if len(result.MissingFromLeft) > 0 {
		lines = append(lines, missingKeysLine(result.Left, result.MissingFromLeft))
	}
	if len(result.MissingFromRight) > 0 {
		lines = append(lines, missingKeysLine(result.Right, result.MissingFromRight))
	}
	return strings.Join(lines, "\n")
}

type payload struct {
	Left             string   `json:"left"`
	Right            string   `json:"right"`
	MissingFromLeft  []string `json:"missing_from_left"`
	MissingFromRight []string `json:"missing_from_right"`
	Different        bool     `json:"different"`
}

// JSON renders a diff result as indented machine-readable output without a
// trailing newline. Nil missing-key slices are represented as empty arrays.
func JSON(result internaldiff.Result) ([]byte, error) {
	missingFromLeft := result.MissingFromLeft
	if missingFromLeft == nil {
		missingFromLeft = []string{}
	}
	missingFromRight := result.MissingFromRight
	if missingFromRight == nil {
		missingFromRight = []string{}
	}

	return json.MarshalIndent(payload{
		Left:             result.Left,
		Right:            result.Right,
		MissingFromLeft:  missingFromLeft,
		MissingFromRight: missingFromRight,
		Different:        result.HasDifferences(),
	}, "", "  ")
}

func missingKeysLine(path string, keys []string) string {
	noun := "key"
	if len(keys) != 1 {
		noun = "keys"
	}
	return fmt.Sprintf("%s is missing %s: %s", path, noun, strings.Join(keys, ", "))
}
