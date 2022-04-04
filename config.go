package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Host     string   `yaml:"host"`
	User     string   `yaml:"user"`
	Shell    string   `yaml:"shell"`
	Location string   `yaml:"location"`
	Command  string   `yaml:"command"`
	Env      []string `yaml:"env"`
}

func readConf(fpath string, overrideCommand string) (*Config, error) {
	buf, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}

	err = yaml.Unmarshal(buf, cfg)
	if err != nil {
		return nil, err
	}

	cfg.Command = buildCommand(cfg, overrideCommand)

	fmt.Println(cfg.Command)

	return cfg, nil
}

func buildCommand(cfg *Config, overrideCommand string) string {
	var cmd []string

	if len(cfg.Env) > 0 {
		for _, val := range cfg.Env {
			cmd = append(cmd, "export "+val)
		}
	}

	if cfg.Location != "" {
		cmd = append(cmd, "cd "+cfg.Location)
	}

	finalCommand := overrideCommand

	if finalCommand == "" {
		finalCommand = cfg.Command
	}

	if cfg.Shell != "" || finalCommand != "" {
		if cfg.Shell != "" && finalCommand != "" {
			cmd = append(cmd, fmt.Sprintf("%s -c %q", cfg.Shell, finalCommand))
		} else if cfg.Shell != "" {
			cmd = append(cmd, cfg.Shell)
		} else {
			cmd = append(cmd, finalCommand)
		}
	}

	return strings.Join(cmd, " && ")
}
