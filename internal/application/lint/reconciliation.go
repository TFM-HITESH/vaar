// Copyright © 2026 envaar
// SPDX-License-Identifier: Apache-2.0

package lint

import lintmodel "github.com/envaar/vaar/internal/lint"

type findingKey struct {
	rule     string
	severity lintmodel.Severity
	file     string
	message  string
}

func keyForFinding(finding lintmodel.Finding) findingKey {
	return findingKey{
		rule:     finding.Rule,
		severity: finding.Severity,
		file:     finding.File,
		message:  finding.Message,
	}
}

// markFixedFindings retains findings from the initial pass that disappeared
// after applying the plan, while preserving findings from the final pass with
// their updated line positions. Matching intentionally ignores line numbers:
// a prior fix may shift later findings in the same file.
func markFixedFindings(original, remaining []lintmodel.Finding) []lintmodel.Finding {
	remainingCounts := make(map[findingKey]int, len(remaining))
	for _, finding := range remaining {
		remainingCounts[keyForFinding(finding)]++
	}

	findings := make([]lintmodel.Finding, 0, len(original)+len(remaining))
	for _, finding := range original {
		key := keyForFinding(finding)
		if remainingCounts[key] > 0 {
			remainingCounts[key]--
			continue
		}

		finding.Fixed = true
		findings = append(findings, finding)
	}

	return append(findings, remaining...)
}
