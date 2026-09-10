<!-- Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0 -->

# Vaar Primer

This page is a short introduction to Vaar for new developers. It explains what the project does, how lint and diff executions move through the codebase and how to run the tool locally.

This primer complements the reference docs: [System Overview](../system-overview.md)
for package boundaries, [Usage](../usage.md) for the command map and
[Lint Guide](../lint/README.md) for the full flag and exit-code reference.

## Foundations

### What Vaar Is

Vaar is a Go command-line tool for checking environment configuration. It discovers dotenv files in a repository, runs the built-in rule set, reports findings with file and line numbers and applies safe formatting fixes with `--fix`.

### The Problem It Solves

`.env` files are rarely reviewed with the same rigor as code. Because they are sensitive, they are difficult to share and diff. So duplicate keys, wrong delimiters, stray whitespaces, non-portable key names and other such issues slip through unnoticed. Vaar makes that drift visible and, where a fix is unambiguous, repairs it.

### What Vaar Checks

By default Vaar discovers `.env` and any filename beginning with `.env.`, such
as `.env.example`, `.env.local` or `.env.preview-local`. It does not treat
`.env.`, `.environment`, `.envrc` or `my.env` as dotenv files.

Discovery is recursive and stays deterministic. The repository walk skips generated, vendored, fixture and VCS directories, as implemented in `internal/fs/`.

### Design Principles

These principles detailed in the [System Overview](../system-overview.md) and are visible in the package layout:

- Deterministic first. Every rule is currently under `internal/lint/rules/deterministic/` and reports an exact finding from file content.
- Stable ordering and output. Findings are sorted by file, line, severity, rule and message so scripts and CI can rely on the result.
- Preserve original file state where possible. The dotenv source and mutation
  boundaries may retain original bytes, line numbers, BOM state and
  line-ending information, while analysis exposes only safe facts to engines
  and reporters.
- Keep the command layer thin. `internal/cli/` handles command wiring, flags and exit codes while the main logic stays in focused packages.

## Components

Lint and diff runs pass through a small set of packages. Each package has one
main responsibility.

| Stage             | Package                                           | Responsibility                                                                                                                                                          |
| ----------------- | ------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Entrypoint        | `cmd/vaar/`                                       | Starts the binary and hands control to `internal/cli`.                                                                                                                  |
| CLI layer         | `internal/cli/`                                   | Cobra command wiring, flag translation, exit codes and user-facing errors.                                                                                              |
| Scope / discovery | `internal/scope/`, `internal/fs/`                 | Selects targets, walks the tree, matches known dotenv filenames and skips ignored directories such as `.git`, `build`, `dist`, `node_modules`, `testdata` and `vendor`. |
| Dotenv source     | `internal/source/dotenv/`                         | Loads files, parses them into a source-owned line-aware model and provides pure dotenv byte transforms.                                                                 |
| Analysis          | `internal/analysis/`, `internal/analysis/dotenv/` | Converts source documents into ordered, value-free facts and key inventories.                                                                                           |
| Rule engine       | `internal/lint/`                                  | Selects rules with `--only`/`--skip`, runs them against analysis and sorts findings.                                                                                    |
| Lint application  | `internal/application/lint/`                      | Coordinates loading, analysis, lint execution and optional mutation re-analysis.                                                                                        |
| Mutations         | `internal/mutations/`                             | Plans fixes without filesystem I/O, validates the complete plan and applies safe per-file replacements through `internal/fs/`.                                          |
| Diff application  | `internal/application/diff/`, `internal/diff/`    | Loads both operands, builds analysis inventories and compares keys.                                                                                                     |
| Rule registry     | `internal/lint/rules/`                            | `All()` returns the built-in rule set in a stable order over the category packages.                                                                                     |
| Report / output   | `internal/output/`                                | Renders typed lint and diff results as plain text or JSON.                                                                                                              |

### How A Lint Run Works

The lint application service in `internal/application/lint/` ties the source,
analysis, lint and output boundaries together:

1. Select the active rules after applying `--only` and `--skip`.
2. Resolve the lint scope: the whole repository, one `--target` file or one `--target-dir` tree.
3. Load selected dotenv sources and convert them into a value-free analysis snapshot.
4. Run the selected rules against that snapshot.
5. If `--fix` is enabled, build a safe mutation plan, validate the
   complete plan, apply safe changes, reload the selected scope, rebuild
   analysis and rerun the rules, marking findings that disappeared as
   `[fixed]`.
6. Sort the findings into a stable order.
7. Render the typed result through `internal/output/`.
8. Map the result to an exit code in `internal/cli/`.

Each finding carries a severity (`warn` or `error`), a rule name, the file, a line number and a message. The built-in rules are catalogued in [docs/lint/rules/README.md](../lint/rules/README.md).

Mutation application validates every planned destination before it creates a
replacement. It rejects duplicate physical destinations, coordinates Vaar
writers through shared locks, and rechecks the target identity, content and
permission bits before finalization. Each replacement uses a same-directory
temporary file and the captured permission mode. The plan applies changes in
order; it does not provide a transaction or rollback across multiple files.
Vaar's stale-state guarantee covers writers that use the shared filesystem
primitives.

