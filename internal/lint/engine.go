/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package lint

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/envaar/vaar/internal/analysis"
)

// EngineOptions contains only rule-selection input for one analysis-backed
// lint execution. Scope, filesystem, output and mutation options belong to
// application orchestration instead.
type EngineOptions struct {
	// OnlyRules keeps only the named rule IDs.
	OnlyRules []string
	// SkipRules removes the named rule IDs after selection.
	SkipRules []string
}

// RulePlan is an engine-owned, immutable selection of rules for one run.
// Plans can only be created by an Engine and can only be executed by the
// Engine that created them.
type RulePlan struct {
	engine *Engine
	rules  []Rule
}

// Engine executes lint rules against an immutable analysis snapshot.
type Engine struct {
	rules []Rule
}

// NewEngine copies the provided rules so callers can reuse or mutate their
// input slice without affecting future runs.
func NewEngine(rules ...Rule) *Engine {
	copied := make([]Rule, len(rules))
	copy(copied, rules)
	return &Engine{rules: copied}
}

// SelectRulePlan validates and resolves rule-selection input against the
// engine's registered rules. The returned plan is owned by this engine and can
// be executed without repeating selection.
func (e *Engine) SelectRulePlan(opts EngineOptions) (RulePlan, error) {
	selected, err := selectRules(e.rules, opts.OnlyRules, opts.SkipRules)
	if err != nil {
		return RulePlan{}, err
	}

	return RulePlan{engine: e, rules: selected}, nil
}

// Rules returns a copy of the plan's selected rules. The returned slice can be
// changed by the caller without changing the plan.
func (p RulePlan) Rules() []Rule {
	selected := make([]Rule, len(p.rules))
	copy(selected, p.rules)
	return selected
}

// SelectRules validates and resolves rule-selection input against the engine's
// registered rules. The returned slice is independent of the engine's rule
// collection and is useful to compatibility adapters that need the selected
// rules for fix planning or other orchestration concerns.
func (e *Engine) SelectRules(opts EngineOptions) ([]Rule, error) {
	plan, err := e.SelectRulePlan(opts)
	if err != nil {
		return nil, err
	}
	return plan.Rules(), nil
}

// ValidateRuleSelection checks requested only/skip rule IDs against the
// available rule set without resolving scope or running any rules.
func ValidateRuleSelection(all []Rule, only, skip []string) error {
	_, err := NewEngine(all...).SelectRulePlan(EngineOptions{
		OnlyRules: only,
		SkipRules: skip,
	})
	return err
}

// Run selects and executes rules against snapshot. It performs no filesystem
// I/O, parsing, mutation, rendering or exit-code mapping.
func (e *Engine) Run(ctx context.Context, snapshot analysis.Snapshot, opts EngineOptions) ([]Finding, error) {
	plan, err := e.SelectRulePlan(opts)
	if err != nil {
		return nil, err
	}

	return e.RunPlan(ctx, snapshot, plan)
}

// RunPlan executes the rules selected by this engine against snapshot. It
// checks cancellation before every rule and never performs rule selection.
func (e *Engine) RunPlan(ctx context.Context, snapshot analysis.Snapshot, plan RulePlan) ([]Finding, error) {
	if plan.engine != e {
		return nil, fmt.Errorf("lint rule plan does not belong to engine")
	}

	findings := make([]Finding, 0, 16)
	ruleContext := Context{Snapshot: snapshot}
	for _, rule := range plan.rules {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		ruleFindings, err := rule.Run(ruleContext)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rule.ID(), err)
		}
		findings = append(findings, ruleFindings...)
	}

	SortFindings(findings)
	return findings, nil
}

// selectRules applies --only and --skip in declaration order so the selected
// set stays predictable for completions, tests and report output.
func selectRules(all []Rule, only, skip []string) ([]Rule, error) {
	allowed := make(map[string]Rule, len(all))
	ordered := make([]Rule, 0, len(all))
	for _, rule := range all {
		allowed[rule.ID()] = rule
		ordered = append(ordered, rule)
	}

	if err := validateRuleIDs(only, "--only", allowed); err != nil {
		return nil, err
	}
	if err := validateRuleIDs(skip, "--skip", allowed); err != nil {
		return nil, err
	}

	if len(all) == 0 {
		return nil, nil
	}

	selectedIDs := make(map[string]struct{}, len(all))

	if len(only) > 0 {
		for _, id := range only {
			selectedIDs[strings.TrimSpace(id)] = struct{}{}
		}
	} else {
		for _, rule := range ordered {
			selectedIDs[rule.ID()] = struct{}{}
		}
	}

	if len(skip) > 0 {
		for _, id := range skip {
			delete(selectedIDs, strings.TrimSpace(id))
		}
	}

	selected := make([]Rule, 0, len(selectedIDs))
	for _, rule := range ordered {
		if _, ok := selectedIDs[rule.ID()]; ok {
			selected = append(selected, rule)
		}
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("no lint rules selected after applying --only and --skip")
	}

	return selected, nil
}

func validateRuleIDs(ids []string, flag string, allowed map[string]Rule) error {
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return fmt.Errorf("invalid empty rule ID in %s", flag)
		}
		if _, ok := allowed[id]; !ok {
			return fmt.Errorf("unknown lint rule %q", id)
		}
	}
	return nil
}

// SortFindings orders output by file, line, severity, rule and message so
// repeated runs produce the same report.
func SortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		left := findings[i]
		right := findings[j]
		if left.File != right.File {
			return left.File < right.File
		}
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		if left.Severity.Rank() != right.Severity.Rank() {
			return left.Severity.Rank() < right.Severity.Rank()
		}
		if left.Rule != right.Rule {
			return left.Rule < right.Rule
		}
		return left.Message < right.Message
	})
}
