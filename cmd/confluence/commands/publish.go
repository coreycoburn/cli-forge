package commands

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/coreycoburn/cli-forge/pkg/forge"
	"github.com/spf13/cobra"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// PublishCmd publishes a local directory (markdown + assets) to a Confluence folder.
func PublishCmd() *cobra.Command {
	var folderFlag string
	var dryRun bool
	var excludes []string

	cmd := &cobra.Command{
		Use:   "publish <source-dir>",
		Short: "Publish a directory of markdown + assets to a Confluence folder",
		Long: `Publish a directory tree to a Confluence folder.

For each directory, a Confluence page is created (or updated if one already
exists with the same title under the same parent). Markdown files become child
pages. Non-markdown files become attachments on the containing directory's page.

Idempotent: re-running updates pages in place (version bump) and overwrites
attachments rather than creating duplicates.

The --folder flag accepts either a raw folder ID or a full Atlassian URL, e.g.
  https://you.atlassian.net/wiki/spaces/ABC/folder/1234567`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := forge.OutputFrom(cmd)
			srcDir := args[0]

			absSrc, err := filepath.Abs(srcDir)
			if err != nil {
				return fmt.Errorf("resolve source dir: %w", err)
			}
			info, err := os.Stat(absSrc)
			if err != nil {
				return fmt.Errorf("source dir: %w", err)
			}
			if !info.IsDir() {
				return fmt.Errorf("%s is not a directory", absSrc)
			}

			folderID, err := parseFolderRef(folderFlag)
			if err != nil {
				return err
			}

			creds, err := resolveCredentials(out, false, "")
			if err != nil {
				return err
			}

			// Resolve folder → spaceId.
			var folderMeta folderResponse
			err = out.Spin("Resolving target folder...", func() error {
				return getFolder(creds, folderID, &folderMeta)
			})
			if err != nil {
				return fmt.Errorf("fetch folder %s: %w", folderID, err)
			}

			if out.IsInteractive() {
				out.Info(fmt.Sprintf("Target: %s (folder %s, space %s)", folderMeta.Title, folderID, folderMeta.SpaceID))
			}

			// Build plan.
			root, err := buildPlan(absSrc, excludes)
			if err != nil {
				return fmt.Errorf("build plan: %w", err)
			}

			if out.IsInteractive() {
				out.Header("Plan")
				printPlan(out, root, 0)
				out.Print("")
			}

			if dryRun {
				out.Info("Dry run — nothing was published.")
				return nil
			}

			if out.IsInteractive() {
				fmt.Fprint(os.Stderr, "Proceed? [y/N] ")
				reader := bufio.NewReader(os.Stdin)
				ans, _ := reader.ReadString('\n')
				ans = strings.ToLower(strings.TrimSpace(ans))
				if ans != "y" && ans != "yes" {
					out.Warn("Aborted.")
					return nil
				}
			}

			// Execute.
			ctx := &publishCtx{
				creds:   creds,
				spaceID: folderMeta.SpaceID,
				out:     out,
			}

			// Top-level children of the source dir land directly under the folder.
			for _, child := range root.children {
				if err := ctx.publishNode(child, folderID); err != nil {
					return err
				}
			}

			out.Success(fmt.Sprintf("Published %d page(s), %d attachment(s).", ctx.pagesWritten, ctx.attachmentsWritten))
			return nil
		},
	}

	cmd.Flags().StringVarP(&folderFlag, "folder", "f", "", "Target folder ID or Atlassian URL (required)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show the plan without publishing")
	cmd.Flags().StringSliceVar(&excludes, "exclude", []string{"zips-full.json", ".DS_Store"}, "Filenames or glob patterns to skip")
	_ = cmd.MarkFlagRequired("folder")

	return cmd
}

// ---------------------------------------------------------------------------
// Folder ref parsing
// ---------------------------------------------------------------------------

var folderURLRe = regexp.MustCompile(`/folder/([0-9]+)`)

func parseFolderRef(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("--folder is required")
	}
	if m := folderURLRe.FindStringSubmatch(s); m != nil {
		return m[1], nil
	}
	// Raw ID — must be numeric-ish.
	for _, r := range s {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("could not parse folder ID from %q", s)
		}
	}
	return s, nil
}

// ---------------------------------------------------------------------------
// Plan tree
// ---------------------------------------------------------------------------

type planNode struct {
	title       string     // Confluence page title
	path        string     // absolute path (dir or md file)
	isDir       bool       // true if this node is a directory-page
	children    []planNode // child pages (subdirs + md files)
	attachments []string   // absolute paths of non-md files to attach
}

func buildPlan(srcDir string, excludes []string) (*planNode, error) {
	root := &planNode{
		title: titleFromPath(srcDir, ""),
		path:  srcDir,
		isDir: true,
	}
	if err := populatePlan(root, excludes); err != nil {
		return nil, err
	}
	return root, nil
}

