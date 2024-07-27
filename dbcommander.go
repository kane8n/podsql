package main

import (
	"strings"
)

type CommandType string

func (c CommandType) String() string {
	return string(c)
}

const (
	MySQL      CommandType = "mysql"
	SQLCmd     CommandType = "sqlcmd"
	PostgreSQL CommandType = "postgresql"
	Unknown    CommandType = "unknown"
)

type DBCommander interface {
	ConnectInfo() ConnectInfo
	Query() string
	HelpCommand() string
	Command() string
	InteractiveCommand() string
	ContainerImage() string
	SecretEnvKV() map[string]string
	CommandType() CommandType
	ParseResults(result string) []string
	IsInteractive() bool

	parseArgs(args []string) error
}

type ConnectInfo struct {
	Server   string
	Port     string
	User     string
	Password string
	DbName   string
}

func DetermineDatabaseTypeFronConnectionString(connectionString string) CommandType {
	if connectionString == "" {
		return Unknown
	}

	switch {
	case strings.Contains(connectionString, "mysql"):
		return MySQL
	case strings.Contains(connectionString, "sqlserver"):
		return SQLCmd
	case strings.Contains(connectionString, "postgresql"):
		return PostgreSQL
	default:
		return Unknown
	}
}
