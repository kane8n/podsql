package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/urfave/cli/v2"
)

type PostgresCommander struct {
	originalArgs []string

	escapedArgs []string
	connectInfo ConnectInfo
	query       []string
	help        bool
	helpCommand string
}

func NewPostgresCommander(args []string) (*PostgresCommander, error) {
	c := &PostgresCommander{}
	c.originalArgs = args
	c.escapedArgs = make([]string, 0)
	c.connectInfo = ConnectInfo{Port: "5432"}
	if err := c.parseArgs(args); err != nil {
		return nil, err
	}
	return c, nil
}

func (m *PostgresCommander) parseArgs(args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-?" || arg == "--help":
			m.help = true
			break
		case strings.HasPrefix(arg, "--help="):
			m.help = true
			m.helpCommand = strings.Split(arg, "=")[1]
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
		case arg == "-U" || arg == "--username":
			i++
			m.connectInfo.User = args[i]
		case strings.HasPrefix(arg, "--username="):
			parts := strings.Split(arg, "=")
			m.connectInfo.User = parts[1]
		case arg == "--password":
			i++
			m.connectInfo.Password = args[i]
		case strings.HasPrefix(arg, "--password="):
			parts := strings.Split(arg, "=")
			m.connectInfo.Password = parts[1]
		case arg == "-d" || arg == "--dbname":
			i++
			m.connectInfo.DbName = args[i]
		case strings.HasPrefix(arg, "--dbname="):
			parts := strings.Split(arg, "=")
			m.connectInfo.DbName = parts[1]
		case strings.HasPrefix(arg, "--command="):
			parts := strings.Split(arg, "=")
			m.query = append(m.query, parts[1])
		case arg == "-c" || arg == "--command":
			i++
			m.query = append(m.query, args[i])
		case arg == "-f" || arg == "--file":
			i++
			lines, err := m.readFile(args[i], "")
			if err != nil {
				return err
			}
			m.query = append(m.query, lines...)
		case strings.HasPrefix(arg, "--file="):
			parts := strings.Split(arg, "=")
			lines, err := m.readFile(parts[1], "")
			if err != nil {
				return err
			}
			m.query = append(m.query, lines...)
		case strings.HasPrefix(arg, "--") && strings.Contains(arg, "="):
			parts := strings.Split(arg, "=")
			m.escapedArgs = append(m.escapedArgs, fmt.Sprintf("%s=\"%s\"", parts[0], parts[1]))
		case strings.HasPrefix(arg, "-"):
			m.escapedArgs = append(m.escapedArgs, arg)
		default:
			m.escapedArgs = append(m.escapedArgs, fmt.Sprintf("\"%s\"", arg))
		}
	}
	m.fetchConnectInfoEnv()
	if err := m.fetchPGPass(); err != nil {
		return err
	}
	return nil
}

func (m *PostgresCommander) fetchConnectInfoEnv() {
	server, exist := os.LookupEnv("PGHOST")
	if exist && m.connectInfo.Server == "" {
		m.connectInfo.Server = server
	}
	port, exist := os.LookupEnv("PGPORT")
	if exist && m.connectInfo.Port == "5432" {
		m.connectInfo.Port = port
	}
	dbname, exist := os.LookupEnv("PGDATABASE")
	if exist && m.connectInfo.DbName == "" {
		m.connectInfo.DbName = dbname
	}
	username, exist := os.LookupEnv("PGUSER")
	if exist && m.connectInfo.User == "" {
		m.connectInfo.User = username
	}
	password, exist := os.LookupEnv("PGPASSWORD")
	if exist && m.connectInfo.Password == "" {
		m.connectInfo.Password = password
	}
	return
}

