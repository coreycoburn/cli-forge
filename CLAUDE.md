# cli-forge

Go monorepo for building self-contained, distributable CLIs with a consistent look and feel.

## Architecture

```
cli-forge/
├── pkg/forge/          # Shared framework (theme, output, spinner, app)
├── cmd/                # Individual CLIs (each compiles to one binary)
│   ├── example/        #   Proof-of-concept CLI
│   │   ├── main.go
│   │   └── commands/
│   └── {name}/         #   Add more CLIs here
├── templates/          # Scaffolding templates for agent skills
├── .goreleaser.yaml    # Multi-platform build + Homebrew tap
└── .github/workflows/  # CI/CD
```

## Core Concepts

### The forge package (`pkg/forge/`)

Every CLI imports `github.com/coreycoburn/cli-forge/pkg/forge` and uses:

- `forge.New(name, description, opts...)` — creates the app with theme, flags, completion
- `forge.OutputFrom(cmd)` — retrieves themed output from any command handler
- `forge.WithTheme(t)` / `forge.WithVersion(v)` — app configuration
- `forge.DefaultTheme()` — the shared visual identity

### Output modes

Every command must support two output paths:

```go
out := forge.OutputFrom(cmd)
if out.IsInteractive() {
    out.Header("Result")
    out.Success("Done")
} else {
    out.JSON(result)
}
```

- **Interactive** (default): styled terminal output via Lipgloss
- **JSON** (`--json` flag): machine-readable output to stdout

### Spinner pattern

Long operations use the spinner:

```go
err := out.Spin("Processing...", func() error {
    return doWork()
})
```

In JSON mode, the spinner is skipped and the function runs directly.

### Output methods

| Method | Purpose | Visibility |
|--------|---------|------------|
| `out.Header(text)` | Bold primary-colored heading | Interactive only |
| `out.Info(text)` | Informational message | Interactive only |
| `out.Success(text)` | Success with checkmark | Interactive only |
| `out.Warn(text)` | Warning message | Interactive only |
| `out.Error(text)` | Error with X mark | Always |
| `out.Print(text)` | Plain text to stdout | Interactive only |
| `out.JSON(v)` | JSON to stdout | Any mode |
| `out.Spin(title, fn)` | Spinner while fn runs | Spinner in interactive, plain in JSON |

### Theme

All CLIs share `DefaultTheme()` which defines:
- Primary, Secondary, Accent colors (with light/dark terminal adaptation)
- Success, Warning, Error colors
- Spinner style

Override per-CLI with `forge.WithTheme(customTheme)`.

## Adding a CLI

Use `/new-cli` to scaffold a new CLI, or manually:

1. Create `cmd/{name}/main.go` (see `cmd/example/main.go`)
2. Create `cmd/{name}/commands/` with command files
3. Add build + brew entries to `.goreleaser.yaml`

## Adding a Command

Use `/new-command` to scaffold a new command, or manually:

1. Create `cmd/{cli}/commands/{name}.go` (see `cmd/example/commands/hello.go`)
2. Register in `cmd/{cli}/main.go` with `app.AddCommand()`

## Command Pattern

Every command function follows this structure:

```go
func ExampleCmd() *cobra.Command {
    // 1. Declare flag variables
    var flagName string

    cmd := &cobra.Command{
        Use:   "example <arg>",
        Short: "One-line description",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            // 2. Get themed output
            out := forge.OutputFrom(cmd)

            // 3. Validate and process
            // 4. Use out.Spin() for long operations
            // 5. Branch output on out.IsInteractive()

            return nil
        },
    }

    // 6. Bind flags
    cmd.Flags().StringVarP(&flagName, "flag", "f", "", "Flag description")

    return cmd
}
```

## Building

```bash
make build          # Build all CLIs to ./bin/
make install        # Install all CLIs to $GOPATH/bin
go build ./cmd/example  # Build a single CLI
```

## Releasing

1. Tag: `git tag v1.0.0 && git push --tags`
2. GitHub Actions runs GoReleaser
3. Binaries appear on GitHub Releases
4. Homebrew formulae pushed to `coreycoburn/homebrew-tap`

Users install via:
```bash
brew tap coreycoburn/tap
brew install example
```

## Dependencies

- [cobra](https://github.com/spf13/cobra) — command tree, flag parsing, completions
- [lipgloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework (spinner)
- [bubbles](https://github.com/charmbracelet/bubbles) — TUI components
