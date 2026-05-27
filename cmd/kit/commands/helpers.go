package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/coreycoburn/cli-forge/pkg/forge"
	"golang.org/x/term"
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

// ensureBinary guarantees an external binary is available, offering to install
// it via Homebrew on first use. When the session is interactive and `brew` is
// present, it asks the user to confirm installing `formula`; on yes it runs
// `brew install <formula>`. In any non-interactive context (JSON output, CI),
// when brew is absent, or when the user declines, it falls back to a
// hint-and-error — the same safe, predictable behaviour as requireBinary.
func ensureBinary(out *forge.Output, name, formula string) error {
	if _, err := exec.LookPath(name); err == nil {
		return nil
	}

	hint := fmt.Errorf("%s is required but not installed\n  Install with: brew install %s", name, formula)

	// Only auto-install for a human at a terminal with brew available;
	// everything else (JSON mode, piped/CI stdin, no brew) gets the
	// predictable hint. Note: forge's "interactive" only means --json was not
	// passed, so we additionally require stdin to be a real terminal before
	// prompting — otherwise a piped or CI invocation would block on input.
	if !out.IsInteractive() || !stdinIsTerminal() {
		return hint
	}
	if _, err := exec.LookPath("brew"); err != nil {
		return hint
	}

	out.Warn(fmt.Sprintf("%s is required but not installed.", name))
	fmt.Fprintf(os.Stderr, "  Install it now with 'brew install %s'? [y/N] ", formula)
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
		return hint
	}

	err := out.Spin(fmt.Sprintf("Installing %s", formula), func() error {
		return exec.Command("brew", "install", formula).Run()
	})
	if err != nil {
		return fmt.Errorf("failed to install %s via brew: %w", formula, err)
	}
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s still not found after install", name)
	}
	return nil
}

// stdinIsTerminal reports whether stdin is an interactive terminal (not a
// pipe, redirect, /dev/null, or closed handle), so we never prompt where no
// human can answer.
func stdinIsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
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