func (m *PostgresCommander) fetchPGPass() error {
	var pgPassPath string
	pgPassPath, exist := os.LookupEnv("PGPASSFILE")
	if !exist {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		pgPassPath = filepath.Join(homeDir, ".pgpass")
	}
	if _, err := os.Stat(pgPassPath); os.IsNotExist(err) {
		return nil
	}

	lines, err := m.readFile(pgPassPath, "#")
	if err != nil {
		return err
	}

	for _, line := range lines {
		// はじめに現在の接続パラメータと一致した行を探す
		fields := strings.Split(line, ":")
		if len(fields) != 5 {
			return fmt.Errorf("Error: Invalid format in %s", pgPassPath)
		}
		if m.connectInfo.Server != "" && fields[0] != m.connectInfo.Server && fields[0] != "*" {
			continue
		}
		if fields[1] != m.connectInfo.Port && fields[1] != "*" {
			continue
		}
		if m.connectInfo.DbName != "" && fields[2] != m.connectInfo.DbName && fields[2] != "*" {
			continue
		}
		if m.connectInfo.User != "" && fields[3] != m.connectInfo.User && fields[3] != "*" {
			continue
		}

		if fields[0] != "*" && m.connectInfo.Server == "" {
			m.connectInfo.Server = fields[0]
		}
		if fields[1] != "*" && m.connectInfo.Port == "5432" {
			m.connectInfo.Port = fields[1]
		}
		if fields[2] != "*" && m.connectInfo.DbName == "" {
			m.connectInfo.DbName = fields[2]
		}
		if fields[3] != "*" && m.connectInfo.User == "" {
			m.connectInfo.User = fields[3]
		}
		if m.connectInfo.Password == "" {
			m.connectInfo.Password = fields[4]
		}
		return nil
	}

	return nil
}

func (m *PostgresCommander) readFile(file, commentPrefix string) ([]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lines := []string{}
	for scanner.Scan() {
		l := scanner.Text()
		if commentPrefix != "" && strings.HasPrefix(l, commentPrefix) {
			continue
		}
		lines = append(lines, l)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func (m *PostgresCommander) IsInteractive() bool {
	return len(m.query) == 0 && !m.help
}

func (m *PostgresCommander) ConnectInfo() ConnectInfo {
	return m.connectInfo
}

func (m *PostgresCommander) Query() string {
	return strings.Join(m.query, "\n")
}

func (m *PostgresCommander) HelpCommand() string {
	if m.helpCommand != "" {
		return fmt.Sprintf("psql --help=%s", m.helpCommand)
	}
	return "psql --help"
}

func (m *PostgresCommander) Command() string {
	if m.help {
		return m.HelpCommand()
	}

	connectionArgs := []string{
		"psql",
		"-h", m.connectInfo.Server,
		"-p", m.connectInfo.Port,
		"-U", "$PGUSER",
		"-w",
		"-f", "/sql/query.sql",
	}
	if m.connectInfo.DbName != "" {
		connectionArgs = append(connectionArgs, "-d", m.connectInfo.DbName)
	}
	return fmt.Sprintf(strings.Join(slices.Concat(connectionArgs, m.escapedArgs), " "))
}

func (m *PostgresCommander) InteractiveCommand() string {
	connectionArgs := []string{
		"psql",
		"-h", m.connectInfo.Server,
		"-p", m.connectInfo.Port,
		"-U", "$PGUSER",
		"-w",
	}
	if m.connectInfo.DbName != "" {
		connectionArgs = append(connectionArgs, "-d", m.connectInfo.DbName)
	}
	return fmt.Sprintf(strings.Join(slices.Concat(connectionArgs, m.escapedArgs), " "))
}

func (m *PostgresCommander) ContainerImage() string {
	return "postgres:16"
}

func (m *PostgresCommander) SecretEnvKV() map[string]string {
	return map[string]string{
		"PGUSER":     "username",
		"PGPASSWORD": "password",
	}
}

func (m *PostgresCommander) CommandType() CommandType {
	return SQLCmd
}

func (m *PostgresCommander) ParseResults(result string) []string {
	return []string{}
}

func PostgresCommands() *cli.Command {
	return &cli.Command{
		Name:            "psql",
		Usage:           "execute psql commands",
		ArgsUsage:       "<psql options>",
		SkipFlagParsing: true,
		Action:          executePostgresAction,
	}
}

func executePostgresAction(c *cli.Context) error {
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
	dbCommander, err := NewPostgresCommander(args)
	if err != nil {
		return err
	}

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
