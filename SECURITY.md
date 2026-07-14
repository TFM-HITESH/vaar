<!-- Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0 -->

# Security Policy

Vaar deals with environment files, configuration values and report output that may include sensitive material. Security issues are treated seriously.

## Supported Versions

- `main` during active development
- tagged releases once they are published

If you are checking a specific commit or release candidate, include that reference in your report.

## What to Report

Report any issue that could expose sensitive configuration, including:

- secrets or full environment values being printed in output
- redaction failures in JSON, SARIF, annotations or other reporters
- commands or rules that unexpectedly reveal secret material
- issues that allow a crafted file to bypass masking or change severity in unsafe ways
- unsafe handling of files or generated output that could lead to leakage

## How to Report

Do not open a public issue or pull request for a suspected vulnerability.

Report security concerns privately to the Envaar maintainers at [security@envaar.dev](mailto:security@envaar.dev).

Please ensure to include:

- the version, commit hash or release tag
- the command you ran
- the detailed reproduction steps, configurations and other important data
- whether any secret material was exposed
- any relevant logs or output, with sensitive values redacted

## What To Avoid

- Do not paste raw secrets, tokens, API keys or private certificates into an issue.
- Do not include full `.env` files if a trimmed example will reproduce the problem.
- Do not post exploit details publicly before the issue has been reviewed.

## Response Expectations

The maintainers will acknowledge the report, investigate privately and coordinate any fix or advisory as needed.

If a fix requires coordination with downstream users, the disclosure will be handled with appropriate care.
