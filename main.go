package main

import (
	"fmt"
	"os"

	"github.com/titpetric/cli"

	"github.com/titpetric/vuego-cli/commands/diff"
	"github.com/titpetric/vuego-cli/commands/docs"
	"github.com/titpetric/vuego-cli/commands/format"
	"github.com/titpetric/vuego-cli/commands/render"
	"github.com/titpetric/vuego-cli/commands/serve"
	"github.com/titpetric/vuego-cli/commands/tour"
	"github.com/titpetric/vuego-cli/commands/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	app := cli.NewApp("vuego-cli")

	// Register commands
	app.AddCommand("fmt", format.Name, format.New)
	app.AddCommand("render", render.Name, render.New)
	app.AddCommand("diff", diff.Name, diff.New)
	app.AddCommand("serve", serve.Name, serve.New)
	app.AddCommand("tour", tour.Name, tour.New)
	app.AddCommand("docs", docs.Name, docs.New)

	// Version command requires build info
	app.AddCommand("version", version.Name, func() *cli.Command {
		return version.NewWithInfo(version.Info{
			Version:    Version,
			Commit:     Commit,
			CommitTime: CommitTime,
			Branch:     Branch,
			Modified:   Modified,
		})
	})

	return app.Run()
}
