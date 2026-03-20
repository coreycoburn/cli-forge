package main

import (
	"github.com/coreycoburn/cli-forge/cmd/kit/commands"
	"github.com/coreycoburn/cli-forge/pkg/forge"
)

var version = "dev"

func main() {
	app := forge.New("kit", "Design asset toolkit — convert and optimize files",
		forge.WithVersion(version),
	)

	app.AddCommand(
		commands.ConvertCmd(),
		commands.OptimizeCmd(),
	)

	app.Execute()
}
