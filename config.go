package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"errors"

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

type Defaults struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Shell    string `yaml:"shell"`
	Location string `yaml:"location"`
}

type Environment struct {
	Name       string    `yaml:"name"`
	Servers    []*Server `yaml:"servers"`
	Defaults   *Defaults `yaml:"defaults"`
}

type Project struct {
	Name         string         `yaml:"name"`
	Environments []*Environment `yaml:"environments"`
	Defaults     *Defaults      `yaml:"defaults"`
	Servers      []*Server      `yaml:"servers"`
}

type Config struct {
	Projects []*Project `yaml:"projects"`
}

type ServersMapping struct {
	byAlias map[string]*Server
	byPath map[string]*Server
}

var serversMapping = &ServersMapping{}

func fillEmpty(server *Server, defaults *Defaults) {
	if defaults == nil {
		return
	}

	if server.User == "" {
		server.User = defaults.User
	}
	if server.Host == "" {
		server.Host = defaults.Host
	}
	if server.Shell == "" {
		server.Shell = defaults.Shell
	}
	if server.Location == "" {
		server.Location = defaults.Location
	}
}

func putServerToMapping(server *Server, project *Project, environment *Environment) {
	if environment != nil {
		fillEmpty(server, environment.Defaults)
	}
	fillEmpty(server, project.Defaults)

	if serversMapping.byAlias[server.Alias] != nil {
		exit(errors.New(fmt.Sprintf("Invalid config: alias \"%v\" declared twice", server.Alias)))
	} else {
		serversMapping.byAlias[server.Alias] = server
	}
	serverPath := fmt.Sprintf("%v:%v", project.Name, server.Name)

	if environment != nil {
		serverPath = fmt.Sprintf("%v:%v:%v", project.Name, environment.Name, server.Name)
	}

	if serversMapping.byPath[serverPath] != nil {
		exit(errors.New(fmt.Sprintf("Invalid config: server with path \"%v\" declared twice", serverPath)))
	} else {
		serversMapping.byPath[serverPath] = server
	}
}

func loadConfig(fpath string) {
	buf, err := ioutil.ReadFile(fpath)
	if err != nil {
		exit(errors.New(fmt.Sprintf("Failed to read config file from %v", fpath)))
		return
	}

	cfg := &Config{}

	err = yaml.Unmarshal(buf, cfg)
	if err != nil {
		exit(errors.New(fmt.Sprintf("Failed to load parse YAML from %v", fpath)))
		return
	}

	serversMapping.byAlias = make(map[string]*Server)
	serversMapping.byPath = make(map[string]*Server)

	for _, project := range cfg.Projects {
		for _, server := range project.Servers {
			putServerToMapping(server, project, nil)
		}
		for _, environment := range project.Environments {
			for _, server := range environment.Servers {
				putServerToMapping(server, project, environment)
			}
		}
	}
}

func findServer(serverPath string) *Server {
	server := serversMapping.byAlias[serverPath]
	if (server == nil) {
		server = serversMapping.byPath[serverPath]
	}

	if (server == nil) {
		exit(errors.New(fmt.Sprintf("Server not found for alias or path: %v", serverPath)))
	}

	return server;
}

func readConf(fpath, serverAlias, overrideCommand string) (*Server, error) {
	loadConfig(fpath)

	serverConfig := findServer(serverAlias)

	if overrideCommand != "" {
		serverConfig.Command = overrideCommand
	}

	serverConfig.Command = buildCommand(serverConfig)

	return serverConfig, nil
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
