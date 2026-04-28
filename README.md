# Wolfpack CLI

`wolfpack` bootstraps common AI coding CLIs on macOS and Linux.

It currently installs:

- Claude Code from `@anthropic-ai/claude-code`
- OpenAI Codex CLI from `@openai/codex` (`code` is accepted as an alias)
- OpenCode CLI from `opencode-ai`
- Shared AI skills from `https://github.com/ayushag-nv/ai-skills`
- API keys into `~/.bashrc`, including `NVIDIA_API_KEY`, `ANTHROPIC_API_KEY`, and `OPENAI_API_KEY`

The CLI is implemented in Go and uses user-local Node.js/npm through `nvm` when a suitable writable Node install is not already available. `install.sh` is only a bootstrapper: it builds the Go binary and can download a temporary Go toolchain if Go is not already installed.

## Install

```bash
./install.sh
```

This builds the Go CLI, installs it to `~/.local/bin/wolfpack`, adds `~/.local/bin` to `PATH` through `~/.bashrc`, and bootstraps runtime dependencies such as Node.js/npm.

To install only the binary:

```bash
./install.sh --no-deps
```

## Use

Run the interactive menu:

```bash
wolfpack
```

Install everything:

```bash
wolfpack install
```

Install one tool:

```bash
wolfpack install claude
wolfpack install codex
wolfpack install code
wolfpack install opencode
wolfpack install skills
```

List available versions fetched from the npm registry:

```bash
wolfpack versions claude
wolfpack versions codex
wolfpack versions opencode
```

During installation, the CLI shows recent versions plus `latest`; you can also type an exact npm version that is not in the displayed list.

Install or list skills:

```bash
wolfpack skills install
wolfpack skills list
```

Skills are installed into `~/.claude/skills`, `${CODEX_HOME:-~/.codex}/skills`, and `~/.config/opencode/skills`. The bundle excludes permission-bypass skills such as `ultimate-auto` and `ultimate-bypass`.

Configure API keys:

```bash
wolfpack keys
```

Check the local setup:

```bash
wolfpack doctor
```

## Development

Build from source:

```bash
go build ./cmd/wolfpack
```

The source-checkout wrapper at `bin/wolfpack` runs the Go CLI with `go run` when Go is available. Installed users should use `./install.sh`, which produces a standalone binary.

## Roadmap

Near-term improvements:

- Add a `status --json` mode so other scripts can consume installed versions, skill state, and missing dependencies.
- Add a structured config file under `~/.config/wolfpack/config.toml` for default tool versions, skill destinations, and API-key prompts.
- Add checksums or signed release verification for downloaded toolchains and skill archives.
- Add self-update support for both `wolfpack` and the `ai-skills` bundle.
- Add `skills update`, `skills diff`, and `skills remove` commands instead of only install/list.
- Add first-class shell support for zsh/fish in addition to the current bash-focused startup edits.
- Add optional install targets for common developer tools such as `uv`, `ruff`, `gh`, `glab`, and container tooling.
- Add dry-run support for install commands so changes to shell files and destination directories can be reviewed first.
- Add proper unit tests around semver sorting, shell rc-file edits, skill copy behavior, and archive extraction.

## Troubleshooting

If an install reports success but the command is not available in your shell, reload your shell startup file:

```bash
source ~/.bashrc
```

`wolfpack deps` and `wolfpack install ...` also make sure `~/.bashrc` loads `nvm` or includes the npm global `bin` path for future shells. They also install a shell wrapper so future `wolfpack install`, `wolfpack deps`, and `wolfpack keys` runs automatically re-source `~/.bashrc` after successful setup changes.

## Notes

- Claude Code requires Node.js 18 or newer.
- The CLI avoids `sudo npm install -g`. If the existing npm global prefix is not writable, it installs and uses Node through `nvm`.
- API keys are written inside a managed block in `~/.bashrc`. Set `WOLFPACK_RC=/path/to/rcfile` to use a different file.
- Set `WOLFPACK_SKILLS_REPO=...` or `WOLFPACK_SKILLS_REF=...` to install skills from a different git source.
- Set `WOLFPACK_SKILLS_SOURCE=/path/to/ai-skills` to install skills from a local checkout instead of fetching GitHub.
