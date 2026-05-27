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

// TraceCmd reconstructs a raster image (PNG) as a vector SVG via vtracer.
// Unlike convert (a faithful format translation), trace is a lossy
// reconstruction — it approximates a pixel grid with vector paths. See
// CONTEXT.md for the convert/trace distinction. vtracer is lazily installed
// on first use.
func TraceCmd() *cobra.Command {
	var (
		mode      string
		detail    string
		gradients bool
		optimize  bool
	)

	cmd := &cobra.Command{
		Use:     "trace [flags] <png-file> <svg-file>",
		Aliases: []string{"vectorize"},
		Short:   "Trace a raster image into vector SVG",
		Long: "Reconstruct a raster image (PNG) as a vector SVG.\n\n" +
			"Trace is a lossy reconstruction, not a faithful conversion — it\n" +
			"approximates pixels with vector paths (requires: vtracer).\n\n" +
			"Output is flat-color by default — ideal for logos. vtracer fakes\n" +
			"gradients as many stacked color bands, which bloats the file; pass\n" +
			"--gradients only when you want that photographic banding.\n\n" +
			"Flags:\n" +
			"  --mode       color (default) or bw\n" +
			"  --detail     low | med | high  (default med)\n" +
			"  --gradients  reconstruct gradients as color bands (default: flat)\n" +
			"  --optimize   optimize the SVG after tracing",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := forge.OutputFrom(cmd)
			from, to := args[0], args[1]

			if _, err := os.Stat(from); os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", from)
			}

			fromExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(from)), ".")
			toExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(to)), ".")
			if fromExt != "png" || toExt != "svg" {
				return fmt.Errorf("unsupported trace: %s → %s (supported: png → svg)", fromExt, toExt)
			}

			if mode != "color" && mode != "bw" {
				return fmt.Errorf("invalid --mode %q (want: color, bw)", mode)
			}
			speckle, precision, err := detailPreset(detail)
			if err != nil {
				return err
			}

			// Lazily ensure vtracer — confirm-then-install when interactive,
			// hint-and-exit otherwise.
			if err := ensureBinary(out, "vtracer", "coreycoburn/tap/vtracer"); err != nil {
				return err
			}

			err = out.Spin(fmt.Sprintf("Tracing %s → %s", fromExt, toExt), func() error {
				return tracePNGtoSVG(from, to, mode, speckle, precision, gradients)
			})
			if err != nil {
				return err
			}

			if _, err := os.Stat(to); os.IsNotExist(err) {
				return fmt.Errorf("trace failed — output file not created")
			}

			if optimize {
				return runOptimize(out, to)
			}

			if out.IsInteractive() {
				out.Success(fmt.Sprintf("Traced %s → %s", from, to))
			} else {
				return out.JSON(map[string]string{
					"status":  "success",
					"from":    from,
					"to":      to,
					"message": "trace complete",
				})
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "color", "Color mode: color or bw")
	cmd.Flags().StringVar(&detail, "detail", "med", "Detail level: low, med, or high")
	cmd.Flags().BoolVar(&gradients, "gradients", false, "Reconstruct gradients as color bands (default: flat)")
	cmd.Flags().BoolVar(&optimize, "optimize", false, "Optimize the SVG after tracing")

	return cmd
}

// detailPreset maps the plain-language --detail level onto vtracer's
// filter_speckle (discard patches smaller than N px) and color_precision
// (significant bits per RGB channel) knobs. Higher detail keeps more colors
// and removes fewer speckles.
func detailPreset(level string) (speckle, precision int, err error) {
	switch level {
	case "low":
		return 10, 4, nil
	case "med":
		return 8, 6, nil
	case "high":
		return 2, 8, nil
	default:
		return 0, 0, fmt.Errorf("invalid --detail %q (want: low, med, high)", level)
	}
}

func tracePNGtoSVG(from, to, mode string, speckle, precision int, gradients bool) error {
	// vtracer reconstructs gradients as stacked flat-color bands; a larger
	// gradient_step collapses those bands, giving clean flat output suited to
	// logos. Flat is the default; --gradients drops to vtracer's smaller step
	// for the banded, photographic look.
	gradientStep := 32
	if gradients {
		gradientStep = 16
	}

	vt := exec.Command("vtracer",
		"--input", from,
		"--output", to,
		"--colormode", mode,
		"--filter_speckle", fmt.Sprintf("%d", speckle),
		"--color_precision", fmt.Sprintf("%d", precision),
		"--gradient_step", fmt.Sprintf("%d", gradientStep),
		// Fit smoother splines than vtracer's defaults (corner 60, segment 4):
		// the blemishes in traced logos are mostly wobbly edges, and smoother
		// curves clean them up while preserving fine detail like small text.
		// corner_threshold 70 still keeps intentionally sharp corners.
		"--corner_threshold", "70",
		"--segment_length", "8",
	)
	vt.Stderr = nil
	if err := vt.Run(); err != nil {
		return fmt.Errorf("vtracer failed: %w", err)
	}
	return nil
}
