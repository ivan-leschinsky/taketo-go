package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

type Server struct {
	Name     string   `yaml:"name"`
	Alias    string   `yaml:"alias"`
	Host     string   `yaml:"host"`
	User     string   `yaml:"user"`
	Port     string   `yaml:"port"`
	Shell    string   `yaml:"shell"`
	Location string   `yaml:"location"`
	Command  string   `yaml:"command"`
	Env      []string `yaml:"env"`
}

type Defaults struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Port     string `yaml:"port"`
	Shell    string `yaml:"shell"`
	Location string `yaml:"location"`
}

type Environment struct {
	Name     string    `yaml:"name"`
	Servers  []*Server `yaml:"servers"`
	Defaults *Defaults `yaml:"defaults"`
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
	byPath  map[string]*Server
}

var serversMapping = &ServersMapping{}

func fillEmpty(server *Server, defaults *Defaults) {
	if defaults == nil {
		return
	}

	if server.User == "" {
		server.User = defaults.User
	}
	if server.Port == "" {
		server.Port = defaults.Port
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
		exit(fmt.Errorf("invalid config: alias %q declared twice", server.Alias))
	} else {
		serversMapping.byAlias[server.Alias] = server
	}
	serverPath := fmt.Sprintf("%v:%v", project.Name, server.Name)

	if environment != nil {
		serverPath = fmt.Sprintf("%v:%v:%v", project.Name, environment.Name, server.Name)
	}

	if serversMapping.byPath[serverPath] != nil {
		exit(fmt.Errorf("invalid config: server with path %q declared twice", serverPath))
	} else {
		serversMapping.byPath[serverPath] = server
	}
}

func loadConfig(fpath string) {
	buf, err := os.ReadFile(fpath)
	if err != nil {
		exit(fmt.Errorf("failed to read config file from %v", fpath))
		return
	}

	cfg := &Config{}

	err = yaml.Unmarshal(buf, cfg)
	if err != nil {
		exit(fmt.Errorf("failed to parse YAML from %v", fpath))
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
	if server == nil {
		server = serversMapping.byPath[serverPath]
	}

	if server == nil {
		exit(fmt.Errorf("server not found for alias or path: %v", serverPath))
	}

	return server
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

func listServers(fpath string) {
	buf, err := os.ReadFile(fpath)
	if err != nil {
		exit(fmt.Errorf("failed to read config file from %v", fpath))
		return
	}

	cfg := &Config{}
	if err = yaml.Unmarshal(buf, cfg); err != nil {
		exit(fmt.Errorf("failed to parse YAML from %v", fpath))
		return
	}

	cyanBold := color.New(color.FgCyan, color.Bold)
	yellow := color.New(color.FgYellow, color.Bold)
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	dim := color.New(color.FgHiBlack)

	printServer := func(prefix string, s *Server, projDefaults, envDefaults *Defaults) {
		sc := *s
		fillEmpty(&sc, envDefaults)
		fillEmpty(&sc, projDefaults)

		fmt.Printf("  %s ", prefix)
		bold.Printf("%-20s", sc.Name)
		green.Printf(" %-14s", "["+sc.Alias+"]")
		dim.Printf(" %s@%s", sc.User, sc.Host)
		if sc.Port != "" {
			dim.Printf(":%s", sc.Port)
		}
		fmt.Println()
	}

	for pi, project := range cfg.Projects {
		if pi > 0 {
			fmt.Println()
		}
		cyanBold.Printf("  %s\n", project.Name)

		total := len(project.Servers) + len(project.Environments)
		idx := 0

		for _, s := range project.Servers {
			idx++
			conn := "├──"
			if idx == total {
				conn = "└──"
			}
			printServer(conn, s, project.Defaults, nil)
		}

		for _, env := range project.Environments {
			idx++
			isLastEnv := idx == total
			envConn := "├──"
			if isLastEnv {
				envConn = "└──"
			}
			fmt.Printf("  %s ", envConn)
			yellow.Printf("%s\n", env.Name)

			for si, s := range env.Servers {
				isLast := si == len(env.Servers)-1
				var serverConn string
				switch {
				case isLastEnv && isLast:
					serverConn = "    └──"
				case isLastEnv:
					serverConn = "    ├──"
				case isLast:
					serverConn = "│   └──"
				default:
					serverConn = "│   ├──"
				}
				printServer(serverConn, s, project.Defaults, env.Defaults)
			}
		}
	}
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
