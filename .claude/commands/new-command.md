Create a new CLI command following all established patterns in this repository.

The user will tell you the name and purpose of the new command, plus any tools it depends on and any flags it accepts. Ask for clarification if needed before proceeding.

---

## Step 0: Discover CLI identity

Before writing any code, read these files to ground yourself in the CLI's identity and current patterns:

- **Entry point** — find the file in the repo root that sources `lib/tools.sh`. Read it to extract:
  - `{cli_name}` — the binary name (e.g. `kit`, `mycli`, `confluence`)
  - `{CLI_UPPER}` — the NO_INTERACTIVE env var prefix (e.g. `KIT` from `KIT_NO_INTERACTIVE=0`)
- **`config/theme.sh`** — read to extract:
  - `{COLOR_PREFIX}` — the color variable prefix (e.g. `KIT_COLOR`, `MYCLI_COLOR`)
  - `{header_fn}` — the header function name (e.g. `kit_header`, `mycli_header`)

Then read the rest of the architecture files:

- `lib/tools.sh` — single source of truth for all external tool metadata
- `lib/deps.sh` — `ensure_dep()` for checking/installing tools
- `lib/conversions.sh` — `conversion_deps()` and `conversions_list()` (if command does file conversion)
- `lib/optimizers.sh` — `optimizer_tool()` and `optimizers_list()` (if command does in-place optimization)
- `commands/` — any existing commands for reference
- `completions/_`{cli_name} — zsh tab completion definitions

Use `{cli_name}`, `{CLI_UPPER}`, `{COLOR_PREFIX}`, `{header_fn}`, and `{cmd_name}` (the new command name) as explicit tokens in all code you write. Fill them from the discovery step.

---

## Files to create or modify

### 1. `lib/tools.sh` — register any new external tools

Add entries to all four functions for each new tool. This is the **only** place tool metadata lives — everything else derives from it.

```bash
tool_binary() {
  case "$1" in
    newtool) echo "newtool-bin" ;;   # binary name as it appears in PATH
    ...
  esac
}

tool_install_cmd() {
  case "$1" in
    newtool) echo "brew install newtool" ;;
    ...
  esac
}

tool_uninstall_cmd() {
  case "$1" in
    newtool) echo "brew uninstall newtool" ;;
    ...
  esac
}

tools_all() {
  echo "newtool"   # space-separated; uninstall.sh loops this
}
```

If the command needs no new external tools, skip this step.

### 2. `lib/conversions.sh` or `lib/optimizers.sh` — register the new operation

**For conversion commands** (take an input file, produce an output file), add to `conversions.sh`:

```bash
conversion_deps() {
  case "$1" in
    eps_svg)    echo "ghostscript inkscape" ;;
    from_to)    echo "newtool"             ;;  # key = from_ext_to_ext
    *)          return 1 ;;
  esac
}

conversions_list() {
  printf "  %-8s → %-8s  requires: %s\n" "eps" "svg" "ghostscript, inkscape"
  printf "  %-8s → %-8s  requires: %s\n" "from" "to"  "newtool"
}
```

**For optimizer commands** (modify a file in-place), add to `optimizers.sh`:

```bash
optimizer_tool() {
  case "$1" in
    svg)  echo "svgo"    ;;
    ext)  echo "newtool" ;;
    *)    return 1 ;;
  esac
}

optimizers_list() {
  printf "  %-8s  requires: %s\n" "svg" "svgo"
  printf "  %-8s  requires: %s\n" "ext" "newtool"
}
```

If the new command is neither a conversion nor an optimizer (e.g. a fetch, a report, a validation), skip this step and handle tool deps directly in the command file.

### 3. `commands/{cmd_name}.sh` — the command implementation

Use this structure exactly:

```bash
# commands/{cmd_name}.sh
# Sourced by {cli_name} main script

_{cmd_name}_help() {
  gum format -- "# {cli_name} {cmd_name}

> One-line description of what this command does.

## Usage

\`{cli_name} [flags] {cmd_name} [--flag] <arg>\`

## Global Flags

| Flag                    | Description                                          |
|-------------------------|------------------------------------------------------|
| \`-n, --no-interactive\` | Skip prompts, auto-install deps, output JSON         |
| \`-h, --help\`           | Show this help                                       |

## Command Flags

| Flag       | Description              |
|------------|--------------------------|
| \`--flag\` | What this flag does      |

## Examples

\`\`\`
{cli_name} {cmd_name} input.ext output.ext
{cli_name} {cmd_name} --flag input.ext
\`\`\`

## Non-interactive Usage

\`\`\`
{cli_name} --no-interactive {cmd_name} input.ext output.ext
\`\`\`"
}

_{cmd_name}_output_json() {
  # Adapt fields to what this command produces
  local status="$1" file="$2" message="${3:-}"
  printf '{"status":"%s","file":"%s","message":"%s"}\n' \
    "$status" "$file" "$message"
}

cmd_{cmd_name}() {
  # 1. Help flag
  if [[ "${1:-}" =~ ^(-h|--help)$ ]]; then
    _{cmd_name}_help
    return 0
  fi

  # 2. Parse command-specific flags
  local myflag=0
  while [[ "${1:-}" =~ ^- ]]; do
    case "${1:-}" in
      --flag) myflag=1; shift ;;
      *)      break ;;
    esac
  done

  # 3. Validate argument count
  if [[ $# -ne 1 ]]; then
    gum log --level error "Expected 1 argument, got $#"
    echo "Usage: {cli_name} {cmd_name} <file>"
    exit 1
  fi

  local file="$1"

  # 4. Validate file exists
  if [[ ! -f "$file" ]]; then
    [[ "${CLI_UPPER}_NO_INTERACTIVE" == "1" ]] \
      && _{cmd_name}_output_json "error" "$file" "file not found" \
      || gum log --level error "File not found: $file"
    exit 1
  fi

  # 5. Validate format / look up operation (if format-dependent)
  local ext
  ext="$(echo "${file##*.}" | tr '[:upper:]' '[:lower:]')"
  local tool
  if ! tool="$(optimizer_tool "$ext")"; then
    [[ "${CLI_UPPER}_NO_INTERACTIVE" == "1" ]] \
      && _{cmd_name}_output_json "error" "$file" "unsupported format: $ext" \
      || { gum log --level error "Unsupported format: $ext"; echo ""; optimizers_list; }
    exit 1
  fi

  # 6. Ensure deps
  ensure_dep "$tool"

  # 7. Run operation — spinner in interactive TTY, plain otherwise
  if [[ -t 1 ]] && [[ "${CLI_UPPER}_NO_INTERACTIVE" != "1" ]]; then
    gum spin --title "Processing ${file##*/}" \
      -- bash -c '...' "$file"
  else
    # same operation, no spinner
    ...
  fi

  # 8. Output result
  if [[ "${CLI_UPPER}_NO_INTERACTIVE" == "1" ]]; then
    _{cmd_name}_output_json "success" "$file" "done"
  else
    gum log --level info "Done → $file"
  fi
}
```

