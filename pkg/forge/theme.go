package forge

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// Theme defines the visual identity for a CLI. All CLIs in the forge family
// share the same default theme, creating a consistent look and feel.
type Theme struct {
	Primary   lipgloss.AdaptiveColor
	Secondary lipgloss.AdaptiveColor
	Accent    lipgloss.AdaptiveColor
	Success   lipgloss.AdaptiveColor
	Warning   lipgloss.AdaptiveColor
	Error     lipgloss.AdaptiveColor
	Spinner   spinner.Spinner
}

// DefaultTheme returns the standard forge theme.
func DefaultTheme() *Theme {
	return &Theme{
		Primary:   lipgloss.AdaptiveColor{Light: "#7B2FBE", Dark: "#FF6AC1"},
		Secondary: lipgloss.AdaptiveColor{Light: "#666666", Dark: "#8B8B8B"},
		Accent:    lipgloss.AdaptiveColor{Light: "#5A4FCF", Dark: "#7B61FF"},
		Success:   lipgloss.AdaptiveColor{Light: "#2ECC71", Dark: "#50FA7B"},
		Warning:   lipgloss.AdaptiveColor{Light: "#E67E22", Dark: "#FFB86C"},
		Error:     lipgloss.AdaptiveColor{Light: "#E74C3C", Dark: "#FF5555"},
		Spinner:   spinner.Dot,
	}
}
