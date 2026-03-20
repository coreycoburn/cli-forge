package forge

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// setStyledHelp overrides cobra's default help and usage templates with
// themed versions that use the forge color palette.
func (a *App) setStyledHelp() {
	t := a.Theme

	title := lipgloss.NewStyle().Bold(true).Foreground(t.Primary)
	heading := lipgloss.NewStyle().Bold(true).Foreground(t.Accent)
	command := lipgloss.NewStyle().Foreground(t.Primary)
	flag := lipgloss.NewStyle().Foreground(t.Primary)
	desc := lipgloss.NewStyle().Foreground(t.Secondary)
	muted := lipgloss.NewStyle().Foreground(t.Secondary)

	a.root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		var b strings.Builder

		// Title
		if cmd.HasParent() {
			b.WriteString(title.Render(cmd.Parent().Name()+" "+cmd.Name()) + "\n")
		} else {
			b.WriteString(title.Render(cmd.Name()) + "\n")
		}
		b.WriteString(desc.Render(cmd.Short) + "\n")

		// Long description
		if cmd.Long != "" {
			b.WriteString("\n" + cmd.Long + "\n")
		}

		// Usage
		b.WriteString("\n" + heading.Render("Usage") + "\n")
		b.WriteString("  " + muted.Render(cmd.UseLine()) + "\n")
		if cmd.HasAvailableSubCommands() {
			b.WriteString("  " + muted.Render(cmd.CommandPath()+" [command]") + "\n")
		}

		// Commands
		if cmd.HasAvailableSubCommands() {
			b.WriteString("\n" + heading.Render("Commands") + "\n")
			for _, sub := range cmd.Commands() {
				if sub.IsAvailableCommand() {
					name := command.Width(14).Render(sub.Name())
					b.WriteString(fmt.Sprintf("  %s %s\n", name, desc.Render(sub.Short)))
				}
			}
		}

		// Flags
		if cmd.HasAvailableLocalFlags() {
			b.WriteString("\n" + heading.Render("Flags") + "\n")
			for _, line := range strings.Split(cmd.LocalFlags().FlagUsages(), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					b.WriteString("  " + styleFlagLine(line, flag, desc) + "\n")
				}
			}
		}

		// Global flags (only show on subcommands)
		if cmd.HasParent() && cmd.InheritedFlags().HasAvailableFlags() {
			b.WriteString("\n" + heading.Render("Global Flags") + "\n")
			for _, line := range strings.Split(cmd.InheritedFlags().FlagUsages(), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					b.WriteString("  " + styleFlagLine(line, flag, desc) + "\n")
				}
			}
		}

		// Footer
		if cmd.HasAvailableSubCommands() {
			b.WriteString("\n" + muted.Render(fmt.Sprintf("Use \"%s [command] --help\" for more information.", cmd.CommandPath())) + "\n")
		}

		fmt.Fprint(cmd.OutOrStdout(), b.String())
	})
}

// styleFlagLine colors the flag name and description separately.
func styleFlagLine(line string, flagStyle, descStyle lipgloss.Style) string {
	// Flag lines from cobra look like: "--json   Output as JSON"
	// or: "-s, --shout   SHOUT THE GREETING"
	// Find where the description starts (after multiple spaces)
	parts := strings.SplitN(line, "   ", 2)
	if len(parts) == 2 {
		return flagStyle.Render(parts[0]) + "   " + descStyle.Render(strings.TrimSpace(parts[1]))
	}
	return flagStyle.Render(line)
}
