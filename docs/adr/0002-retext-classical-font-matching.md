# Retext font matching: classical, human-reviewed variants (no ML)

`kit retext` detects raster text in an image and re-sets it as outlined glyphs
from a matched font. Font matching is **classical**: render each locally
installed candidate font as the OCR'd string (Tesseract for OCR) and rank by
image similarity to the source crop — not ML/embedding-based.

**Why no ML, even though accuracy is the stated priority.** The accuracy
mechanism is *the human reviewing rendered variants*, not the auto-matcher. Our
inputs are logos — often AI-generated, where the lettering isn't a real typeface
at all — so there is no ground-truth font and "closest" is the ceiling. A person
judges "closest" better than any algorithm. The matcher therefore only needs to
surface a *reasonable shortlist*; the human picks from a variant grid. ML
(embedding) matching would improve the shortlist *ordering* but not raise that
ceiling, so it does not justify dragging Python + torch + a model into a tool
whose identity is a static Go binary plus a few brew CLIs — especially with no
torch wheels for the dev machine's Python 3.14. ML matching and a broader
(Google Fonts) corpus are recorded in BACKLOG.md as the "best" version.

## Consequences

- The default surfaces a small per-block grid of variants for review
  (`--variants`, default 3) plus a best-match assembled SVG; `--picks` assembles
  a chosen combination. Accuracy comes from review, so the default is **not** a
  blind single pick. (`--variants 1` opts into a single best-match file.)
- Matching corpus is the locally installed fonts; `--fonts` constrains it.
- Replaced text is always **outlined** — self-contained, no font dependency at
  render time. No live `<text>`, no embedded `@font-face`.
- `retext` is its own command, composing `trace`; it does not overload `trace`'s
  single-file contract.
