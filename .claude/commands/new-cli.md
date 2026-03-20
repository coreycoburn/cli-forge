---
description: Scaffold a new CLI under cmd/
---

# Scaffold a New CLI

You are creating a new CLI in the cli-forge monorepo. Follow these steps precisely.

## 1. Gather Information

Ask the user for:
- **CLI name**: lowercase, hyphen-separated (e.g., `imgtools`, `dev-utils`)
- **Description**: one-line description of what the CLI does

## 2. Discover Existing Patterns

Read these files to understand the established conventions:
- `cmd/example/main.go` — entry point pattern
- `cmd/example/commands/hello.go` — command implementation pattern
- `pkg/forge/app.go` — framework API

## 3. Create the CLI

### Entry point: `cmd/{name}/main.go`

Use the template at `templates/cli/main.go.tmpl` as the starting point. Replace:
- `{{.Name}}` with the CLI name
- `{{.Description}}` with the description

### Commands directory: `cmd/{name}/commands/`

Create the directory. Optionally scaffold a first command using `/new-command`.

## 4. Register in GoReleaser

Open `.goreleaser.yaml` and add a new build entry under `builds:` and a new brew formula under `brews:`. Follow the commented-out examples already in the file.

## 5. Verify

Run:
```bash
go build ./cmd/{name}
```

Confirm it compiles and `./cmd/{name} --help` shows the expected output.

## 6. Checklist

- [ ] `cmd/{name}/main.go` exists and follows the entry point pattern
- [ ] `cmd/{name}/commands/` directory exists
- [ ] `.goreleaser.yaml` has a build entry for the new CLI
- [ ] `.goreleaser.yaml` has a brew formula for the new CLI
- [ ] `go build ./cmd/{name}` succeeds
