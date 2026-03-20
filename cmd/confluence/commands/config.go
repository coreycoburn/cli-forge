package commands

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/coreycoburn/cli-forge/pkg/forge"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"os"
)

// ConfigCmd manages Confluence credentials.
func ConfigCmd() *cobra.Command {
	var show bool

	cmd := &cobra.Command{
		Use:   "config [key] [value]",
		Short: "Manage Confluence credentials",
		Long: `Get and set Confluence credentials.

With no arguments, run interactive setup.
With one argument, show the value. With two, set it.

Keys: base-url, email, token

Examples:
  confluence config                      # interactive setup
  confluence config --show               # show all credentials
  confluence config base-url             # show base URL
  confluence config email user@co.com    # set email
  confluence config token                # prompt for token (masked)`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := forge.OutputFrom(cmd)

			if show {
				return showCredentials(out)
			}

			// No args: interactive setup
			if len(args) == 0 {
				if !out.IsInteractive() {
					return fmt.Errorf("interactive setup requires a terminal; use 'config [key] [value]' instead")
				}
				creds, err := setupCredentials()
				if err != nil {
					return err
				}
				out.Success(fmt.Sprintf("Credentials saved to %s", credentialsPath()))
				_ = creds
				return nil
			}

			key := args[0]

			// Validate key
			if key != "base-url" && key != "email" && key != "token" {
				return fmt.Errorf("unknown key: %s (valid: base-url, email, token)", key)
			}

			// One arg: get value
			if len(args) == 1 {
				// Token with no value: prompt for it (masked input)
				if key == "token" && out.IsInteractive() {
					return promptAndSetToken(out)
				}
				return getValue(out, key)
			}

			// Two args: set value
			return setValue(out, key, args[1])
		},
	}

	cmd.Flags().BoolVar(&show, "show", false, "Show all credentials (token masked)")

	return cmd
}

func getValue(out *forge.Output, key string) error {
	creds, err := loadCredentials()
	if err != nil {
		return err
	}

	var val string
	switch key {
	case "base-url":
		val = creds.BaseURL
	case "email":
		val = creds.Email
	case "token":
		val = maskToken(creds.APIToken)
	}

	if out.IsInteractive() {
		out.Print(val)
	} else {
		return out.JSON(map[string]string{key: val})
	}
	return nil
}

func setValue(out *forge.Output, key, value string) error {
	creds, _ := loadCredentials()
	if creds == nil {
		creds = &credentials{}
	}

	switch key {
	case "base-url":
		creds.BaseURL = strings.TrimRight(value, "/")
	case "email":
		creds.Email = value
	case "token":
		creds.APIToken = value
	}

	if err := saveCredentials(creds); err != nil {
		return err
	}

	out.Success(fmt.Sprintf("Set %s", key))
	return nil
}

func promptAndSetToken(out *forge.Output) error {
	fmt.Print("API Token: ")
	tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}
	fmt.Println()
	token := strings.TrimSpace(string(tokenBytes))

	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	return setValue(out, "token", token)
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
	return token[:4] + "••••" + token[len(token)-4:]
}
