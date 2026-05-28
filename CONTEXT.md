# kit

`kit` is a design-asset toolkit: a CLI for converting, tracing, and optimizing
image and vector files. This glossary defines the operations it performs and,
in particular, the distinctions between operations that look similar but make
different promises about fidelity.

## Language

**Convert**:
A _faithful_ translation of a file from one format to another, preserving the
original's fidelity (e.g. EPS → SVG, where both sides are vector). The output is
a re-encoding of the same information, not a reinterpretation.
_Avoid_: transform, export

**Trace**:
A _lossy reconstruction_ of a raster image as vector paths — approximating a
pixel grid with shapes and curves (e.g. PNG → SVG). The output is a best-effort
guess and is never an exact reproduction of the source.
_Avoid_: convert (a trace is not faithful — they are different operations).
`vectorize` is an accepted alias.

**Retext**:
Detecting raster text in an image and re-setting it as clean glyph outlines from
a matched font, instead of leaving it as rough traced paths. A re-typeset, not a
pixel reconstruction — distinct from **Trace**, which it builds on. The output is
self-contained (outlined paths, no font dependency).
_Avoid_: detext, OCR-replace

**Optimize**:
An _in-place_ reduction of a file's size, with no change to format and no
perceptible change to appearance (SVG minification, PNG quantization and
lossless compression).
_Avoid_: compress, shrink, minify (these name techniques, not the operation)

**Design asset**:
An image or vector file that kit operates on — currently PNG, SVG, and EPS.

## Example dialogue

**Dev:** "Can I convert this logo PNG to an SVG?"

**Maintainer:** "Not convert — trace. Convert is for faithful format swaps like
EPS to SVG, where both sides are vector and nothing is lost. A PNG has no paths
to preserve, so we _trace_ it: reconstruct vector shapes that approximate the
pixels. The result is a guess, which is exactly why it's `kit trace` and not
`kit convert`."

**Dev:** "And if the traced SVG comes out noisy?"

**Maintainer:** "Raise `--detail`, or run `--optimize` afterwards to tidy the
SVG. Optimize won't change how it looks — it only shrinks the file."
