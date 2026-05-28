package commands

import (
	"bufio"
	"fmt"
	"image"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/coreycoburn/cli-forge/pkg/forge"
	"github.com/spf13/cobra"
)

// Block is a logical text run detected in a raster image: a merged bounding
// box, the OCR'd text, and the median word height (a proxy for font size).
// Blocks are ordered top-to-bottom in reading order.
type Block struct {
	Text   string
	Box    image.Rectangle // pixel coordinates in the source image
	Height int             // median word height (px)
}

// RetextCmd detects raster text in an image and re-sets it as outlined glyphs
// from a matched font. Composes trace internally. See CONTEXT.md (Retext) and
// docs/adr/0002 for the design.
//
// MVP scope: OCR + block grouping. Font matching, glyph outlining, and the
// final SVG splice are layered in subsequent commits.
func RetextCmd() *cobra.Command {
	var (
		variants int
		picks    string
		fonts    string
	)

	cmd := &cobra.Command{
		Use:   "retext [flags] <input.png> <output>",
		Short: "Detect raster text and re-set it as outlined real type",
		Long: "Detect raster text via OCR, match it to the closest installed\n" +
			"font, and re-set the text as outlined glyph paths in the final\n" +
			"SVG — no font dependency at render time. Distinct from trace,\n" +
			"which reconstructs raster as pixel-following paths.\n\n" +
			"Flags:\n" +
			"  --variants  N candidates per text block (default 1 = single best)\n" +
			"  --picks     positional list of grid indices, e.g. 3,1\n" +
			"  --fonts     constrain matching to a comma-separated font list",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := forge.OutputFrom(cmd)
			src, dst := args[0], args[1]
			_ = dst // wired in once assembly lands

			if _, err := os.Stat(src); os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", src)
			}
			if err := requireBinary("tesseract", "brew install tesseract"); err != nil {
				return err
			}

			blocks, imgSize, err := runOCR(src)
			if err != nil {
				return err
			}

			// MVP report. Matching + assembly come next.
			if out.IsInteractive() {
				out.Info(fmt.Sprintf("image: %dx%d   blocks derived: %d",
					imgSize.X, imgSize.Y, len(blocks)))
				for i, b := range blocks {
					out.Info(fmt.Sprintf("  block %d: %q  box=(%d,%d,%d,%d)  h≈%dpx",
						i+1, b.Text, b.Box.Min.X, b.Box.Min.Y,
						b.Box.Dx(), b.Box.Dy(), b.Height))
				}
			} else {
				type jsonBlock struct {
					Text string `json:"text"`
					X    int    `json:"x"`
					Y    int    `json:"y"`
					W    int    `json:"w"`
					H    int    `json:"h"`
				}
				jb := make([]jsonBlock, len(blocks))
				for i, b := range blocks {
					jb[i] = jsonBlock{b.Text, b.Box.Min.X, b.Box.Min.Y, b.Box.Dx(), b.Box.Dy()}
				}
				return out.JSON(map[string]any{
					"status": "blocks-detected",
					"image":  map[string]int{"w": imgSize.X, "h": imgSize.Y},
					"blocks": jb,
					"phase":  "MVP — matching and assembly not yet wired",
				})
			}

			_ = variants
			_ = picks
			_ = fonts
			return nil
		},
	}

	cmd.Flags().IntVar(&variants, "variants", 1, "N candidates per block (default 1)")
	cmd.Flags().StringVar(&picks, "picks", "", "Positional list of grid indices (e.g. 3,1)")
	cmd.Flags().StringVar(&fonts, "fonts", "", "Comma-separated font names to constrain matching")
	return cmd
}

// runOCR runs Tesseract in word-level TSV mode and returns the derived text
// blocks plus the detected image size.
func runOCR(src string) ([]Block, image.Point, error) {
	cmd := exec.Command("tesseract", src, "stdout", "tsv")
	cmd.Stderr = nil
	raw, err := cmd.Output()
	if err != nil {
		return nil, image.Point{}, fmt.Errorf("tesseract failed: %w", err)
	}
	return parseTSVBlocks(raw)
}

