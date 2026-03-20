package commands

import (
	"fmt"
	"os/exec"
)

// requireBinary checks that an external binary exists in PATH.
// Returns nil if found, or an error with install instructions.
func requireBinary(name, installHint string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("%s is required but not installed\n  Install with: %s", name, installHint)
	}
	return nil
}

// humanBytes formats a byte count as a human-readable string.
func humanBytes(b int) string {
	if b >= 1024 {
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	}
	return fmt.Sprintf("%d B", b)
}

// pctChange calculates the percent change between two values.
func pctChange(before, after int) string {
	if before == 0 {
		return "0.0"
	}
	pct := (float64(after) - float64(before)) / float64(before) * 100
	if pct >= 0 {
		return fmt.Sprintf("+%.1f", pct)
	}
	return fmt.Sprintf("%.1f", pct)
}
