# framework/lib/utils.sh
# Shared utility functions available to all CLI commands.

# Print a human-readable byte size (e.g. 1.2 KB or 456 B)
_human_bytes() {
  local bytes="$1"
  if [[ "$bytes" -ge 1024 ]]; then
    awk -v b="$bytes" 'BEGIN {printf "%.1f KB", b / 1024}'
  else
    printf "%d B" "$bytes"
  fi
}

# Print percent change between two byte counts (e.g. -12.3 or +4.5)
_pct_change() {
  local before="$1" after="$2"
  awk -v b="$before" -v a="$after" 'BEGIN {
    if (b == 0) printf "0.0"
    else printf "%.1f", ((a - b) / b) * 100
  }'
}
