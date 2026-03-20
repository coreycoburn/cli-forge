package main

import (
	"github.com/coreycoburn/cli-forge/cmd/example/commands"
	"github.com/coreycoburn/cli-forge/pkg/forge"
)

var version = "dev"

func main() {
	app := forge.New("example", "An example CLI built with cli-forge",
		forge.WithVersion(version),
	)

	app.AddCommand(commands.HelloCmd())

	app.Execute()
}
