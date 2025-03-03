package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/urfave/cli/v2"
)

type SqlServerCommander struct {
	originalArgs []string

	escapedArgs []string
	connectInfo ConnectInfo
	query       string
	help        bool
}

func NewSqlServerCommander(args []string) *SqlServerCommander {
	c := &SqlServerCommander{}
	c.originalArgs = args
	c.escapedArgs = make([]string, 0)
	c.connectInfo = ConnectInfo{}
	c.parseArgs(args)
	return c
}

func (m *SqlServerCommander) parseArgs(args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-?" || arg == "--help":
			m.help = true
			break
		case strings.HasPrefix(arg, "-S"):
			if arg == "-S" {
				i++
				arg = args[i]
			} else {
				arg = strings.TrimPrefix(arg, "-S")
			}
			serverPort := strings.Split(arg, ",")
			if strings.HasPrefix(serverPort[0], "tcp:") {
				serverPort[0] = serverPort[0][4:]
			}
			m.connectInfo.Server = serverPort[0]
			if len(serverPort) > 1 {
				m.connectInfo.Port = serverPort[1]
			}
		case strings.HasPrefix(arg, "-U"):
			if arg == "-U" {
				i++
				arg = args[i]
			} else {
				arg = strings.TrimPrefix(arg, "-U")
			}
			m.connectInfo.User = arg
		case strings.HasPrefix(arg, "-P"):
			if arg == "-P" {
				i++
				arg = args[i]
			} else {
				arg = strings.TrimPrefix(arg, "-P")
			}
			m.connectInfo.Password = arg
		case strings.HasPrefix(arg, "-d"):
			if arg == "-d" {
				i++
				arg = args[i]
			} else {
				arg = strings.TrimPrefix(arg, "-d")
			}
			m.connectInfo.DbName = arg
		case strings.HasPrefix(arg, "-q"):
			if arg == "-q" {
				i++
				arg = args[i]
			} else {
				arg = strings.TrimPrefix(arg, "-q")
			}
			m.query = arg
		case strings.HasPrefix(arg, "-Q"):
			if arg == "-Q" {
				i++
				arg = args[i]
			} else {
				arg = strings.TrimPrefix(arg, "-Q")
			}
			m.query = arg
		case strings.HasPrefix(arg, "-D"):
			// Do Nothing
		case strings.HasPrefix(arg, "-i"):
			// Do Nothing
		case strings.HasPrefix(arg, "-"):
			if len(arg) == 2 {
				m.escapedArgs = append(m.escapedArgs, arg)
			} else {
				m.escapedArgs = append(m.escapedArgs, fmt.Sprintf("%s\"%s\"", arg[:2], arg[2:]))
			}
		default:
			m.escapedArgs = append(m.escapedArgs, fmt.Sprintf("\"%s\"", arg))
		}
	}
	return nil
}

func (m *SqlServerCommander) IsInteractive() bool {
	return m.query == "" && !m.help
}

func (m *SqlServerCommander) ConnectInfo() ConnectInfo {
	return m.connectInfo
}

func (m *SqlServerCommander) Query() string {
	return m.query
}

func (m *SqlServerCommander) HelpCommand() string {
	return "/opt/mssql-tools/bin/sqlcmd -?"
}

func (m *SqlServerCommander) Command() string {
	if m.help {
		return m.HelpCommand()
	}

	serverPort := m.connectInfo.Server
	if m.connectInfo.Port != "" {
		serverPort = fmt.Sprintf("%s,%s", m.connectInfo.Server, m.connectInfo.Port)
	}
	connectionArgs := []string{
		"/opt/mssql-tools/bin/sqlcmd",
		"-S", serverPort,
		"-U", "$SECRET_DB_USER",
		"-P", "$SECRET_DB_PASSWORD",
		"-i", "/sql/query.sql",
	}
	if m.connectInfo.DbName != "" {
		connectionArgs = append(connectionArgs, "-d", m.connectInfo.DbName)
	}
	return fmt.Sprintf(strings.Join(slices.Concat(connectionArgs, m.escapedArgs), " "))
}

func (m *SqlServerCommander) InteractiveCommand() string {
	serverPort := m.connectInfo.Server
	if m.connectInfo.Port != "" {
		serverPort = fmt.Sprintf("%s,%s", m.connectInfo.Server, m.connectInfo.Port)
	}
	connectionArgs := []string{
		"/opt/mssql-tools/bin/sqlcmd",
		"-S", serverPort,
		"-U", "$SECRET_DB_USER",
		"-P", "$SECRET_DB_PASSWORD",
	}
	if m.connectInfo.DbName != "" {
		connectionArgs = append(connectionArgs, "-d", m.connectInfo.DbName)
	}
	return fmt.Sprintf(strings.Join(slices.Concat(connectionArgs, m.escapedArgs), " "))
}

func (m *SqlServerCommander) ContainerImage() string {
	return "mcr.microsoft.com/mssql-tools"
}

func (m *SqlServerCommander) SecretEnvKV() map[string]string {
	return map[string]string{
		"SECRET_DB_USER":     "username",
		"SECRET_DB_PASSWORD": "password",
	}
}

func (m *SqlServerCommander) CommandType() CommandType {
	return SQLCmd
}

func (m *SqlServerCommander) ParseResults(result string) []string {
	return []string{}
}

func SQLServerCommands() *cli.Command {
	return &cli.Command{
		Name:            "sqlcmd",
		Usage:           "execute sqlcmd commands",
		ArgsUsage:       "<sqlcmd options>",
		SkipFlagParsing: true,
		Action:          executeSqlServerAction,
	}
}

func executeSqlServerAction(c *cli.Context) error {
	config, err := NewConfig(c)
	if err != nil {
		return err
	}
	podName, err := CreatePodName("podsql")
	if err != nil {
		return err
	}
	args := c.Args().Slice()
	if len(args) == 0 {
		args = []string{"-?"}
	}
	dbCommander := NewSqlServerCommander(args)

	if dbCommander.IsInteractive() {
		return ExecPod(config, podName, dbCommander)
	}

	out, err := RunPod(config, podName, dbCommander)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}
