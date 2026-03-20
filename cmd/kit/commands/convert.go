package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/coreycoburn/cli-forge/pkg/forge"
	"github.com/spf13/cobra"
)

// ConvertCmd converts files between formats. Currently supports EPS → SVG.
// Requires external tools: ghostscript (gs) and inkscape.
func ConvertCmd() *cobra.Command {
	var optimize bool

	cmd := &cobra.Command{
		Use:   "convert [--optimize] <from-file> <to-file>",
		Short: "Convert files by extension",
		Long:  "Convert files between formats.\n\nSupported conversions:\n  eps → svg    (requires: ghostscript, inkscape)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := forge.OutputFrom(cmd)
			from, to := args[0], args[1]

			// Validate input exists
			if _, err := os.Stat(from); os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", from)
			}

			// Get extensions
			fromExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(from)), ".")
			toExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(to)), ".")
			key := fromExt + "_" + toExt

			if key != "eps_svg" {
				return fmt.Errorf("unsupported conversion: %s → %s (supported: eps → svg)", fromExt, toExt)
			}

			// Check external dependencies
			if err := requireBinary("gs", "brew install ghostscript"); err != nil {
				return err
			}
			if err := requireBinary("inkscape", "brew install --cask inkscape"); err != nil {
				return err
			}

			// Convert
			err := out.Spin(fmt.Sprintf("Converting %s → %s", fromExt, toExt), func() error {
				return convertEPStoSVG(from, to)
			})
			if err != nil {
				return err
			}

			// Verify output was created
			if _, err := os.Stat(to); os.IsNotExist(err) {
				return fmt.Errorf("conversion failed — output file not created")
			}

			// Optimize if requested
			if optimize {
				return runOptimize(out, to)
			}

			if out.IsInteractive() {
				out.Success(fmt.Sprintf("Converted %s → %s", from, to))
			} else {
				return out.JSON(map[string]string{
					"status":  "success",
					"from":    from,
					"to":      to,
					"message": "conversion complete",
				})
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&optimize, "optimize", false, "Optimize output after converting")

	return cmd
}

func convertEPStoSVG(from, to string) error {
	// Create temp PDF for intermediate step
	tmp, err := os.CreateTemp("", "kit_*.pdf")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	// EPS → PDF via ghostscript
	gs := exec.Command("gs",
		"-dNOPAUSE", "-dBATCH", "-dEPSCrop",
		"-sDEVICE=pdfwrite",
		"-sOutputFile="+tmpPath,
		from,
	)
	gs.Stderr = nil
	if err := gs.Run(); err != nil {
		return fmt.Errorf("ghostscript failed: %w", err)
	}

	// PDF → SVG via inkscape
	ink := exec.Command("inkscape",
		tmpPath,
		"--export-filename="+to,
	)
	ink.Stderr = nil
	if err := ink.Run(); err != nil {
		return fmt.Errorf("inkscape failed: %w", err)
	}

	return nil
}
