# cli-forge

Go monorepo for self-contained, distributable CLIs with a shared themed output
layer. The primary tool is **`kit`** — a design-asset toolkit for converting
and optimizing files.

## Install

```bash
brew tap coreycoburn/tap
brew install kit
```

Brew installs `ghostscript`, `pngquant`, and `oxipng` automatically as runtime
dependencies. `kit convert` additionally requires Inkscape (not auto-installed
because it's a cask):

```bash
brew install --cask inkscape
```

## `kit optimize`

Optimize a file in-place. Reports size change in both raw and gzipped bytes.

```
kit optimize <file> [--lossless]
```

### Supported formats

| Format | Pipeline                                             | External deps       |
| ------ | ---------------------------------------------------- | ------------------- |
| `.svg` | minify, strip dimensions, colors → `currentColor`, ensure `xmlns` | none (pure Go) |
| `.png` | pngquant (lossy palette, quality 80–95, strip metadata) → oxipng (lossless, `-o max`) | pngquant, oxipng |

### Flags

| Flag           | Purpose                                                            |
| -------------- | ------------------------------------------------------------------ |
| `--lossless`   | PNG only — skip pngquant, run oxipng only (guaranteed no visual change) |
| `--json`       | Machine-readable output (global)                                   |

### Examples

```bash
# Single file
kit optimize logo.svg
kit optimize favicon.png

# Whole directory — shell loop
for f in static/favicons/*.png; do kit optimize "$f"; done

# Lossless PNG (for photos / pixel-perfect assets)
kit optimize photo.png --lossless

# JSON output for scripting / CI
kit optimize logo.svg --json
```

### When to use `--lossless`

The default PNG pipeline is lossy — pngquant quantizes to a 256-color palette
at quality 80–95, which is visually indistinguishable for logos, icons, UI
screenshots, and favicons (typical savings: 40–80%). Use `--lossless` for:

- Photographs with smooth gradients
- Design source-of-truth assets where downstream tools re-encode
- Any PNG where every pixel must match the original

Lossless-only savings are more modest (typically 5–15%), all from oxipng's
deflate re-pass.

### SVG notes

`kit optimize` converts fill/stroke colors to `currentColor` so the parent
element's `color` CSS property controls rendering. This only takes effect when
the SVG is **inlined** into HTML (e.g. `{@html svgString}` in Svelte, or
server-rendered). It has no effect when referenced via `<img src="logo.svg">`.

## `kit convert`

Convert between file formats.

```
kit convert [--optimize] <from-file> <to-file>
```

### Supported conversions

| From  | To    | Pipeline                                  | External deps          |
| ----- | ----- | ----------------------------------------- | ---------------------- |
| `eps` | `svg` | ghostscript (EPS → PDF) → Inkscape (PDF → SVG) | ghostscript, Inkscape  |

### Examples

```bash
kit convert logo.eps logo.svg
kit convert logo.eps logo.svg --optimize   # convert + SVG optimize pass
```

## Global flags

| Flag        | Purpose                                           |
| ----------- | ------------------------------------------------- |
| `--json`    | Output as JSON (every command, every subcommand)  |
| `-v, --version` | Print version                                 |
| `-h, --help`    | Help for any command                          |

## Other CLIs in this repo

- **`confluence`** — personal Confluence CLI for fetching pages. Install via
  `brew install coreycoburn/tap/confluence`.

## Development

See [`CLAUDE.md`](./CLAUDE.md) for the framework internals (the `pkg/forge`
output/theme/spinner layer) and for how to add a new CLI or command.

Quick build:

```bash
make build           # build all CLIs → ./bin/
make install         # install all CLIs → $GOPATH/bin
go build ./cmd/kit   # build just kit
```

Releases are cut by tag — GoReleaser builds multi-platform binaries, uploads
them to GitHub Releases, and pushes formulae to `coreycoburn/homebrew-tap`.
