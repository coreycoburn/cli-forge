package main

import (
	"github.com/coreycoburn/cli-forge/cmd/confluence/commands"
	"github.com/coreycoburn/cli-forge/pkg/forge"
)

var version = "dev"

func main() {
	app := forge.New("confluence", "Confluence CLI — fetch and manage pages",
		forge.WithVersion(version),
	)

	app.AddCommand(
		commands.GetCmd(),
		commands.ConfigCmd(),
	)

	app.Execute()
}
