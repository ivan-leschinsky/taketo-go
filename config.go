package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
)

type Server struct {
	Name           string    `yaml:"name"`
	Alias          string    `yaml:"alias"`
	Aliases        []string  `yaml:"aliases"`
	Host           string    `yaml:"host"`
	Hosts          []string  `yaml:"hosts"`
	User           string    `yaml:"user"`
	Port           string    `yaml:"port"`
	Shell          string    `yaml:"shell"`
	Location       string    `yaml:"location"`
	Command        string    `yaml:"command"`
	Env            []string  `yaml:"env"`
	Identity       string    `yaml:"identity"`
	Forward        []string  `yaml:"forward"`
	SSHOptions     []string  `yaml:"ssh_options"`
	ConnectTimeout int       `yaml:"connect_timeout"`
	Targets        []*Server `yaml:"targets"`
}

type Defaults struct {
	Host           string   `yaml:"host"`
	Hosts          []string `yaml:"hosts"`
	User           string   `yaml:"user"`
	Port           string   `yaml:"port"`
	Shell          string   `yaml:"shell"`
	Location       string   `yaml:"location"`
	Command        string   `yaml:"command"`
	Env            []string `yaml:"env"`
	Identity       string   `yaml:"identity"`
	Forward        []string `yaml:"forward"`
	SSHOptions     []string `yaml:"ssh_options"`
	ConnectTimeout int      `yaml:"connect_timeout"`
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

type NamedCommand struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
}

type Config struct {
	Commands []*NamedCommand `yaml:"commands"`
	Projects []*Project      `yaml:"projects"`
}

type ServersMapping struct {
	byAlias map[string]*Server
	byPath  map[string]*Server
	all     []*Server
}

var serversMapping = &ServersMapping{}
var namedCommands = map[string]string{}

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
	if len(server.Hosts) == 0 {
		server.Hosts = defaults.Hosts
	}
	if server.Shell == "" {
		server.Shell = defaults.Shell
	}
	if server.Location == "" {
		server.Location = defaults.Location
	}
	if server.Command == "" {
		server.Command = defaults.Command
	}
	if len(server.Env) == 0 {
		server.Env = defaults.Env
	}
	if server.Identity == "" {
		server.Identity = defaults.Identity
	}
	if len(server.Forward) == 0 {
		server.Forward = defaults.Forward
	}
	if len(server.SSHOptions) == 0 {
		server.SSHOptions = defaults.SSHOptions
	}
	if server.ConnectTimeout == 0 {
		server.ConnectTimeout = defaults.ConnectTimeout
	}
}

// inheritServer fills a target's empty fields from its parent server (which has
// already had project/environment defaults applied).
func inheritServer(target, parent *Server) {
	if target.User == "" {
		target.User = parent.User
	}
	if target.Port == "" {
		target.Port = parent.Port
	}
	if target.Host == "" {
		target.Host = parent.Host
	}
	if len(target.Hosts) == 0 {
		target.Hosts = parent.Hosts
	}
	if target.Shell == "" {
		target.Shell = parent.Shell
	}
	if target.Location == "" {
		target.Location = parent.Location
	}
	if target.Command == "" {
		target.Command = parent.Command
	}
	if len(target.Env) == 0 {
		target.Env = parent.Env
	}
	if target.Identity == "" {
		target.Identity = parent.Identity
	}
	if len(target.Forward) == 0 {
		target.Forward = parent.Forward
	}
	if len(target.SSHOptions) == 0 {
		target.SSHOptions = parent.SSHOptions
	}
	if target.ConnectTimeout == 0 {
		target.ConnectTimeout = parent.ConnectTimeout
	}
}

// serverAliases returns every alias a server responds to (alias + aliases).
func serverAliases(s *Server) []string {
	var out []string
	if s.Alias != "" {
		out = append(out, s.Alias)
	}
	out = append(out, s.Aliases...)
	return out
}