### How A Diff Run Works

The diff application service in `internal/application/diff/` ties the source,
analysis, diff and output boundaries together:

1. The CLI validates that exactly two dotenv paths were provided.
2. The application service loads both operands through
   `internal/source/dotenv/`, preserving their order and display paths.
3. The source documents are converted into value-free analysis documents.
4. `internal/analysis/` builds a key inventory for each document.
5. `internal/diff/` compares key presence between the two inventories.
6. The service returns a typed diff result containing paths, missing keys and
   difference status.
7. `internal/output/diff/` renders the result as text or JSON.
8. `internal/cli/` maps the result to the process exit code.

Diff compares key presence only. It does not compare dotenv values, raw source
lines or source bytes. A clean comparison exits with `0`, a comparison with
missing keys exits with `1`, and an input, loading or rendering failure exits
with `2`.

## Getting Started

### Build And Verify

For install-from-release and `go install` paths, see the [README Installation section](../../README.md#installation). To build from a clone:

```bash
make build
./bin/vaar --version
```

```text
vaar version dev
```

A source build reports the version as `dev`. Release binaries report the release tag.

The examples below use `./bin/vaar` from a local clone. If `vaar` is already on your `PATH`, use the same commands without the `./bin/` prefix.

### First Lint Run

From the Vaar repository, run the linter against the [broken example](../../examples/broken/README.md):

```bash
./bin/vaar lint --target=examples/broken/.env.example
```

It reports:

```text
warn space-character examples/broken/.env.example:2 line has spaces around the key, delimiter or value
warn trailing-whitespace examples/broken/.env.example:2 line has trailing whitespace
error duplicate-key examples/broken/.env.example:4 APP_ENV is defined more than once
error incorrect-delimiter examples/broken/.env.example:5 DATABASE_URL uses ':' instead of '='
error invalid-key-name examples/broken/.env.example:6 api-key is not a portable env key name
warn ending-blank-line examples/broken/.env.example:8 file must end with exactly one final newline
warn extra-blank-line examples/broken/.env.example:8 repeated blank line
```

### Reading The Output

Every line follows `<severity> <rule> <file>:<line> <message>`; during `--fix` runs, repaired findings additionally carry a `[fixed]` prefix (see below). A clean run prints nothing. Exit codes are script-friendly:

- `0` - no findings remain (fixes made during `--fix` are still reported).
- `1` - findings remain.
- `2` - the command failed before producing results (for example, an unknown rule name).

### JSON Output

Use `--json` when you need structured findings. Add `--output`/`-o` to write the JSON to a file instead of stdout (it requires `--json`):

```bash
./bin/vaar lint --target=examples/broken/.env.example --json
./bin/vaar lint --target=examples/broken/.env.example --json --output=findings.json
```

```json
{
  "findings": [
    {
      "rule": "duplicate-key",
      "severity": "error",
      "file": "examples/broken/.env.example",
      "line": 4,
      "message": "APP_ENV is defined more than once"
    }
  ]
}
```

### Applying Safe Fixes

`--fix` normalizes any findings that can be repaired safely, then re-checks the file. Repaired findings are prefixed with `[fixed]`; anything that still needs human intervention (such as a duplicate key or a wrong delimiter) remains:

```bash
./bin/vaar lint --target=examples/broken/.env.example --fix
```

```text
warn space-character examples/broken/.env.example:2 line has spaces around the key, delimiter or value
[fixed] warn trailing-whitespace examples/broken/.env.example:2 line has trailing whitespace
error duplicate-key examples/broken/.env.example:4 APP_ENV is defined more than once
error incorrect-delimiter examples/broken/.env.example:5 DATABASE_URL uses ':' instead of '='
error invalid-key-name examples/broken/.env.example:6 api-key is not a portable env key name
[fixed] warn ending-blank-line examples/broken/.env.example:8 file must end with exactly one final newline
[fixed] warn extra-blank-line examples/broken/.env.example:8 repeated blank line
```

### Selecting Rules

`--only` runs just the chosen rules; `--skip` removes rules specific rules. Both flags are repeatable and an unknown rule name fails the run before linting starts:

```bash
./bin/vaar lint --target=examples/broken/.env.example --only=duplicate-key
./bin/vaar lint --target=examples/broken/.env.example --skip=trailing-whitespace --skip=extra-blank-line
```

```text
error duplicate-key examples/broken/.env.example:4 APP_ENV is defined more than once
```

### Choosing Search Scope

By default, Vaar walks the whole repository. Narrow the scope with `--target` for one file or `--target-dir` for one tree; the two are mutually exclusive:

```bash
vaar lint --target=.env.staging
vaar lint --target-dir=src
```

## Where To Go Next

- Every flag, output detail and exit code: [Lint Guide](../lint/README.md).
- The full command map: [Usage](../usage.md).
- What each rule catches, with good and bad examples: [Rule Reference](../lint/rules/README.md).
- Package boundaries and where new code belongs: [System Overview](../system-overview.md).
- Contributing a change or a new rule: [CONTRIBUTING.md](../../CONTRIBUTING.md).
