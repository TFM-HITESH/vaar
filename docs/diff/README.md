<!-- Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0 -->

# Diff Guide

This page contains the diff-specific command reference for Vaar. Refer to [README](../../README.md) for installation, verification and the shortest possible first-run steps.

## Description of the Command

`vaar diff` compares the keys declared by two dotenv files. It reports which
keys are missing from each operand without comparing, printing, masking,
hashing or otherwise exposing dotenv values.

To use diff, run the command with exactly two dotenv file paths:

- `vaar diff .env .env.example`

The first path is the left operand and the second path is the right operand.
Relative and absolute paths are supported, including paths containing spaces
when the shell argument is quoted as required.

Diff also comes with additional flags such as:

- `vaar diff --json .env .env.example`
- `vaar diff --quiet .env .env.example`

## Diff Flags

### `--json`

Renders the comparison result as JSON for CI jobs, editor integrations and
wrapper scripts that need structured output instead of text.

The JSON result contains the operand labels, missing key names and a boolean
indicating whether the key sets differ:

```bash
vaar diff --json .env .env.example
```

Example output:

```json
{
  "left": ".env",
  "right": ".env.example",
  "missing_from_left": ["API_URL"],
  "missing_from_right": ["LOCAL_ONLY"],
  "different": true
}
```

`--json` writes the report to standard output. Diff does not provide JSON file
export.

### `--quiet`

Suppresses normal success and difference output when the caller only needs the
exit status:

```bash
vaar diff --quiet .env .env.example
```

`--quiet` cannot be combined with `--json`; that combination is a usage error.

## Output and Exit Codes

`vaar diff` uses simple exit codes that are useful for scripting:

- `0` means no key differences were reported.
- `1` means key differences were reported.
- `2` means the command failed before producing results.

The default output of `vaar diff` is plain text. Use `--json` when you want
machine-readable key differences for automation purposes; use `--quiet` when
you only need the exit status.

When `--quiet` is not used, text or JSON is written to `stdout`. When
`--quiet` is used, normal comparison output is suppressed and no result is
written to `stdout`.

Producing comparison output and passing diff are separate outcomes:

- If both files contain the same key set, `vaar diff` prints `No key differences found` and exits with code `0`.
- If the files contain different key sets, `vaar diff` reports the missing keys or renders the JSON result and exits with code `1`.
- If the arguments are invalid, a file cannot be read, an output operation fails or the flags conflict, `vaar diff` fails before producing a comparison result and exits with code `2`.

## Examples

Vaar parses both files and compares their declared keys. It does not compare
the values assigned to those keys.

These files have the same key set, so the diff is clean even though their
values differ:

```dotenv
# .env
API_URL=https://local.invalid
LOG_LEVEL=debug
```

```dotenv
# .env.example
API_URL=https://example.invalid
LOG_LEVEL=info
```

Empty assignments still declare a key. Duplicate declarations contribute one
key to the comparison. Text and JSON output contain paths and key names only;
they do not include dotenv values or raw source lines.
