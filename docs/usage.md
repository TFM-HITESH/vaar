<!-- Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0 -->

> [!NOTE]
> This page is soon to be deprecated. Command documentation is being moved to [vaar.envaar.dev/docs](https://vaar.envaar.dev/docs).

# Command Map

This page is a short and simple codebase-side map of Vaar's available commands, as well as the related developer-maintained references.

For installation, verification and the fastest first run, start with the [README](../README.md).

## Available Commands

| Command                               | What it does                                         | Read more                                  |
| ------------------------------------- | ---------------------------------------------------- | ------------------------------------------ |
| `vaar lint [flags]`                   | Runs repository lint checks and optional safe fixes. | [Lint guide](./lint/README.md)             |
| `vaar diff <left> <right> [flags]`    | Compares key presence between two dotenv files.      | [Diff guide](./diff/README.md)             |
| `vaar --help` / `vaar help [command]` | Shows command help.                                  | [Help guide](./help/README.md)             |
| `vaar completion <shell>`             | Prints a shell completion script.                    | [Completion guide](./completion/README.md) |
| `vaar --version`                      | Prints the installed version.                        | [README](../README.md)                     |

## Where To Go Next

- Need the lint command reference, flags, exit codes or rule catalog? Read [docs/lint/README.md](./lint/README.md).
- Need rule-by-rule behavior and examples? Read [docs/lint/rules/README.md](./lint/rules/README.md).
- Need to compare two dotenv files or use diff in CI? Read [docs/diff/README.md](./diff/README.md).
- Need the root help screen or help-screen behavior? Read [docs/help/README.md](./help/README.md).
- Need shell setup instructions for completions? Read [docs/completion/README.md](./completion/README.md).

## Common Paths

1. Install Vaar and verify the binary from [README](../README.md).
2. Run `vaar lint` in the repository you want to check.
3. Use `vaar lint --json --output=lint-report.json` when you want to export the JSON report to a file instead of `stdout`.
4. Run `vaar diff .env .env.example` to compare presence of keys between a local dotenv file and example dotenv file.
5. Open [docs/lint/README.md](./lint/README.md) or [docs/diff/README.md](./diff/README.md) for detailed command behavior and exit codes.
