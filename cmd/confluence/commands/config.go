package commands

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/coreycoburn/cli-forge/pkg/forge"
	"github.com/spf13/cobra"
)

// ConfigCmd manages Confluence credentials.
func ConfigCmd() *cobra.Command {
	var show bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage Confluence credentials",
		Long:  "Configure or view Confluence API credentials.\n\nCredentials are stored at ~/.config/confluence/credentials.",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := forge.OutputFrom(cmd)

			if show {
				return showCredentials(out)
			}

			if !out.IsInteractive() {
				return fmt.Errorf("--setup requires interactive mode; use --show to view credentials")
			}

			creds, err := setupCredentials()
			if err != nil {
				return err
			}
			out.Success(fmt.Sprintf("Credentials saved to %s", credentialsPath()))
			_ = creds
			return nil
		},
	}

	cmd.Flags().BoolVar(&show, "show", false, "Show current credentials (token masked)")

	return cmd
}

func showCredentials(out *forge.Output) error {
	creds, err := loadCredentials()
	if err != nil {
		return err
	}

	maskedToken := maskToken(creds.APIToken)

	if out.IsInteractive() {
		theme := out.Theme()
		label := lipgloss.NewStyle().Foreground(theme.Secondary).Width(12)
		value := lipgloss.NewStyle().Foreground(theme.Accent)

		out.Print("")
		out.Print(fmt.Sprintf("  %s %s", label.Render("Base URL"), value.Render(creds.BaseURL)))
		out.Print(fmt.Sprintf("  %s %s", label.Render("Email"), value.Render(creds.Email)))
		out.Print(fmt.Sprintf("  %s %s", label.Render("API Token"), value.Render(maskedToken)))
		out.Print(fmt.Sprintf("  %s %s", label.Render("File"), value.Render(credentialsPath())))
		out.Print("")
	} else {
		return out.JSON(map[string]string{
			"base_url":  creds.BaseURL,
			"email":     creds.Email,
			"api_token": maskedToken,
			"file":      credentialsPath(),
		})
	}

	return nil
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return strings.Repeat("•", len(token))
	}
	return token[:4] + strings.Repeat("•", len(token)-8) + token[len(token)-4:]
}
