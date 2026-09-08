<!-- Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0 -->

# System Overview

Vaar keeps the command layer thin and pushes the real logic into a few focused packages. That makes the project easier to test, easier to extend and easier to keep stable as it grows. This page explains the current layout, the lint and diff pipeline and the right place for new work. For command usage, see [Usage](./usage.md).

## Design Principles

- Keep the command layer thin and declarative in nature.
- Keep parsing, source modeling and pure byte transforms in
  `internal/source/`; keep generic filesystem mechanics in `internal/fs/`.
- Keep normalized analysis facts value-free in `internal/analysis/`.
- Make repository discovery explicit and deterministic.
- Prefer stable ordering and stable output so that scripts can rely on results.
- Preserve original environment file state unless a safe fix intentionally changes them.
- Treat documentation as part of the public contract that has to be maintained.

## Package Map

| Package                              | Responsibility                     | Notes                                                                                     |
| ------------------------------------ | ---------------------------------- | ----------------------------------------------------------------------------------------- |
| `cmd/vaar/`                          | Program entrypoint                 | Starts the binary and hands control to `internal/cli`.                                    |
| `internal/cli/`                      | Command wiring and exit codes      | Handles Cobra setup, flag translation and user-facing error handling.                     |
| `internal/fs/`                       | Filesystem utilities               | Walks repositories, validates paths, preserves modes and performs safe atomic operations. |
| `internal/scope/`                    | Scope selection                    | Decides which paths the command should analyse.                                           |
| `internal/source/dotenv/`            | Dotenv source boundary             | Loads files, owns the dotenv parser/model and pure source-specific byte transforms.       |
| `internal/analysis/`                 | Shared analysis facts              | Stores ordered, value-free snapshots and key inventories.                                 |
| `internal/analysis/dotenv/`          | Dotenv analysis adapter            | Converts source documents into value-free analysis documents.                             |
| `internal/lint/`                     | Lint engine and rule contracts     | Selects rules, executes them against analysis and sorts findings.                         |
| `internal/lint/rules/`               | Rule catalog                       | Exposes `All()` and compatibility helpers for the rule packages.                          |
| `internal/lint/rules/deterministic/` | Deterministic rule implementations | One file per rule along with focused tests and docs.                                      |
| `internal/application/lint/`         | Lint application workflow          | Coordinates scope, source loading, analysis, linting and optional mutations.              |
| `internal/mutations/`                | Mutation planning/application      | Validates plans and applies safe, permission-preserving replacements.                     |
| `internal/diff/`                     | Diff engine                        | Compares analysis key inventories.                                                        |
| `internal/application/diff/`         | Diff application workflow          | Coordinates source loading, analysis and diff comparison.                                 |
| `internal/output/`                   | Output adapters                    | Produces typed terminal and JSON representations for command results.                     |

## How a Lint Run Works

1. Resolve the repository root and selected scope.
2. Load the selected files through `internal/source/dotenv`.
3. Convert source documents into a value-free `internal/analysis` snapshot.
4. Select rules and run the lint engine against that snapshot.
5. If automatic safe fixing is enabled with `--fix`, build and validate a
   mutation plan, apply it through the mutation/filesystem boundaries, then
   reload and re-analyse the same scope.
6. Sort findings into a stable order.
7. Render the typed result through `internal/output`.
8. Map the final result to the command exit code at the CLI boundary.

The source boundary keeps original bytes, line numbers, BOM state and
line-ending information available to source and mutation stages. The analysis
snapshot exposes only safe, value-free facts to engines and reporters.

## How a Diff Run Works

1. Accept exactly two dotenv paths at the CLI boundary.
2. Pass both paths to `internal/application/diff/`.
3. Load both files through `internal/source/dotenv/`, preserving operand order
   and display paths.
4. Convert both source documents into value-free analysis documents.
5. Build document-local key inventories through `internal/analysis/`.
6. Compare the inventories through `internal/diff/`.
7. Produce a typed diff result containing only paths, missing keys and the
   difference status.
8. Render text or JSON through `internal/output/diff/`.
9. Map the final result to the command exit code at the CLI boundary.

Diff compares key presence only. It does not compare dotenv values, raw source
lines or source bytes.

## Where to Add New Functionality/Code

- New command or flag behavior: `internal/cli/`
- Repository walking or ignore logic: `internal/fs/`
- Dotenv parsing and source-specific formatting transforms: `internal/source/dotenv/`
- Scope selection: `internal/scope/`
- Rule selection and execution: `internal/lint/`
- Lint workflow orchestration: `internal/application/lint/`
- Mutation planning and application: `internal/mutations/`
- Diff comparison: `internal/diff/` and `internal/application/diff/`
- A new deterministic lint rule: `internal/lint/rules/deterministic/`
- Reporter changes: `internal/output/`
- User-facing documentation: `README.md`, `docs/usage.md`, `docs/lint/README.md` or the relevant rule page

If a change affects user-facing behaviour, add or update tests and document the behaviour appropriately.

> [!NOTE]
> Please note that the CLI command set, documented flags, exit codes, output formats and shipped rule behaviour should remain stable. While internal package names may evolve or change, their behaviour should stay stable and documented once released. Breaking changes should ensure backwards compatibility wherever possible or proper migration paths wherever not.
