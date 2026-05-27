# PNG→SVG tracing via vtracer, repackaged in our tap and lazily installed

`kit trace` reconstructs raster images (PNG) as vector SVGs. We do this by
shelling out to [vtracer](https://github.com/visioncortex/vtracer), a Rust
raster-to-vector tool, rather than potrace or a paid API.

**Why vtracer over potrace:** potrace is black-and-white only and cannot
reproduce a color logo; vtracer does full-color clustering + curve fitting,
which is the capability we actually wanted (the open-source analog to
Vectorizer.AI). potrace fit our existing brew-dependency model more cleanly, but
that convenience doesn't justify shipping a strictly lesser feature.

**Why our own tap formula:** vtracer is not in homebrew-core and no community
formula exists, so `brew install vtracer` is impossible out of the box. vtracer
publishes prebuilt darwin (arm64/x86) binaries, so we add a hand-maintained
`vtracer.rb` to our existing `coreycoburn/homebrew-tap` that downloads the
binary — no Rust toolchain, no compile. Cost: we bump version + sha256 when
vtracer releases (rare).

**Why lazy auto-install instead of the existing check-and-hint convention:**
kit's other deps (ghostscript, pngquant, oxipng) are only *checked* via
`requireBinary`, which prints an install hint and exits. For `trace` we instead
offer to install vtracer on first use — but only when output is not `--json`,
stdin is a real terminal, *and* `brew` is present, with a yes/no confirm;
otherwise (CI, JSON output, piped stdin, Linux without brew) we fall back to the
hint-and-exit behavior. The stdin-terminal check matters because forge's
"interactive" flag only reflects the absence of `--json`, not an attached tty. This keeps the
human-at-a-terminal path frictionless without silently mutating anyone's system
or breaking non-macOS builds.

## Consequences

- vtracer stays an implementation detail: `kit trace` exposes plain-language
  flags (`--mode`, `--detail`), not vtracer's raw vocabulary, so the backend
  could be swapped without a breaking CLI change.
- A new `ensureBinary` helper (confirm-then-install) lives alongside the
  unchanged `requireBinary` (check-and-hint); existing commands keep the latter.
- Output is flat-color by default. vtracer reconstructs gradients as many
  stacked flat-color bands (a real logo ballooned to ~1700 paths); since kit's
  primary inputs are logos and icons, we raise `gradient_step` to collapse the
  banding (~160 paths) and gate the photographic banded look behind
  `--gradients`. Trades gradient fidelity for dramatically smaller, cleaner SVGs.
