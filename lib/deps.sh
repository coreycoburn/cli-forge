# framework/lib/deps.sh
# Dependency checking and installation prompts.
# Reads tool metadata from the project's lib/tools.sh (tool_binary, tool_install_cmd).
#
# Requires the entry point to export:
#   CLI_NO_INTERACTIVE  — 0 (interactive) or 1 (non-interactive/CI)
#   CLI_NAME            — binary name, used in error hint messages (e.g. "kit")

ensure_dep() {
  local tool="$1"
  local binary
  binary="$(tool_binary "$tool")"

  if command -v "$binary" &>/dev/null; then
    return 0
  fi

  gum log --level warn "'$tool' is not installed"

  if [[ ! -t 1 ]]; then
    if [[ "${CLI_NO_INTERACTIVE:-0}" == "1" ]]; then
      gum log --level info "Auto-installing $tool (--no-interactive)"
    else
      gum log --level error "$tool is required. Re-run with --no-interactive to auto-install."
      echo ""
      echo "  ${CLI_NAME:-cli} --no-interactive <command> [args]"
      exit 1
    fi
  else
    if [[ "${CLI_NO_INTERACTIVE:-0}" != "1" ]]; then
      gum confirm "Install $tool now?" || {
        gum log --level error "Aborted — $tool is required"
        exit 1
      }
    fi
  fi

  local install_cmd
  install_cmd="$(tool_install_cmd "$tool")"
  gum spin --title "Installing $tool..." -- bash -c "$install_cmd"
  gum log --level info "$tool installed"
}
