package forge

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type contextKey string

const outputKey contextKey = "forge_output"

// App is the top-level CLI application. It wraps cobra with the forge theme,
// output modes, and consistent conventions.
type App struct {
	Name        string
	Description string
	Version     string
	Theme       *Theme
	root        *cobra.Command
	jsonFlag    bool
}

// Option configures an App.
type Option func(*App)

// WithTheme overrides the default theme.
func WithTheme(t *Theme) Option {
	return func(a *App) { a.Theme = t }
}

// WithVersion sets the CLI version (injected at build time via ldflags).
func WithVersion(v string) Option {
	return func(a *App) { a.Version = v }
}

// New creates a new CLI application with the given name and description.
func New(name, description string, opts ...Option) *App {
	a := &App{
		Name:        name,
		Description: description,
		Version:     "dev",
		Theme:       DefaultTheme(),
	}

	for _, opt := range opts {
		opt(a)
	}

	a.root = &cobra.Command{
		Use:   name,
		Short: description,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			mode := Interactive
			if a.jsonFlag {
				mode = JSON
			}
			out := NewOutput(mode, a.Theme)
			ctx := context.WithValue(cmd.Context(), outputKey, out)
			cmd.SetContext(ctx)
		},
	}

	a.root.PersistentFlags().BoolVar(&a.jsonFlag, "json", false, "Output as JSON")
	a.root.Version = a.Version
	a.root.SetVersionTemplate(fmt.Sprintf("%s {{.Version}}\n", name))
	a.root.CompletionOptions.HiddenDefaultCmd = true
	a.root.SilenceErrors = true
	a.root.SilenceUsage = true

	a.setStyledHelp()

	return a
}

// AddCommand registers one or more subcommands.
func (a *App) AddCommand(cmds ...*cobra.Command) {
	a.root.AddCommand(cmds...)
}

// Execute runs the CLI. Call this from main().
func (a *App) Execute() {
	if err := a.root.Execute(); err != nil {
		style := lipgloss.NewStyle().Foreground(a.Theme.Error)
		fmt.Fprintln(os.Stderr, style.Render("✗ "+err.Error()))
		os.Exit(1)
	}
}

// OutputFrom retrieves the Output instance from a cobra command's context.
// Call this inside RunE handlers to access themed output.
func OutputFrom(cmd *cobra.Command) *Output {
	if out, ok := cmd.Context().Value(outputKey).(*Output); ok {
		return out
	}
	return NewOutput(Interactive, DefaultTheme())
}