**Key patterns to preserve:**

- Redirect noisy tool stderr: `tool ... 2>/dev/null`
- Temp files: `tmp="$(mktemp /tmp/{cli_name}_XXXXXX.ext)"` + `trap 'rm -f "$tmp"' EXIT` + `trap - EXIT` after cleanup
- The `gum spin` block and the plain block must perform **identical operations**
- JSON output only in `--no-interactive` mode; all human output via `gum log` / `gum format`
- Never call `exit` from inside a `gum spin` subshell — check the output file after spin completes

### 4. Entry point (`{cli_name}`) — dispatch the new command

Add a case entry and source the command file:

```bash
case "$CMD" in
  {cmd_name})
    source "$SCRIPT_DIR/commands/{cmd_name}.sh"
    cmd_{cmd_name} "$@"
    ;;
  ...
esac
```

Also update `_{cli_name}_help()` in the entry point to add the command to the commands table:

```
| {cmd_name}  | One-line description           |
```

### 5. `completions/_{cli_name}` — zsh tab completion

Add the command to the commands array, add a case entry, and write a completion function:

```zsh
local -a commands=(
  'existing:Existing command'
  '{cmd_name}:One-line description'          # ← add
  'help:Show help'
)

case $words[2] in
  existing)   _{cli_name}_existing ;;
  {cmd_name}) _{cli_name}_{cmd_name} ;;    # ← add
  ...
esac

_{cli_name}_{cmd_name}() {
  _arguments \
    '(-h --help)'{-h,--help}'[Show help]' \
    '--flag[What this flag does]' \
    '1:file:_files'
}
```

---

## Dependency lifecycle — no manual install.sh changes needed

`install.sh` and `uninstall.sh` do **not** need changes for new command-level tools:

- **Installing** tools happens automatically at runtime via `ensure_dep` in `lib/deps.sh`
- **Uninstalling** tools happens via the `tools_all()` loop in `uninstall.sh` — adding the tool to `tools_all()` in `lib/tools.sh` is sufficient

---

## TUI conventions

| Use case                     | Pattern                                                                 |
|------------------------------|-------------------------------------------------------------------------|
| Success message              | `gum log --level info "..."`                                            |
| Warning                      | `gum log --level warn "..."`                                            |
| Error message                | `gum log --level error "..."`                                           |
| Long-running operation       | `gum spin --title "..." -- command`                                     |
| Markdown help                | `gum format -- "# heading\n..."`                                        |
| Styled text (primary color)  | `gum style --foreground "${COLOR_PREFIX}_PRIMARY" "..."`                |
| Styled text (muted)          | `gum style --foreground "${COLOR_PREFIX}_SECONDARY" "..."`              |
| Styled text (accent)         | `gum style --foreground "${COLOR_PREFIX}_ACCENT" "..."`                 |
| Banner header                | `{header_fn} "title"` (from config/theme.sh)                            |
| Interactive yes/no           | `gum confirm "Do this?" \|\| { gum log --level error "Aborted"; exit 1; }` |

Spinner type and color are set globally by `config/theme.sh` via `GUM_SPIN_SPINNER` and `GUM_SPIN_SPINNER_FOREGROUND` — do not pass `--spinner` inline.

---

## Checklist before finishing

- [ ] New tools added to all four functions in `lib/tools.sh` (or confirmed none needed)
- [ ] Conversion/optimizer lookup table updated in `lib/conversions.sh` or `lib/optimizers.sh`
- [ ] `commands/{cmd_name}.sh` created with help, JSON output, validation, spinner, and plain execution paths
- [ ] Entry point dispatch case added and help table updated
- [ ] `completions/_{cli_name}` updated with command, case, and completion function
- [ ] Spinner and plain paths perform identical operations
- [ ] `--no-interactive` produces JSON; interactive produces gum output
- [ ] Temp files use `mktemp /tmp/{cli_name}_XXXXXX.ext` with `trap` cleanup
- [ ] Tool stderr suppressed with `2>/dev/null` where appropriate
