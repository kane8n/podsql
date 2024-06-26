package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "podsql",
		Usage: "usage",
		// Subcommands
		Commands: []*cli.Command{
			MysqlCommands(),
			SQLServerCommands(),
			PostgresCommands(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