func populatePlan(node *planNode, excludes []string) error {
	entries, err := os.ReadDir(node.path)
	if err != nil {
		return err
	}
	// Deterministic order: dirs first, then files, alphabetical.
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.IsDir() != b.IsDir() {
			return a.IsDir()
		}
		return a.Name() < b.Name()
	})

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if isExcluded(name, excludes) {
			continue
		}
		full := filepath.Join(node.path, name)

		if entry.IsDir() {
			child := planNode{
				title: titleFromDirName(name),
				path:  full,
				isDir: true,
			}
			if err := populatePlan(&child, excludes); err != nil {
				return err
			}
			node.children = append(node.children, child)
			continue
		}

		if strings.EqualFold(filepath.Ext(name), ".md") {
			title, err := titleFromMarkdown(full)
			if err != nil {
				return err
			}
			node.children = append(node.children, planNode{
				title: title,
				path:  full,
				isDir: false,
			})
			continue
		}

		// Non-markdown file → attachment on the containing dir's page.
		node.attachments = append(node.attachments, full)
	}
	return nil
}

func isExcluded(name string, patterns []string) bool {
	for _, p := range patterns {
		if ok, _ := filepath.Match(p, name); ok {
			return true
		}
	}
	return false
}

func titleFromPath(dir, fallback string) string {
	base := filepath.Base(dir)
	if base == "" || base == "." || base == "/" {
		return fallback
	}
	return titleFromDirName(base)
}

func titleFromDirName(name string) string {
	// Strip any extension (shouldn't exist for dirs but safe), normalize separators.
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	return name
}

var h1Re = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)

func titleFromMarkdown(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if m := h1Re.FindSubmatch(data); m != nil {
		return strings.TrimSpace(string(m[1])), nil
	}
	// Fall back to filename stem.
	base := filepath.Base(path)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	return titleFromDirName(stem), nil
}

// ---------------------------------------------------------------------------
// Plan rendering (dry run / preview)
// ---------------------------------------------------------------------------

func printPlan(out *forge.Output, node *planNode, depth int) {
	indent := strings.Repeat("  ", depth)
	marker := "📄"
	if node.isDir {
		marker = "📁"
	}
	if depth > 0 { // skip the synthetic root
		out.Print(fmt.Sprintf("%s%s %s", indent, marker, node.title))
	}
	for _, c := range node.children {
		childDepth := depth + 1
		if depth == 0 {
			childDepth = 0 + 1
		}
		printPlan(out, &c, childDepth)
	}
	for _, a := range node.attachments {
		out.Print(fmt.Sprintf("%s  📎 %s", indent, filepath.Base(a)))
	}
}

// ---------------------------------------------------------------------------
// Publish execution
// ---------------------------------------------------------------------------

type publishCtx struct {
	creds              *credentials
	spaceID            string
	out                *forge.Output
	pagesWritten       int
	attachmentsWritten int
}

func (c *publishCtx) publishNode(node planNode, parentID string) error {
	var body string
	if !node.isDir {
		data, err := os.ReadFile(node.path)
		if err != nil {
			return err
		}
		body, err = markdownToStorage(data)
		if err != nil {
			return fmt.Errorf("render %s: %w", node.path, err)
		}
	} else {
		body = fmt.Sprintf("<p><em>Index page for %s.</em></p>", htmlEscape(node.title))
	}

	pageID, err := c.upsertPage(parentID, node.title, body)
	if err != nil {
		return fmt.Errorf("upsert %q: %w", node.title, err)
	}
	c.pagesWritten++
	c.out.Success(fmt.Sprintf("  page: %s", node.title))

	// Attachments on this page (only applies to dir-pages in this design,
	// but works uniformly if we ever attach to md-pages).
	for _, a := range node.attachments {
		if err := c.uploadAttachment(pageID, a); err != nil {
			return fmt.Errorf("attach %s: %w", a, err)
		}
		c.attachmentsWritten++
		c.out.Success(fmt.Sprintf("    attach: %s", filepath.Base(a)))
	}

	for _, child := range node.children {
		if err := c.publishNode(child, pageID); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Confluence API client
// ---------------------------------------------------------------------------

type folderResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	SpaceID string `json:"spaceId"`
}

func getFolder(creds *credentials, id string, out *folderResponse) error {
	u := fmt.Sprintf("%s/wiki/api/v2/folders/%s", creds.BaseURL, id)
	resp, err := doJSON(creds, "GET", u, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp, out)
}

type pageSummary struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
}

type pagesResponse struct {
	Results []pageSummary `json:"results"`
	Links   struct {
		Next string `json:"next"`
	} `json:"_links"`
}

