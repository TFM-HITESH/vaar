/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

// Package lint renders typed lint findings for humans and machines.
package lint

import (
	"encoding/json"
	"fmt"
	"strings"

	model "github.com/envaar/vaar/internal/lint"
)

// Text renders findings one per line in a stable, grep-friendly format and
// returns an empty string for clean runs.
func Text(findings []model.Finding) string {
	if len(findings) == 0 {
		return ""
	}

	var builder strings.Builder
	for i, finding := range findings {
		if i > 0 {
			builder.WriteByte('\n')
		}
		if finding.Fixed {
			builder.WriteString("[fixed] ")
		}
		fmt.Fprintf(&builder, "%s %s %s:%d %s", finding.Severity, finding.Rule, finding.File, finding.Line, finding.Message)
	}
	builder.WriteByte('\n')
	return builder.String()
}

type payload struct {
	Findings []model.Finding `json:"findings"`
}

// JSON renders findings as indented machine-readable output with an explicit
// findings array, even when the run found nothing.
func JSON(findings []model.Finding) ([]byte, error) {
	if findings == nil {
		findings = []model.Finding{}
	}

	return json.MarshalIndent(payload{Findings: findings}, "", "  ")
}
