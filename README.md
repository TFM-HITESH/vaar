<!-- Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0 -->

![Vaar banner](./assets/readme/banner.png)

[![Latest release](https://img.shields.io/github/v/release/envaar/vaar?label=latest)](https://github.com/envaar/vaar/releases/latest)
[![Go version](https://img.shields.io/github/go-mod/go-version/envaar/vaar)](./go.mod)
[![Docs](https://img.shields.io/badge/docs-vaar.envaar.dev%2Fdocs-2e7de9)](https://vaar.envaar.dev/docs)

# Vaar

Vaar is an intelligent environment analysis toolkit for validating, understanding, and safely working with .env files across developer and agentic workflows.

It helps you catch broken `.env` files, misconfigurations in environments and helps you work with your secrets safely, be it development or production, for both humans and agents.

## Why Vaar?

Issues with `.env` files are usually missed or overlooked during code review due to them being difficult to share and compare thanks to their sensitive nature. Since environment variables are not as easily reviewable as code artifacts, it is all the more important to ensure their hygiene and quality.

Vaar provides various tools to analyse your environment configuration safely. Each tool selects a safe scope, parses information from multiple sources, analyses it safely without leaking secrets, applies its
own semantics and produces stable results.

## What makes Vaar different?

Most dotenv linters are limited to `.env` files while secret scanners look for leaked credentials. Generic code-quality tools are not centered on environment correctness. Additionally, runtime validators only run after application code starts.

Vaar has begun with simple `.env` hygiene, but is envisioned to grow into a suite of repo-aware, intelligent environment correctness tooling.

Vaar currently provides:

- deterministic dotenv linting through `vaar lint`
- key-presence comparison between dotenv files through `vaar diff`
- stable text and JSON output for command-line users and automation
- safe, deterministic formatting fixes for supported lint findings

The roadmap extends this foundation toward repository-aware analysis across source code, contracts, infrastructure and external providers. Read the [Roadmap](#roadmap) for the planned direction.

## Installation

### Go install

```bash
go install github.com/envaar/vaar/cmd/vaar@latest
vaar --version
```

Source installs use the Go version declared in go.mod.

### GitHub Release binaries

Release binaries are published for Linux, macOS and Windows on `amd64` and `arm64`.

| Platform | Archives                                               |
| -------- | ------------------------------------------------------ |
| Linux    | `vaar_linux_amd64.tar.gz`, `vaar_linux_arm64.tar.gz`   |
| macOS    | `vaar_darwin_amd64.tar.gz`, `vaar_darwin_arm64.tar.gz` |
| Windows  | `vaar_windows_amd64.zip`, `vaar_windows_arm64.zip`     |

Download the matching archive and `vaar_checksums.txt`, then verify the archive, extract it and confirm the binary.

> [!NOTE]
> The release workflow mandatorily publishes `vaar_checksums.txt` with SHA256 checksums for every release archive.

For Unix-like systems:

```bash
archive=vaar_linux_amd64.tar.gz
curl -LO "https://github.com/envaar/vaar/releases/latest/download/$archive"
curl -LO https://github.com/envaar/vaar/releases/latest/download/vaar_checksums.txt
grep "$archive" vaar_checksums.txt | sha256sum -c -
tar -xzf "$archive"
./vaar --version
```

For macOS systems:

```bash
archive=vaar_linux_amd64.tar.gz
curl -LO "https://github.com/envaar/vaar/releases/latest/download/$archive"
curl -LO https://github.com/envaar/vaar/releases/latest/download/vaar_checksums.txt
grep "$archive" vaar_checksums.txt | shasum -a 256 -c -
tar -xzf "$archive"
./vaar --version
```

For Windows PowerShell:

```powershell
$archive = 'vaar_windows_amd64.zip'
Invoke-WebRequest -Uri "https://github.com/envaar/vaar/releases/latest/download/$archive" -OutFile $archive
Invoke-WebRequest -Uri "https://github.com/envaar/vaar/releases/latest/download/vaar_checksums.txt" -OutFile vaar_checksums.txt
$line = Select-String -Path .\vaar_checksums.txt -Pattern $archive
$expected = ($line.Line -split '\s+')[0].ToLower()
$actual = (Get-FileHash .\$archive -Algorithm SHA256).Hash.ToLower()
if ($actual -ne $expected) { throw "checksum mismatch" }
Expand-Archive .\$archive -DestinationPath .
.\vaar.exe --version
```

## Quick start

Vaar has command-specific guides for detailed flags, output formats and exit
codes. The examples below are simplified for an easy quick start.

### Lint

To test out the linter's capabilities, run Vaar from the repository you want to check:

```bash
vaar lint
```

See the [Lint guide](./docs/lint/README.md) for rule selection, JSON output,
safe fixes, explicit targets, exit codes and the rule catalog. The
[basic example](./examples/basic/README.md) and [broken example](./examples/broken/README.md)
show representative lint input.

### Diff

Compare the keys declared/present in two dotenv files:

```bash
vaar diff .env .env.example
```

Diff compares key presence only and never compares or prints dotenv values. See
the [Diff guide](./docs/diff/README.md) for JSON output, quiet mode, exit codes
and CI usage.

## Command documentation

Use the [command map](./docs/usage.md) to find the supported commands and
their detailed references:

- [Lint guide](./docs/lint/README.md), including the [rule catalog](./docs/lint/rules/README.md)
- [Diff guide](./docs/diff/README.md)
- [Help guide](./docs/help/README.md) and command-specific help references
- [Developer primer](./docs/primer/README.md) for a practical codebase tour

## Roadmap

Vaar's intends to become the one stop solution for all things related to environment variables.

### Current commands

- `vaar lint` checks dotenv syntax and deterministic formatting rules, with safe
  fixes where the result is unambiguous.
- `vaar diff` compares key presence between two dotenv documents without
  exposing their values.

### Planned analysis sources

- source-code usage and repository structure
- contracts and schemas
- Docker, CI and framework configuration
- optional external providers and cloud state

### Planned command capabilities

- `vaar query` for inspecting supported environment facts and status
- broader lint rules that use repository context and explicit contracts
- richer diff comparisons across supported sources and scopes
- machine-readable integrations such as SARIF and CI annotations

> [!NOTE]
> These goals are not set in stone and are subject to change. To propose a
> change in direction or scope, contact [core@envaar.dev](mailto:core@envaar.dev)
> or open a [Discussion](https://github.com/envaar/vaar/discussions).

## Non-goals

Vaar is intentionally focused on environment and configurational correctness across complex use cases. It is not trying to be:

- simple environment syncing across teams without detailed features such as repository context, code usage or drift analysis.
- a tool that treats `.env` files as the only source of truth instead of comparing them with example files, source code and config state.
- runtime-only validation that waits for the application to start instead of analyzing the repo first.
- a generic configuration platform for arbitrary file formats rather than environment-variable correctness.
- a secret manager or vault replacement.
- a guessy scanner that reports weak patterns without clear evidence or reviewable context.
- a deployment or infrastructure orchestration tool that goes beyond environment correctness.

Those boundaries are intentional. Vaar should stay focused before it grows broader.

> [!NOTE]
> These non-goals are not set in stone and are subject to change. If you wish to request to add or reconsider (remove) a non-goal or some new direction/scope that the team should explore, please reach out by sending a mail to [core@envaar.dev](mailto:core@envaar.dev) or open a [Discussion](https://github.com/envaar/vaar/discussions).

## Project Support

| Area                   | Current Support as of Latest Release  |
| ---------------------- | ------------------------------------- |
| Operating systems      | Linux, macOS, Windows                 |
| Architectures          | `amd64`, `arm64`                      |
| Shell completions      | Bash, Zsh, Fish, PowerShell           |
| Official install paths | `go install`, GitHub Release binaries |
| Go toolchain           | Go version declared in go.mod         |

> [!NOTE]
> The scope of "Supported" means that the project expects installation, checksum verification and described vaar functionality to work for the above configurations. To request a new configuration, please create a new [Issue](https://github.com/envaar/vaar/issues)

## Documentation

Vaar's documentation is being moved to
[vaar.envaar.dev/docs](https://vaar.envaar.dev/docs), where command usage and other reference material will be organized.

The documentation at /docs is the developer and maintainer documentation:

- [Map of Commands](./docs/usage.md)
- [Developer Primer](./docs/primer/README.md)
- [System Overview](./docs/system-overview.md)
- [Contributing Guide](./CONTRIBUTING.md)

## System Overview and Repository Layout

Please read [System Overview](./docs/system-overview.md) to learn about the Repository Layout, Design Principles and other useful information.

## Development

Please read [Development Workflow](./CONTRIBUTING.md) to understand the typical development steps for Vaar.

## FAQ / Troubleshooting

- If lint reports nothing, Vaar scanned the selected scope and found no remaining findings. Confirm you are in the repository/root you meant to scan.
- If diff reports no key differences, both dotenv files contain the same declared key set. Values are not compared.
- To verify a downloaded binary, compare it against `vaar_checksums.txt` and then run `vaar --version` after extraction.

## Contributing

Please read [CONTRIBUTING.md](./CONTRIBUTING.md) before opening a pull request.

## Security

Please read [SECURITY.md](./SECURITY.md) before reporting a vulnerability or a redaction concern.

## License

Vaar is licensed under the Apache License 2.0. See [LICENSE](./LICENSE).

## Contact

To contact the maintainers of this project, please reach out by sending a mail to [core@envaar.dev](mailto:core@envaar.dev).
