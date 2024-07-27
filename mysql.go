package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/urfave/cli/v2"
)

type MysqlCommander struct {
	originalArgs []string

	escapedArgs []string
	connectInfo ConnectInfo
	query       string
}

func NewMysqlCommander(args []string) *MysqlCommander {
	c := &MysqlCommander{}
	c.originalArgs = args
	c.escapedArgs = make([]string, 0)
	c.connectInfo = ConnectInfo{Port: "3306"}
	c.parseArgs(args)
	return c
}

func (m *MysqlCommander) parseArgs(args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--host":
			i++
			m.connectInfo.Server = args[i]
		case strings.HasPrefix(arg, "--host="):
			m.connectInfo.Server = strings.Split(arg, "=")[1]
		case arg == "-P" || arg == "--port":
			i++
			m.connectInfo.Port = args[i]
		case strings.HasPrefix(arg, "--port="):
			parts := strings.Split(arg, "=")
			m.connectInfo.Port = parts[1]
		case arg == "-u" || arg == "--user":
			i++
			m.connectInfo.User = args[i]
		case strings.HasPrefix(arg, "--user="):
			parts := strings.Split(arg, "=")
			m.connectInfo.User = parts[1]
		case arg == "-p" || arg == "--password":
			i++
			m.connectInfo.Password = args[i]
		case strings.HasPrefix(arg, "--password="):
			parts := strings.Split(arg, "=")
			m.connectInfo.Password = parts[1]
		case arg == "-D" || arg == "--database":
			i++
			m.connectInfo.DbName = args[i]
		case strings.HasPrefix(arg, "--database="):
			parts := strings.Split(arg, "=")
			m.connectInfo.DbName = parts[1]
		case strings.HasPrefix(arg, "--execute="):
			parts := strings.Split(arg, "=")
			m.query = parts[1]
		case arg == "-e" || arg == "--execute":
			i++
			m.query = args[i]
		case strings.HasPrefix(arg, "--") && strings.Contains(arg, "="):
			parts := strings.Split(arg, "=")
			m.escapedArgs = append(m.escapedArgs, fmt.Sprintf("%s=\"%s\"", parts[0], parts[1]))
		case strings.HasPrefix(arg, "-"):
			m.escapedArgs = append(m.escapedArgs, arg)
		default:
			m.escapedArgs = append(m.escapedArgs, fmt.Sprintf("\"%s\"", arg))
		}
	}
	return nil
}

func (m *MysqlCommander) IsInteractive() bool {
	return m.query == ""
}

func (m *MysqlCommander) ConnectInfo() ConnectInfo {
	return m.connectInfo
}

func (m *MysqlCommander) Query() string {
	return m.query
}

func (m *MysqlCommander) Command() string {
	connectionArgs := []string{
		"mysql",
		"-h", m.connectInfo.Server,
		"-P", m.connectInfo.Port,
		"-u", "$SECRET_DB_USER",
	}
	if m.connectInfo.DbName != "" {
		connectionArgs = append(connectionArgs, "-D", m.connectInfo.DbName)
	}
	return fmt.Sprintf("%s < %s", strings.Join(slices.Concat(connectionArgs, m.escapedArgs), " "), "/sql/query.sql")
}

func (m *MysqlCommander) InteractiveCommand() string {
	connectionArgs := []string{
		"mysql",
		"-h", m.connectInfo.Server,
		"-P", m.connectInfo.Port,
		"-u", "$SECRET_DB_USER",
	}
	if m.connectInfo.DbName != "" {
		connectionArgs = append(connectionArgs, "-D", m.connectInfo.DbName)
	}
	return strings.Join(slices.Concat(connectionArgs, m.escapedArgs), " ")
}

func (m *MysqlCommander) ContainerImage() string {
	return "mysql:8.0"
}

func (m *MysqlCommander) SecretEnvKV() map[string]string {
	return map[string]string{
		"SECRET_DB_USER": "username",
		"MYSQL_PWD":      "password",
	}

}

func (m *MysqlCommander) CommandType() CommandType {
	return MySQL
}

func (m *MysqlCommander) ParseResults(result string) []string {
	return []string{}
}

func MysqlCommands() *cli.Command {
	return &cli.Command{
		Name:            "mysql",
		Usage:           "execute mysql commands",
		ArgsUsage:       "<mysql options>",
		SkipFlagParsing: true,
		Action:          executeMysqlAction,
	}
}

func executeMysqlAction(c *cli.Context) error {
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
		args = []string{"--help"}
	}
	dbCommander := NewMysqlCommander(args)

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
