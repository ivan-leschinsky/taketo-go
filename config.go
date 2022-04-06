package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v3"
)

type Server struct {
	Name     string   `yaml:"name"`
	Alias    string   `yaml:"alias"`
	Host     string   `yaml:"host"`
	User     string   `yaml:"user"`
	Shell    string   `yaml:"shell"`
	Location string   `yaml:"location"`
	Command  string   `yaml:"command"`
	Env      []string `yaml:"env"`
}

type Environment struct {
	Name       string    `yaml:"name"`
	Servers    []*Server `yaml:"servers"`
}

type Project struct {
	Name         string         `yaml:"name"`
	Environments []*Environment `yaml:"environments"`
}

type Config struct {
	Projects []*Project `yaml:"projects"`
}

func findServer(projects []*Project, serverAlias string) *Server {
	for _, project := range projects {
		for _, environment := range project.Environments {
			for _, server := range environment.Servers {
				if server.Alias == serverAlias {
					return server;
				}
			}
		}
	}

	var emptyServer Server = Server{}
	return &emptyServer;
}

func readConf(fpath string, serverAlias string, overrideCommand string) (*Server, error) {
	buf, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}

	entireCfg := &Config{}

	err = yaml.Unmarshal(buf, entireCfg)
	if err != nil {
		return nil, err
	}

	cfg := findServer(entireCfg.Projects, serverAlias)

	if overrideCommand != "" {
		cfg.Command = overrideCommand
	}

	cfg.Command = buildCommand(cfg)

	return cfg, nil
}

func buildCommand(cfg *Server) string {
	var cmd []string

	if len(cfg.Env) > 0 {
		for _, val := range cfg.Env {
			cmd = append(cmd, "export "+val)
		}
	}

	if cfg.Location != "" {
		cmd = append(cmd, "cd "+cfg.Location)
	}

	if cfg.Shell != "" || cfg.Command != "" {
		if cfg.Shell != "" && cfg.Command != "" {
			cmd = append(cmd, fmt.Sprintf("%s -c %q", cfg.Shell, cfg.Command))
		} else if cfg.Shell != "" {
			cmd = append(cmd, cfg.Shell)
		} else {
			cmd = append(cmd, cfg.Command)
		}
	}

	return strings.Join(cmd, " && ")
}
