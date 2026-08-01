/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

// Package report is a temporary compatibility facade for lint output.
//
// New code should use internal/output/lint directly. The forwarding functions
// remain until the diff output migration removes this package in #71.
package report

import (
	model "github.com/envaar/vaar/internal/lint"
	lintoutput "github.com/envaar/vaar/internal/output/lint"
)

// JSON renders findings as indented machine-readable output with an explicit
// findings array, even when the run found nothing.
func JSON(findings []model.Finding) ([]byte, error) {
	return lintoutput.JSON(findings)
}
