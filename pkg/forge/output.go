package forge

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

// OutputMode controls whether the CLI produces interactive styled output or JSON.
type OutputMode int

const (
	Interactive OutputMode = iota
	JSON
)

// Output handles all user-facing output, adapting between interactive and JSON modes.
type Output struct {
	mode  OutputMode
	theme *Theme
}

// NewOutput creates a new Output with the given mode and theme.
func NewOutput(mode OutputMode, theme *Theme) *Output {
	return &Output{mode: mode, theme: theme}
}

// IsInteractive returns true when output is styled for a human at a terminal.
func (o *Output) IsInteractive() bool {
	return o.mode == Interactive
}

// Header prints a prominent styled heading.
func (o *Output) Header(text string) {
	if o.mode != Interactive {
		return
	}
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(o.theme.Primary)
	fmt.Fprintln(os.Stderr, style.Render(text))
}

// Info prints an informational message.
func (o *Output) Info(text string) {
	if o.mode != Interactive {
		return
	}
	icon := lipgloss.NewStyle().Foreground(o.theme.Accent).Render("●")
	msg := lipgloss.NewStyle().Foreground(o.theme.Secondary).Render(text)
	fmt.Fprintf(os.Stderr, "%s %s\n", icon, msg)
}

// Success prints a success message.
func (o *Output) Success(text string) {
	if o.mode != Interactive {
		return
	}
	icon := lipgloss.NewStyle().Foreground(o.theme.Success).Render("✓")
	fmt.Fprintf(os.Stderr, "%s %s\n", icon, text)
}

// Warn prints a warning message.
func (o *Output) Warn(text string) {
	if o.mode != Interactive {
		return
	}
	icon := lipgloss.NewStyle().Foreground(o.theme.Warning).Render("▲")
	fmt.Fprintf(os.Stderr, "%s %s\n", icon, text)
}

// Error prints an error message. Always visible, even in JSON mode.
func (o *Output) Error(text string) {
	icon := lipgloss.NewStyle().Foreground(o.theme.Error).Render("✗")
	fmt.Fprintf(os.Stderr, "%s %s\n", icon, text)
}

// Print writes unformatted text to stdout. Suppressed in JSON mode.
func (o *Output) Print(text string) {
	if o.mode != Interactive {
		return
	}
	fmt.Fprintln(os.Stdout, text)
}

// JSON encodes a value as indented JSON to stdout.
func (o *Output) JSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Theme returns the output's theme for custom styled rendering.
func (o *Output) Theme() *Theme {
	return o.theme
}