// findChildPageByTitle looks for an existing child page of parentID with an
// exact title match. Returns (pageID, version, true) if found.
func findChildPageByTitle(creds *credentials, parentID, title string) (string, int, bool, error) {
	// v2 API: pages filtered by parent-id.
	next := fmt.Sprintf("%s/wiki/api/v2/pages?parent-id=%s&limit=250",
		creds.BaseURL, parentID)
	for next != "" {
		resp, err := doJSON(creds, "GET", next, nil)
		if err != nil {
			return "", 0, false, err
		}
		var pr pagesResponse
		if err := json.Unmarshal(resp, &pr); err != nil {
			return "", 0, false, err
		}
		for _, p := range pr.Results {
			if p.Title == title {
				return p.ID, p.Version.Number, true, nil
			}
		}
		if pr.Links.Next == "" {
			return "", 0, false, nil
		}
		next = creds.BaseURL + pr.Links.Next
	}
	return "", 0, false, nil
}

func (c *publishCtx) upsertPage(parentID, title, storageBody string) (string, error) {
	existingID, existingVer, found, err := findChildPageByTitle(c.creds, parentID, title)
	if err != nil {
		return "", fmt.Errorf("lookup existing page: %w", err)
	}

	bodyPayload := map[string]any{
		"representation": "storage",
		"value":          storageBody,
	}

	if found {
		payload := map[string]any{
			"id":     existingID,
			"status": "current",
			"title":  title,
			"body":   bodyPayload,
			"version": map[string]any{
				"number":  existingVer + 1,
				"message": "confluence-cli publish",
			},
		}
		u := fmt.Sprintf("%s/wiki/api/v2/pages/%s", c.creds.BaseURL, existingID)
		b, _ := json.Marshal(payload)
		if _, err := doJSON(c.creds, "PUT", u, b); err != nil {
			return "", fmt.Errorf("update page: %w", err)
		}
		return existingID, nil
	}

	payload := map[string]any{
		"spaceId":  c.spaceID,
		"parentId": parentID,
		"status":   "current",
		"title":    title,
		"body":     bodyPayload,
	}
	u := fmt.Sprintf("%s/wiki/api/v2/pages", c.creds.BaseURL)
	b, _ := json.Marshal(payload)
	resp, err := doJSON(c.creds, "POST", u, b)
	if err != nil {
		return "", fmt.Errorf("create page: %w", err)
	}
	var created pageSummary
	if err := json.Unmarshal(resp, &created); err != nil {
		return "", err
	}
	return created.ID, nil
}

// ---------------------------------------------------------------------------
// Attachments (v1 REST API — Confluence has no v2 attachment upload yet)
// ---------------------------------------------------------------------------

type v1Attachment struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type v1AttachmentsResponse struct {
	Results []v1Attachment `json:"results"`
}

func (c *publishCtx) uploadAttachment(pageID, filePath string) (retErr error) {
	name := filepath.Base(filePath)
	existingID, err := findAttachment(c.creds, pageID, name)
	if err != nil {
		return fmt.Errorf("list attachments: %w", err)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", name)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, f); err != nil {
		return err
	}
	_ = mw.WriteField("minorEdit", "true")
	_ = mw.WriteField("comment", "confluence-cli publish")
	if err := mw.Close(); err != nil {
		return err
	}

	var u string
	if existingID != "" {
		// Update existing attachment data.
		u = fmt.Sprintf("%s/wiki/rest/api/content/%s/child/attachment/%s/data",
			c.creds.BaseURL, pageID, existingID)
	} else {
		u = fmt.Sprintf("%s/wiki/rest/api/content/%s/child/attachment",
			c.creds.BaseURL, pageID)
	}

	req, err := http.NewRequest("POST", u, &body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.creds.Email, c.creds.APIToken)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-Atlassian-Token", "no-check")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func findAttachment(creds *credentials, pageID, filename string) (string, error) {
	u := fmt.Sprintf("%s/wiki/rest/api/content/%s/child/attachment?filename=%s&limit=250",
		creds.BaseURL, pageID, url.QueryEscape(filename))
	resp, err := doJSON(creds, "GET", u, nil)
	if err != nil {
		return "", err
	}
	var ar v1AttachmentsResponse
	if err := json.Unmarshal(resp, &ar); err != nil {
		return "", err
	}
	for _, a := range ar.Results {
		if a.Title == filename {
			return a.ID, nil
		}
	}
	return "", nil
}

// ---------------------------------------------------------------------------
// HTTP helper
// ---------------------------------------------------------------------------

func doJSON(creds *credentials, method, urlStr string, body []byte) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, urlStr, reader)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(creds.Email, creds.APIToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 400))
	}
	return respBody, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ---------------------------------------------------------------------------
// Markdown → Confluence storage format
// ---------------------------------------------------------------------------

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Strikethrough,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithXHTML(),
		html.WithUnsafe(), // allow raw HTML embedded in markdown to pass through
	),
)

func markdownToStorage(src []byte) (string, error) {
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func htmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return r.Replace(s)
}
