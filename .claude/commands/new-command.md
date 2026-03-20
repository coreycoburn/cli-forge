---
description: Add a new command to an existing CLI
---

# Add a New Command

You are adding a command to an existing CLI in the cli-forge monorepo.

## 1. Gather Information

Ask the user for:
- **Target CLI**: which CLI under `cmd/` this command belongs to
- **Command name**: lowercase, hyphen-separated (e.g., `convert`, `optimize`, `list-files`)
- **Description**: one-line description of what the command does
- **Arguments**: what positional args it takes (if any)
- **Flags**: what flags it needs (if any)

## 2. Discover Existing Patterns

Read these files to understand conventions:
- `cmd/{cli}/main.go` — see how commands are registered
- `cmd/{cli}/commands/*.go` — see existing command patterns
- `cmd/example/commands/hello.go` — reference implementation
- `templates/command/command.go.tmpl` — command template

## 3. Create the Command

### Command file: `cmd/{cli}/commands/{name}.go`

Follow the established pattern:

1. **Function name**: `{PascalName}Cmd()` returns `*cobra.Command`
2. **Use field**: the command name with arg placeholders (e.g., `"convert <input> <format>"`)
3. **Args validation**: use `cobra.ExactArgs(n)`, `cobra.MinimumNArgs(n)`, etc.
4. **Flags**: define as local variables, bind with `cmd.Flags()` methods
5. **RunE handler**:
   - Get output: `out := forge.OutputFrom(cmd)`
   - Validate inputs
   - Use `out.Spin()` for long operations
   - Branch on `out.IsInteractive()` for output:
     - Interactive: use `out.Header()`, `out.Success()`, `out.Info()`, etc.
     - JSON: use `out.JSON()` with a structured result

### Go function name convention

Convert the command name to PascalCase for the function:
- `convert` → `ConvertCmd()`
- `list-files` → `ListFilesCmd()`
- `optimize` → `OptimizeCmd()`

## 4. Register the Command

Open `cmd/{cli}/main.go` and add:
1. Import: the commands package (if not already imported)
2. Registration: `app.AddCommand(commands.{PascalName}Cmd())`

## 5. Verify

Run:
```bash
go build ./cmd/{cli}
./bin/{cli} {name} --help
```

## 6. Checklist

- [ ] Command file exists at `cmd/{cli}/commands/{name}.go`
- [ ] Function follows `{PascalName}Cmd() *cobra.Command` pattern
- [ ] Command is registered in `cmd/{cli}/main.go`
- [ ] Both interactive and JSON output paths are implemented
- [ ] `go build ./cmd/{cli}` succeeds
