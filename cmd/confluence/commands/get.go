package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/coreycoburn/cli-forge/pkg/forge"
	"github.com/spf13/cobra"
)

// GetCmd fetches a Confluence page by ID.
func GetCmd() *cobra.Command {
	var setup bool

	cmd := &cobra.Command{
		Use:   "get [--setup] <page-id>",
		Short: "Fetch a Confluence page by ID",
		Long: `Fetch a Confluence page by ID and output the response.

Credentials are read from environment variables or ~/.config/confluence/credentials:
  CONFLUENCE_BASE_URL    e.g. https://yoursite.atlassian.net
  CONFLUENCE_EMAIL       your Atlassian account email
  CONFLUENCE_API_TOKEN   from https://id.atlassian.com/manage-profile/security/api-tokens`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := forge.OutputFrom(cmd)
			pageID := args[0]

			// Load or set up credentials
			creds, err := resolveCredentials(out, setup, pageID)
			if err != nil {
				return err
			}

			// Fetch page
			var body []byte
			err = out.Spin(fmt.Sprintf("Fetching page %s...", pageID), func() error {
				var fetchErr error
				body, fetchErr = fetchPage(creds, pageID)
				return fetchErr
			})
			if err != nil {
				return err
			}

			// Parse response
			var page map[string]any
			if err := json.Unmarshal(body, &page); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			if out.IsInteractive() {
				renderPage(out, creds, page, pageID)
				copyToClipboard(body)
				out.Info("Full JSON copied to clipboard")
			} else {
				return out.JSON(page)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&setup, "setup", false, "(Re-)configure Confluence credentials")

	return cmd
}

func resolveCredentials(out *forge.Output, doSetup bool, pageID string) (*credentials, error) {
	if doSetup {
		if !out.IsInteractive() {
			return nil, fmt.Errorf("--setup requires interactive mode")
		}
		creds, err := setupCredentials()
		if err != nil {
			return nil, err
		}
		out.Success(fmt.Sprintf("Credentials saved to %s", credentialsPath()))
		fmt.Println()
		return creds, nil
	}

	creds, err := loadCredentials()
	if err != nil {
		if !out.IsInteractive() {
			return nil, err
		}
		out.Warn("Confluence credentials not configured — starting setup")
		fmt.Println()
		creds, err = setupCredentials()
		if err != nil {
			return nil, err
		}
		out.Success(fmt.Sprintf("Credentials saved to %s", credentialsPath()))
		fmt.Println()
	}
	return creds, nil
}

func fetchPage(creds *credentials, pageID string) ([]byte, error) {
	url := fmt.Sprintf("%s/wiki/api/v2/pages/%s?body-format=storage", creds.BaseURL, pageID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(creds.Email, creds.APIToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]any
		msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		if json.Unmarshal(body, &errResp) == nil {
			if m, ok := errResp["message"].(string); ok {
				msg = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, m)
			}
		}
		return nil, fmt.Errorf("API error (%s)", msg)
	}

	return body, nil
}

func renderPage(out *forge.Output, creds *credentials, page map[string]any, pageID string) {
	theme := out.Theme()
	header := lipgloss.NewStyle().Bold(true).Foreground(theme.Primary)
	label := lipgloss.NewStyle().Foreground(theme.Secondary).Width(10)
	value := lipgloss.NewStyle().Foreground(theme.Accent)

	title := jsonStr(page, "title")
	status := jsonStr(page, "status")
	spaceID := jsonStr(page, "spaceId")
	created := jsonStr(page, "createdAt")

	version := ""
	if v, ok := page["version"].(map[string]any); ok {
		if n, ok := v["number"].(float64); ok {
			version = fmt.Sprintf("%.0f", n)
		}
	}

	webURL := ""
	if links, ok := page["_links"].(map[string]any); ok {
		if w, ok := links["webui"].(string); ok {
			webURL = creds.BaseURL + w
		}
	}

	out.Print("")
	out.Print(header.Render(title))
	out.Print("")
	out.Print(fmt.Sprintf("  %s %s", label.Render("ID"), value.Render(pageID)))
	out.Print(fmt.Sprintf("  %s %s", label.Render("Status"), value.Render(status)))
	out.Print(fmt.Sprintf("  %s %s", label.Render("Version"), value.Render(version)))
	out.Print(fmt.Sprintf("  %s %s", label.Render("Space"), value.Render(spaceID)))
	out.Print(fmt.Sprintf("  %s %s", label.Render("Created"), value.Render(created)))
	out.Print(fmt.Sprintf("  %s %s", label.Render("URL"), value.Render(webURL)))
	out.Print("")
}

func copyToClipboard(data []byte) {
	// Pretty-print before copying
	var pretty map[string]any
	if json.Unmarshal(data, &pretty) == nil {
		if formatted, err := json.MarshalIndent(pretty, "", "  "); err == nil {
			data = formatted
		}
	}

	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(string(data))
	cmd.Run() //nolint: best-effort clipboard copy
}

func jsonStr(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
