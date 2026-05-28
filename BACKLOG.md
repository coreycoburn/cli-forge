# Backlog

Future enhancements, deferred deliberately. Each entry notes what we ship now
and what "better" looks like later, so the trade-off isn't lost.

## kit trace `--smart-text`: ML-based font matching ("best" version)

**Now (good):** font matching is pure-Go + Tesseract — render each candidate
font in Go and score it against the OCR'd text crop with classical image
similarity. No Python/ML. Chosen because kit is a static Go binary that shells
out to small brew CLIs, and the variant-grid UX (human picks from rendered
options) only needs a *reasonable shortlist*, not exact identification.

**Later (best):** replace the classical scorer with **vision-embedding
matching** — embed the source text crop and each rendered candidate with a CLIP
(or similar) model and rank by cosine similarity. More robust to size,
anti-aliasing, and stylistic variation; produces a better-ordered shortlist.

**Why deferred:** drags Python + torch + a model into a tool whose identity is
"one static Go binary + brew CLIs"; and the dev machine's Python 3.14 likely
has no torch wheels yet. Revisit when either (a) kit gains a sanctioned way to
shell out to a heavier helper, or (b) a small, dependency-light embedding
binary exists that fits the brew-CLI model.

**Pointer:** the matching step is the only part that changes — OCR, category
prune, variant generation, and the pick-from-grid UX stay identical.

## kit retext: Google Fonts corpus (`--google`)

**Now:** match only against fonts installed locally (`fc-list`), with `--fonts`
to constrain to a known set. "Closest *installed* font."

**Later:** a `--google` flag that widens the corpus to the ~1,700 Google Fonts
families. Preferred fetch: download the TTFs directly from the `google/fonts`
GitHub repo (OFL/Apache, plain files the existing renderer handles) and cache
them. Fallback if direct download proves fiddly (woff2-only, naming): render via
a headless browser against the live Google Fonts CSS. Pairs with the ML-matching
item — both are "make matching better / broader."

## kit retext: weight-bias in classical matching

**Known limitation, not yet addressed.** IoU on binarized glyphs is dominated by
black-pixel mass, so heavier (Bold/Black) fonts systematically rank higher than
their lighter counterparts even when the source is a medium/regular weight. On
the prototype's `BEHAVIORAL HEALTH` tagline, Arial Black tops the sans shortlist
while the source visibly reads as a lighter tracked sans.

**Possible fixes:**
- Estimate source stroke density (foreground-pixel ratio over the glyph
  bounding box) and prefer candidates whose rendered density matches.
- Score on edge/contour overlap rather than filled-pixel IoU — sensitive to
  shape, not mass.
- Both. The ML/embedding matcher (top item) would dissolve this naturally.

Worth fixing in the classical version if real-world inputs keep producing wrong
weights; otherwise this is dissolved by the ML upgrade.

## kit retext: `--blocks` segmentation override

`retext` auto-groups OCR words into logical text runs (one line of same-style
text = one block), numbered top-to-bottom; `--picks` indexes those blocks. If
OCR mis-groups (e.g. splits a tagline across two blocks, or merges two), the
pick indices shift and there's no recourse. Add a `--blocks` override to let the
user correct the grouping (merge/split). Defer until it actually bites.
