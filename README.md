# Wolfpack

Wolfpack is a Go-based command-line tool for bootstrapping a local AI development environment on macOS and Linux.

It installs AI coding CLIs, configures provider API keys, and installs shared `SKILL.md` bundles into the directories used by Claude Code, OpenAI Codex, and OpenCode.

## What It Does

- Installs Claude Code, OpenAI Codex CLI, and OpenCode CLI from npm.
- Fetches available package versions from the npm registry before installing.
- Installs Node.js/npm through user-local `nvm` when a suitable writable Node install is not already available.
- Writes API keys to a managed block in `~/.bashrc`.
- Installs shared skills from `https://github.com/ayushag-nv/ai-skills`.
- Avoids `sudo npm install -g`; global npm installs are kept user-local when needed.

## Supported Tools

| Tool | npm package | Binary | Install target | Skills destination |
| --- | --- | --- | --- | --- |
| Claude Code | `@anthropic-ai/claude-code` | `claude` | `wolfpack install claude` | `~/.claude/skills` |
| OpenAI Codex CLI | `@openai/codex` | `codex` | `wolfpack install codex` or `wolfpack install code` | `${CODEX_HOME:-~/.codex}/skills` |
| OpenCode CLI | `opencode-ai` | `opencode` | `wolfpack install opencode` | `~/.config/opencode/skills` |

The skills bundle excludes permission-bypass skills such as `ultimate-auto` and `ultimate-bypass`.

## Requirements

- macOS or Linux
- `bash`
- `curl` or `wget`
- `git` recommended for skill installation; Wolfpack falls back to a GitHub archive download if `git clone` fails

Go and Node.js are optional prerequisites. If Go is missing, `install.sh` downloads a temporary Go toolchain to build Wolfpack. If Node.js/npm are missing or not writable for global installs, Wolfpack installs Node.js through `nvm`.

## Installation

From a source checkout:

```bash
git clone https://github.com/ayushag-nv/wolfpack.git
cd wolfpack
./install.sh
```

This builds the Go CLI, installs it to `~/.local/bin/wolfpack`, adds `~/.local/bin` to `PATH` through `~/.bashrc`, and prepares shared runtime dependencies.

Install only the Wolfpack binary:

```bash
./install.sh --no-deps
```

Install to a custom directory:

```bash
INSTALL_DIR="$HOME/bin" ./install.sh
```

After installation, reload your shell if `wolfpack` is not immediately available:

```bash
source ~/.bashrc
```

## Quick Start

Run the interactive setup menu:

```bash
wolfpack
```

Install the full environment:

```bash
wolfpack install
```

The default install flow runs:

1. Claude Code install
2. OpenAI Codex CLI install
3. OpenCode CLI install
4. Shared skills install
5. API key configuration

## Usage

Install one target:

```bash
wolfpack install claude
wolfpack install codex
wolfpack install code
wolfpack install opencode
wolfpack install skills
```

List available npm versions:

```bash
wolfpack versions claude
wolfpack versions codex
wolfpack versions opencode
```

Install or list skills:

```bash
wolfpack skills install
wolfpack skills list
```

Configure API keys:

```bash
wolfpack keys
```

Check the local setup:

```bash
wolfpack doctor
```

Prepare shared dependencies without installing AI CLIs:

```bash
wolfpack deps
```

## Command Reference

```text
wolfpack install [all|claude|codex|code|opencode|skills]
wolfpack versions [claude|codex|code|opencode]
wolfpack skills [install|list]
wolfpack keys
wolfpack deps
wolfpack doctor
wolfpack help
```

`wolfpack install` defaults to `all`. In non-interactive mode, version selection defaults to `latest`; in an interactive terminal, Wolfpack shows recent npm versions and lets you choose one or type an exact version.

## Configuration

Wolfpack is configured with environment variables.

