# AI Dev Setup CLI

`ai-dev-setup` bootstraps common AI coding CLIs on macOS and Linux.

It currently installs:

- Claude Code from `@anthropic-ai/claude-code`
- OpenAI Codex CLI from `@openai/codex` (`code` is accepted as an alias)
- Shared AI skills from `https://github.com/ayushag-nv/ai-skills`
- API keys into `~/.bashrc`, including `NVIDIA_API_KEY`, `ANTHROPIC_API_KEY`, and `OPENAI_API_KEY`

The CLI is implemented in Go and uses user-local Node.js/npm through `nvm` when a suitable writable Node install is not already available. `install.sh` is only a bootstrapper: it builds the Go binary and can download a temporary Go toolchain if Go is not already installed.

## Install

```bash
./install.sh
```

This builds the Go CLI, installs it to `~/.local/bin/ai-dev-setup`, adds `~/.local/bin` to `PATH` through `~/.bashrc`, and bootstraps runtime dependencies such as Node.js/npm.

To install only the binary:

```bash
./install.sh --no-deps
```

## Use

Run the interactive menu:

```bash
ai-dev-setup
```

Install everything:

```bash
ai-dev-setup install
```

Install one tool:

```bash
ai-dev-setup install claude
ai-dev-setup install codex
ai-dev-setup install code
ai-dev-setup install skills
```

List available versions fetched from the npm registry:

```bash
ai-dev-setup versions claude
ai-dev-setup versions codex
```

During installation, the CLI shows recent versions plus `latest`; you can also type an exact npm version that is not in the displayed list.

Install or list skills:

```bash
ai-dev-setup skills install
ai-dev-setup skills list
```

Skills are installed into `~/.claude/skills` and `${CODEX_HOME:-~/.codex}/skills`. The bundle excludes permission-bypass skills such as `ultimate-auto` and `ultimate-bypass`.

Configure API keys:

```bash
ai-dev-setup keys
```

Check the local setup:

```bash
ai-dev-setup doctor
```

## Development

Build from source:

```bash
go build ./cmd/ai-dev-setup
```

The source-checkout wrapper at `bin/ai-dev-setup` runs the Go CLI with `go run` when Go is available. Installed users should use `./install.sh`, which produces a standalone binary.

## Roadmap

Near-term improvements:

- Add a `status --json` mode so other scripts can consume installed versions, skill state, and missing dependencies.
- Add a structured config file under `~/.config/ai-dev-setup/config.toml` for default tool versions, skill destinations, and API-key prompts.
- Add checksums or signed release verification for downloaded toolchains and skill archives.
- Add self-update support for both `ai-dev-setup` and the `ai-skills` bundle.
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

`ai-dev-setup deps` and `ai-dev-setup install ...` also make sure `~/.bashrc` loads `nvm` or includes the npm global `bin` path for future shells. They also install a shell wrapper so future `ai-dev-setup install`, `ai-dev-setup deps`, and `ai-dev-setup keys` runs automatically re-source `~/.bashrc` after successful setup changes.

## Notes

- Claude Code requires Node.js 18 or newer.
- The CLI avoids `sudo npm install -g`. If the existing npm global prefix is not writable, it installs and uses Node through `nvm`.
- API keys are written inside a managed block in `~/.bashrc`. Set `AI_SETUP_RC=/path/to/rcfile` to use a different file.
- Set `AI_SETUP_SKILLS_REPO=...` or `AI_SETUP_SKILLS_REF=...` to install skills from a different git source.
- Set `AI_SETUP_SKILLS_SOURCE=/path/to/ai-skills` to install skills from a local checkout instead of fetching GitHub.
