package main

import "github.com/urfave/cli/v2"

func PostgresCommands() *cli.Command {
	return &cli.Command{
		Name:      "postgres",
		Usage:     "execute postgres commands",
		ArgsUsage: "<postgres options>",
		Action:    executePostgresAction,
	}
}

func executePostgresAction(c *cli.Context) error {
	return nil
}
