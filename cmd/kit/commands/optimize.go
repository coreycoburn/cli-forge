package commands

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/coreycoburn/cli-forge/pkg/forge"
	"github.com/spf13/cobra"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/svg"
)

// OptimizeCmd optimizes files in-place. Currently supports SVG.
// Fully self-contained — no external dependencies required.
func OptimizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "optimize <file>",
		Short: "Optimize a file in-place",
		Long:  "Optimize files in-place.\n\nSupported formats:\n  svg    (minify, remove dimensions, colors → currentColor, ensure xmlns)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := forge.OutputFrom(cmd)
			file := args[0]

			// Validate file exists
			if _, err := os.Stat(file); os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", file)
			}

			// Check supported format
			ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(file)), ".")
			if ext != "svg" {
				return fmt.Errorf("unsupported format: %s (supported: svg)", ext)
			}

			// Read file
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}

			// Capture before sizes
			rawBefore := len(data)
			gzBefore := gzipSize(data)

			// Optimize
			var optimized []byte
			err = out.Spin("Optimizing "+filepath.Base(file), func() error {
				var err error
				optimized, err = optimizeSVG(data)
				return err
			})
			if err != nil {
				return err
			}

			// Write back
			if err := os.WriteFile(file, optimized, 0644); err != nil {
				return err
			}

			// Capture after sizes
			rawAfter := len(optimized)
			gzAfter := gzipSize(optimized)

			if out.IsInteractive() {
				renderTable(out, filepath.Base(file), rawBefore, rawAfter, gzBefore, gzAfter)
			} else {
				return out.JSON(map[string]any{
					"status":  "success",
					"file":    file,
					"message": "optimization complete",
					"raw":     map[string]int{"before": rawBefore, "after": rawAfter},
					"gzip":    map[string]int{"before": gzBefore, "after": gzAfter},
				})
			}

			return nil
		},
	}

	return cmd
}

// runOptimize is called by the convert command's --optimize flag.
func runOptimize(out *forge.Output, file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	rawBefore := len(data)
	gzBefore := gzipSize(data)

	var optimized []byte
	err = out.Spin("Optimizing "+filepath.Base(file), func() error {
		var err error
		optimized, err = optimizeSVG(data)
		return err
	})
	if err != nil {
		return err
	}

	if err := os.WriteFile(file, optimized, 0644); err != nil {
		return err
	}

	rawAfter := len(optimized)
	gzAfter := gzipSize(optimized)

	if out.IsInteractive() {
		renderTable(out, filepath.Base(file), rawBefore, rawAfter, gzBefore, gzAfter)
	} else {
		return out.JSON(map[string]any{
			"status":  "success",
			"file":    file,
			"message": "conversion and optimization complete",
			"raw":     map[string]int{"before": rawBefore, "after": rawAfter},
			"gzip":    map[string]int{"before": gzBefore, "after": gzAfter},
		})
	}

	return nil
}

// optimizeSVG applies all SVG optimization transforms and minification.
func optimizeSVG(data []byte) ([]byte, error) {
	s := string(data)

	// 1. Remove <script> elements (security — logos shouldn't have scripts)
	s = removeScripts(s)

	// 2. Convert inline styles to presentation attributes (so color replacement catches them)
	s = convertStyleToAttrs(s)

	// 3. Replace fill/stroke colors with currentColor (allows CSS color control)
	s = replaceColorsWithCurrentColor(s)

	// 4. Remove width/height from <svg> (keep viewBox for responsive scaling)
	s = removeDimensions(s)

	// 5. Replace deprecated xlink:href with href (SVG 2)
	s = removeXlink(s)

	// 6. Ensure xmlns attribute (required for <img src="*.svg">)
	s = ensureXMLNS(s)

	// 7. Minify with multipass until output stabilizes
	s, err := minifySVG(s)
	if err != nil {
		return nil, err
	}

	return []byte(s), nil
}

func removeScripts(s string) string {
	re := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	return re.ReplaceAllString(s, "")
}