// parseTSVBlocks turns Tesseract's word-level TSV into [Block]s by grouping
// surviving words on Tesseract's (block,par,line) hierarchy. Filters: confidence
// floor, relative-height decoration cutoff (drops hairline rules like `—`),
// and pure-punctuation tokens. The thresholds are derived from the image's own
// dimensions — nothing image-specific is hardcoded.
func parseTSVBlocks(raw []byte) ([]Block, image.Point, error) {
	const confMin = 30.0

	s := bufio.NewScanner(strings.NewReader(string(raw)))
	s.Buffer(make([]byte, 1<<20), 1<<20)

	type row struct {
		level, blockN, parN, lineN  int
		left, top, width, height    int
		conf                        float64
		text                        string
	}

	var hdr []string
	ix := map[string]int{}
	var rows []row
	var imgSize image.Point

	first := true
	for s.Scan() {
		line := s.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		cols := strings.Split(line, "\t")
		if first {
			hdr = cols
			for i, n := range hdr {
				ix[n] = i
			}
			first = false
			continue
		}
		if len(cols) < len(hdr) {
			continue
		}
		atoi := func(k string) int { n, _ := strconv.Atoi(cols[ix[k]]); return n }
		atof := func(k string) float64 { f, _ := strconv.ParseFloat(cols[ix[k]], 64); return f }
		r := row{
			level: atoi("level"), blockN: atoi("block_num"),
			parN: atoi("par_num"), lineN: atoi("line_num"),
			left: atoi("left"), top: atoi("top"),
			width: atoi("width"), height: atoi("height"),
			conf: atof("conf"),
			text: strings.TrimSpace(cols[ix["text"]]),
		}
		if r.level == 1 {
			imgSize = image.Point{X: r.width, Y: r.height}
		}
		rows = append(rows, r)
	}
	if err := s.Err(); err != nil {
		return nil, imgSize, err
	}

	minH := imgSize.Y / 100
	if minH < 8 {
		minH = 8
	}

	// Group surviving words by Tesseract's line hierarchy.
	type wordEntry struct {
		text          string
		l, t, w, h    int
		insertionOrd  int // preserves Tesseract reading order within a line
	}
	groups := map[string][]wordEntry{}
	order := []string{}
	for i, r := range rows {
		if r.text == "" || r.conf < confMin {
			continue
		}
		if r.height < minH {
			continue
		}
		if isPunctOnly(r.text) {
			continue
		}
		key := fmt.Sprintf("%d/%d/%d", r.blockN, r.parN, r.lineN)
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], wordEntry{r.text, r.left, r.top, r.width, r.height, i})
	}

	var blocks []Block
	for _, k := range order {
		ws := groups[k]
		if len(ws) == 0 {
			continue
		}
		minX, minY := ws[0].l, ws[0].t
		maxX, maxY := minX+ws[0].w, minY+ws[0].h
		heights := make([]int, 0, len(ws))
		parts := make([]string, 0, len(ws))
		for _, w := range ws {
			if w.l < minX {
				minX = w.l
			}
			if w.t < minY {
				minY = w.t
			}
			if w.l+w.w > maxX {
				maxX = w.l + w.w
			}
			if w.t+w.h > maxY {
				maxY = w.t + w.h
			}
			heights = append(heights, w.h)
			parts = append(parts, w.text)
		}
		sort.Ints(heights)
		blocks = append(blocks, Block{
			Text:   strings.Join(parts, " "),
			Box:    image.Rect(minX, minY, maxX, maxY),
			Height: heights[len(heights)/2],
		})
	}

	// Reading order: top-to-bottom by box's top edge.
	sort.SliceStable(blocks, func(i, j int) bool {
		return blocks[i].Box.Min.Y < blocks[j].Box.Min.Y
	})

	return blocks, imgSize, nil
}

// isPunctOnly reports whether every rune in s is dash/punctuation — used to
// drop decorative tokens (em-dash rules around taglines etc.) that Tesseract
// detects as text but aren't real type.
func isPunctOnly(s string) bool {
	const punct = "—-_·.•"
	for _, r := range s {
		if !strings.ContainsRune(punct, r) {
			return false
		}
	}
	return true
}
