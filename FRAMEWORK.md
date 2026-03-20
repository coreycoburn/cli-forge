# FRAMEWORK.md

This file provides guidance to Claude Code when working in a project that uses `cli-framework`.

## What This Is

`cli-framework` is the shared engine for personal Bash CLIs. It provides:

- `lib/deps.sh` — `ensure_dep()` for runtime tool checking and installation
- `lib/utils.sh` — shared helpers (`_human_bytes`, `_pct_change`)
- `.claude/commands/new-command.md` — skill for scaffolding new commands

## How It's Included

The framework lives at `framework/` in each CLI project, added as a git subtree:

```bash
git subtree add --prefix=framework https://github.com/coreycoburn/cli-framework main --squash
```

Update it with:

```bash
git subtree pull --prefix=framework https://github.com/coreycoburn/cli-framework main --squash
```

Or use the `/update-framework` skill.

## What the Entry Point Must Export

Every CLI entry point that uses this framework must export:

```bash
export CLI_NAME="mycli"          # binary name — used in error messages
export CLI_NO_INTERACTIVE=0      # set to 1 by --no-interactive flag
```

`CLI_NO_INTERACTIVE` is a process-scoped variable: it lives only for the duration of
that command invocation and is invisible to other processes or terminals.

## Do Not Edit `framework/`

Files inside `framework/` are managed by the upstream framework repo. Edit them there,
then pull updates via `git subtree pull`. Project-specific code lives in `lib/`, `commands/`,
and `config/` at the project root.