| Variable | Default | Description |
| --- | --- | --- |
| `WOLFPACK_RC` | `~/.bashrc` | Shell rc file used for API keys, PATH entries, and shell wrapper setup. |
| `WOLFPACK_VERSION_LIMIT` | `20` | Number of npm versions to show when selecting a package version. |
| `WOLFPACK_SKILLS_REPO` | `https://github.com/ayushag-nv/ai-skills.git` | Git repository used for shared skills. |
| `WOLFPACK_SKILLS_REF` | `main` | Branch or ref used when fetching shared skills. |
| `WOLFPACK_SKILLS_SOURCE` | unset | Local `ai-skills` checkout to install from instead of fetching from GitHub. |
| `CLAUDE_SKILLS_DIR` | `~/.claude/skills` | Claude Code skills destination. |
| `CODEX_SKILLS_DIR` | `${CODEX_HOME:-~/.codex}/skills` | Codex skills destination. |
| `OPENCODE_SKILLS_DIR` | `~/.config/opencode/skills` | OpenCode skills destination. |
| `NVM_VERSION` | latest detected release | nvm release tag override, for example `v0.40.3`. |

Example:

```bash
WOLFPACK_RC="$HOME/.zshrc" wolfpack keys
```

## API Keys

`wolfpack keys` prompts for:

- `NVIDIA_API_KEY`
- `ANTHROPIC_API_KEY`
- `OPENAI_API_KEY`

You can add additional environment variables during the same prompt flow. Values are written inside a managed block in the configured rc file so future runs can update them safely.

## Skills

Wolfpack installs skills from the `ai-skills` repository into each supported agent's global skills directory:

- Claude Code: `~/.claude/skills`
- OpenAI Codex CLI: `${CODEX_HOME:-~/.codex}/skills`
- OpenCode CLI: `~/.config/opencode/skills`

Use a local skills checkout during development:

```bash
WOLFPACK_SKILLS_SOURCE=/path/to/ai-skills wolfpack skills install
```

## Troubleshooting

Run diagnostics first:

```bash
wolfpack doctor
```

If an install succeeds but the command is not available in the current shell:

```bash
source ~/.bashrc
```

Wolfpack also installs a shell wrapper so future successful `wolfpack install`, `wolfpack deps`, and `wolfpack keys` runs automatically re-source the configured rc file.

If npm global installs fail because the prefix is not writable, run:

```bash
wolfpack deps
```

Wolfpack will install and use user-local Node.js/npm through `nvm`.

## Development

Run from source:

```bash
go run ./cmd/wolfpack help
```

Build:

```bash
go build ./cmd/wolfpack
```

Run checks:

```bash
go test ./...
go vet ./...
```

The source-checkout wrapper at `bin/wolfpack` runs the Go CLI with `go run` when Go is available. Installed users should use `./install.sh`, which produces a standalone binary.

## Repository Layout

| Path | Purpose |
| --- | --- |
| `cmd/wolfpack/` | Go source for the Wolfpack CLI. |
| `bin/wolfpack` | Source-checkout wrapper that runs the CLI with `go run`. |
| `install.sh` | Bootstrap installer that builds and installs the standalone binary. |
| `tests/` | CLI-level integration tests. |
| `go.mod` | Go module definition. |

## Roadmap

Near-term improvements:

- Add `status --json` for machine-readable installed versions, skill state, and missing dependencies.
- Add a structured config file under `~/.config/wolfpack/config.toml`.
- Add checksums or signed release verification for downloaded toolchains and skill archives.
- Add self-update support for Wolfpack and the `ai-skills` bundle.
- Add `skills update`, `skills diff`, and `skills remove`.
- Add first-class shell support for zsh and fish in addition to the current bash-focused startup edits.
- Add optional install targets for tools such as `uv`, `ruff`, `gh`, `glab`, and container tooling.
- Add dry-run support for install commands.
- Expand unit coverage around semver sorting, shell rc-file edits, skill copy behavior, and archive extraction.
