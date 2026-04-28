# AI Dev Setup CLI

`ai-dev-setup` bootstraps common AI coding CLIs on macOS and Linux.

It currently installs:

- Claude Code from `@anthropic-ai/claude-code`
- OpenAI Codex CLI from `@openai/codex` (`code` is accepted as an alias)
- API keys into `~/.bashrc`, including `NVIDIA_API_KEY`, `ANTHROPIC_API_KEY`, and `OPENAI_API_KEY`

The installer and CLI are Bash-only and use user-local Node.js/npm through `nvm` when a suitable writable Node install is not already available.

## Install

```bash
./install.sh
```

This installs the CLI to `~/.local/bin/ai-dev-setup`, adds `~/.local/bin` to `PATH` through `~/.bashrc`, and bootstraps dependencies such as Node.js/npm.

To install only the wrapper:

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
```

List available versions fetched from the npm registry:

```bash
ai-dev-setup versions claude
ai-dev-setup versions codex
```

During installation, the CLI shows recent versions plus `latest`; you can also type an exact npm version that is not in the displayed list.

Configure API keys:

```bash
ai-dev-setup keys
```

Check the local setup:

```bash
ai-dev-setup doctor
```

## Troubleshooting

If an install reports success but the command is not available in your shell, reload your shell startup file:

```bash
source ~/.bashrc
```

`ai-dev-setup deps` and `ai-dev-setup install ...` also make sure `~/.bashrc` loads `nvm` or includes the npm global `bin` path for future shells.

## Notes

- Claude Code requires Node.js 18 or newer.
- The CLI avoids `sudo npm install -g`. If the existing npm global prefix is not writable, it installs and uses Node through `nvm`.
- API keys are written inside a managed block in `~/.bashrc`. Set `AI_SETUP_RC=/path/to/rcfile` to use a different file.
