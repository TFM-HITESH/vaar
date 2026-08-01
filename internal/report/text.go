/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package report

import (
	model "github.com/envaar/vaar/internal/lint"
	lintoutput "github.com/envaar/vaar/internal/output/lint"
)

// Text renders findings one per line in a stable, grep-friendly format and
// returns an empty string for clean runs.
func Text(findings []model.Finding) string {
	return lintoutput.Text(findings)
}