func convertStyleToAttrs(s string) string {
	re := regexp.MustCompile(`\bstyle="([^"]*)"`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		content := re.FindStringSubmatch(match)[1]

		var attrs []string
		var remaining []string

		for _, prop := range strings.Split(content, ";") {
			prop = strings.TrimSpace(prop)
			if prop == "" {
				continue
			}
			parts := strings.SplitN(prop, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			switch key {
			case "fill", "stroke", "fill-opacity", "stroke-opacity", "stroke-width":
				attrs = append(attrs, fmt.Sprintf(`%s="%s"`, key, val))
			default:
				remaining = append(remaining, prop)
			}
		}

		result := strings.Join(attrs, " ")
		if len(remaining) > 0 {
			if result != "" {
				result += " "
			}
			result += fmt.Sprintf(`style="%s"`, strings.Join(remaining, ";"))
		}
		return result
	})
}

func replaceColorsWithCurrentColor(s string) string {
	re := regexp.MustCompile(`\b(fill|stroke)="([^"]*)"`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		parts := re.FindStringSubmatch(match)
		attr := parts[1]
		val := parts[2]

		// Don't replace special values
		switch val {
		case "none", "currentColor", "transparent", "inherit":
			return match
		}
		if strings.HasPrefix(val, "url(") {
			return match
		}

		return fmt.Sprintf(`%s="currentColor"`, attr)
	})
}

func removeDimensions(s string) string {
	// Only remove width/height from the root <svg> element
	svgTagRe := regexp.MustCompile(`(?i)(<svg\b[^>]*>)`)
	return svgTagRe.ReplaceAllStringFunc(s, func(tag string) string {
		dimRe := regexp.MustCompile(`\s+(?:width|height)="[^"]*"`)
		return dimRe.ReplaceAllString(tag, "")
	})
}

func removeXlink(s string) string {
	s = strings.ReplaceAll(s, "xlink:href", "href")
	if !strings.Contains(s, "xlink:") {
		nsRe := regexp.MustCompile(`\s+xmlns:xlink="[^"]*"`)
		s = nsRe.ReplaceAllString(s, "")
	}
	return s
}

func ensureXMLNS(s string) string {
	if strings.Contains(s, `xmlns=`) {
		return s
	}
	return strings.Replace(s, "<svg ", `<svg xmlns="http://www.w3.org/2000/svg" `, 1)
}

func minifySVG(s string) (string, error) {
	m := minify.New()
	m.AddFunc("image/svg+xml", svg.Minify)

	// Multipass: run until output stabilizes (max 10 passes)
	for range 10 {
		result, err := m.String("image/svg+xml", s)
		if err != nil {
			return "", err
		}
		if result == s {
			break
		}
		s = result
	}
	return s, nil
}

func gzipSize(data []byte) int {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(data)
	w.Close()
	return buf.Len()
}

func renderTable(out *forge.Output, filename string, rawBefore, rawAfter, gzBefore, gzAfter int) {
	theme := out.Theme()

	header := lipgloss.NewStyle().Bold(true).Foreground(theme.Primary)
	label := lipgloss.NewStyle().Foreground(theme.Secondary)
	value := lipgloss.NewStyle().Foreground(theme.Accent)
	change := lipgloss.NewStyle().Foreground(theme.Success)

	out.Print("")
	out.Print(header.Render(filename))
	out.Print("")
	out.Print(fmt.Sprintf("  %s  %s  %s  %s",
		label.Width(6).Render("Mode"),
		label.Width(10).Render("Before"),
		label.Width(10).Render("After"),
		label.Width(10).Render("Change"),
	))
	out.Print(fmt.Sprintf("  %s  %s  %s  %s",
		value.Width(6).Render("Raw"),
		value.Width(10).Render(humanBytes(rawBefore)),
		value.Width(10).Render(humanBytes(rawAfter)),
		change.Width(10).Render(pctChange(rawBefore, rawAfter)+"%"),
	))
	out.Print(fmt.Sprintf("  %s  %s  %s  %s",
		value.Width(6).Render("Gzip"),
		value.Width(10).Render(humanBytes(gzBefore)),
		value.Width(10).Render(humanBytes(gzAfter)),
		change.Width(10).Render(pctChange(gzBefore, gzAfter)+"%"),
	))
	out.Print("")
	out.Success("Optimization complete")
}
