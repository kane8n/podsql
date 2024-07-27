package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	defaultConfigPath, err := DefaultConfigPath()
	if err != nil {
		log.Fatal(err)
	}

	app := &cli.App{
		Name:  "podsql",
		Usage: "usage",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "timezone",
				Aliases: []string{"t", "tz"},
				Usage:   "timezone",
				Value:   "UTC",
			},
			&cli.StringFlag{
				Name:    "namespace",
				Aliases: []string{"n", "ns"},
				Usage:   "namespace",
				Value:   "default",
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "config file path",
				Value:   defaultConfigPath,
			},
		},
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
