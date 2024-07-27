package main

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/go-yaml/yaml"
	"github.com/urfave/cli/v2"
)

type Config struct {
	Timezone  string `yaml:"timezone"`
	Namespace string `yaml:"namespace"`
}

func DefaultConfigPath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, ".config", "podsql.yaml"), nil
}

func NewConfig(c *cli.Context) (*Config, error) {
	confPath, err := DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	if c.IsSet("config") {
		confPath = c.String("config")
	}

	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		return &Config{
			Timezone:  c.String("timezone"),
			Namespace: c.String("namespace"),
		}, nil
	}

	conf, err := readConfig(confPath)
	if err != nil {
		return nil, err
	}

	if c.IsSet("timezone") {
		conf.Timezone = c.String("timezone")
	}
	if c.IsSet("namespace") {
		conf.Namespace = c.String("namespace")
	}

	return conf, nil
}

func readConfig(p string) (*Config, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var conf Config
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return nil, err
	}
	return &conf, nil
}
