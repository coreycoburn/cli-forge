package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/coreycoburn/cli-forge/pkg/forge"
	"github.com/spf13/cobra"
)

// HelloCmd demonstrates the core forge patterns: themed output, spinner, and JSON mode.
func HelloCmd() *cobra.Command {
	var shout bool

	cmd := &cobra.Command{
		Use:   "hello [name]",
		Short: "Say hello with style",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := forge.OutputFrom(cmd)

			name := "world"
			if len(args) > 0 {
				name = args[0]
			}

			greeting := fmt.Sprintf("Hello, %s!", name)
			if shout {
				greeting = strings.ToUpper(greeting)
			}

			err := out.Spin("Preparing greeting...", func() error {
				time.Sleep(800 * time.Millisecond)
				return nil
			})
			if err != nil {
				return err
			}

			if out.IsInteractive() {
				out.Header(greeting)
				out.Success("Greeting delivered")
			} else {
				return out.JSON(map[string]string{
					"greeting": greeting,
					"name":     name,
				})
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&shout, "shout", "s", false, "SHOUT THE GREETING")

	return cmd
}