// resolveCommand expands a command that references a named command from the
// top-level commands: block.
func resolveCommand(s *Server) {
	if s.Command == "" {
		return
	}
	if v, ok := namedCommands[s.Command]; ok {
		s.Command = v
	}
}

func registerAlias(alias string, server *Server) {
	if serversMapping.byAlias[alias] != nil {
		exit(fmt.Errorf("invalid config: alias %q declared twice", alias))
	}
	serversMapping.byAlias[alias] = server
}

func registerPath(path string, server *Server) {
	if serversMapping.byPath[path] != nil {
		exit(fmt.Errorf("invalid config: server with path %q declared twice", path))
	}
	serversMapping.byPath[path] = server
}

func registerServer(server *Server, project *Project, environment *Environment, path string) {
	if environment != nil {
		fillEmpty(server, environment.Defaults)
	}
	fillEmpty(server, project.Defaults)
	resolveCommand(server)

	for _, alias := range serverAliases(server) {
		registerAlias(alias, server)
	}
	registerPath(path, server)
	serversMapping.all = append(serversMapping.all, server)

	for _, target := range server.Targets {
		inheritServer(target, server)
		resolveCommand(target)

		for _, alias := range serverAliases(target) {
			registerAlias(alias, target)
		}
		if target.Name != "" {
			registerPath(path+":"+target.Name, target)
		}
		serversMapping.all = append(serversMapping.all, target)
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
	serversMapping.all = nil

	namedCommands = make(map[string]string)
	for _, c := range cfg.Commands {
		namedCommands[c.Name] = c.Command
	}

	for _, project := range cfg.Projects {
		for _, server := range project.Servers {
			registerServer(server, project, nil, project.Name+":"+server.Name)
		}
		for _, environment := range project.Environments {
			for _, server := range environment.Servers {
				registerServer(server, project, environment, project.Name+":"+environment.Name+":"+server.Name)
			}
		}
	}
}

// findServer resolves a server by alias first, then by path. Returns nil when
// nothing matches (callers may fall back to the interactive picker).
func findServer(serverPath string) *Server {
	if server := serversMapping.byAlias[serverPath]; server != nil {
		return server
	}
	return serversMapping.byPath[serverPath]
}

func readConf(fpath, serverAlias, overrideCommand string) (*Server, error) {
	loadConfig(fpath)

	serverConfig := findServer(serverAlias)
	if serverConfig == nil {
		serverConfig = pickServer(serverAlias)
	}

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

	printTargets := func(prefix string, s *Server) {
		indent := strings.Repeat(" ", len([]rune(prefix)))
		for _, tg := range s.Targets {
			tc := *tg
			inheritServer(&tc, s)
			fmt.Printf("  %s   ↳ ", indent)
			green.Printf("%-14s", "["+strings.Join(serverAliases(&tc), ",")+"]")
			if tc.Location != "" {
				dim.Printf(" %s", tc.Location)
			}
			if tc.Command != "" {
				dim.Printf(" » %s", tc.Command)
			}
			fmt.Println()
		}
	}

	printServer := func(prefix string, s *Server, projDefaults, envDefaults *Defaults) {
		sc := *s
		fillEmpty(&sc, envDefaults)
		fillEmpty(&sc, projDefaults)

		host := sc.Host
		if host == "" && len(sc.Hosts) > 0 {
			host = sc.Hosts[0]
		}

		fmt.Printf("  %s ", prefix)
		bold.Printf("%-20s", sc.Name)
		green.Printf(" %-14s", "["+strings.Join(serverAliases(&sc), ",")+"]")
		dim.Printf(" %s@%s", sc.User, host)
		if len(sc.Hosts) > 1 {
			dim.Printf(" (+%d)", len(sc.Hosts)-1)
		}
		if sc.Port != "" {
			dim.Printf(":%s", sc.Port)
		}
		fmt.Println()

		printTargets(prefix, &sc)
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
