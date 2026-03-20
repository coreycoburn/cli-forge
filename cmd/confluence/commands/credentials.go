package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

const credentialsRelPath = ".config/confluence/credentials"

type credentials struct {
	BaseURL  string
	Email    string
	APIToken string
}

// loadCredentials reads credentials from environment variables first,
// then falls back to the credentials file at ~/.config/confluence/credentials.
// Compatible with the bash export format from the old CLI.
func loadCredentials() (*credentials, error) {
	creds := &credentials{
		BaseURL:  os.Getenv("CONFLUENCE_BASE_URL"),
		Email:    os.Getenv("CONFLUENCE_EMAIL"),
		APIToken: os.Getenv("CONFLUENCE_API_TOKEN"),
	}

	if creds.isComplete() {
		creds.BaseURL = strings.TrimRight(creds.BaseURL, "/")
		return creds, nil
	}

	// Fall back to credentials file
	path := credentialsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("credentials not configured; run 'confluence get --setup' first")
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "export ")
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := strings.Trim(parts[1], `"`)
		switch key {
		case "CONFLUENCE_BASE_URL":
			creds.BaseURL = val
		case "CONFLUENCE_EMAIL":
			creds.Email = val
		case "CONFLUENCE_API_TOKEN":
			creds.APIToken = val
		}
	}

	creds.BaseURL = strings.TrimRight(creds.BaseURL, "/")

	if !creds.isComplete() {
		return nil, fmt.Errorf("credentials not configured; run 'confluence get --setup' first")
	}

	return creds, nil
}

// setupCredentials prompts the user for Confluence credentials and saves them.
func setupCredentials() (*credentials, error) {
	fmt.Println("Configure your Atlassian credentials.")
	fmt.Println("Get your API token at: https://id.atlassian.com/manage-profile/security/api-tokens")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Confluence Base URL (e.g. https://yoursite.atlassian.net): ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)
	baseURL = strings.TrimRight(baseURL, "/")

	fmt.Print("Atlassian Email: ")
	email, _ := reader.ReadString('\n')
	email = strings.TrimSpace(email)

	fmt.Print("API Token: ")
	tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("failed to read token: %w", err)
	}
	fmt.Println()
	token := strings.TrimSpace(string(tokenBytes))

	creds := &credentials{
		BaseURL:  baseURL,
		Email:    email,
		APIToken: token,
	}

	if !creds.isComplete() {
		return nil, fmt.Errorf("all fields are required")
	}

	// Save to file
	path := credentialsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	content := fmt.Sprintf("export CONFLUENCE_BASE_URL=\"%s\"\nexport CONFLUENCE_EMAIL=\"%s\"\nexport CONFLUENCE_API_TOKEN=\"%s\"\n",
		creds.BaseURL, creds.Email, creds.APIToken)

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return nil, err
	}

	return creds, nil
}

func (c *credentials) isComplete() bool {
	return c.BaseURL != "" && c.Email != "" && c.APIToken != ""
}

func credentialsPath() string {
	return filepath.Join(os.Getenv("HOME"), credentialsRelPath)
}
