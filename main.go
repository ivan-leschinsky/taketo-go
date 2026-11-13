package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/ktr0731/go-fuzzyfinder"
)

var execCommand = exec.Command
var osExit = os.Exit

const defaultConnectTimeout = 2 * time.Second

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func exit(err error) {
	color.Set(color.FgRed)
	log.Println(err)
	osExit(1)
}

func displayVersion() {
	fmt.Printf("taketo-go version %s\n", version)
	fmt.Printf("Git commit: %s\n", commit)
	fmt.Printf("Built: %s\n", date)
	osExit(0)
}

func parseArguments() (string, string) {
	if len(os.Args) < 2 {
		exit(errors.New("Expected at least one argument as server"))
	}

	var overrideCommand string
	var server = os.Args[1]

	if server == "--version" || server == "-v" {
		displayVersion()
	}

	if len(os.Args) > 2 {
		mySet := flag.NewFlagSet("", flag.ExitOnError)
		mySet.StringVar(&overrideCommand, "c", "", "command to run on server")
		mySet.Parse(os.Args[2:])
	}
	return server, overrideCommand
}

// probeHost reports whether a TCP connection to addr succeeds within timeout.
// It is a variable so tests can stub network access.
var probeHost = func(addr string, timeout time.Duration) bool {
	// macOS "Local Network" privacy blocks third-party binaries from the Mac's own
	// subnet (EHOSTUNREACH) while Apple's nc/ssh are allowed, so probe with nc there.
	if runtime.GOOS == "darwin" {
		host, port, _ := net.SplitHostPort(addr)
		secs := strconv.Itoa(int(math.Ceil(timeout.Seconds())))
		return exec.Command("/usr/bin/nc", "-z", "-G", secs, host, port).Run() == nil
	}
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func portOr22(port string) string {
	if port == "" {
		return "22"
	}
	return port
}

// selectHost returns the first reachable host from the list. With a single host
// it returns it immediately (no probing); if none are reachable it falls back to
// the first so ssh can surface the real error.
func selectHost(hosts []string, port string, timeout time.Duration) string {
	if len(hosts) == 0 {
		return ""
	}
	if len(hosts) == 1 {
		return hosts[0]
	}
	for _, h := range hosts {
		if probeHost(net.JoinHostPort(h, portOr22(port)), timeout) {
			return h
		}
	}
	return hosts[0]
}

func chooseHost(cfg *Server) string {
	hosts := cfg.Hosts
	if len(hosts) == 0 && cfg.Host != "" {
		hosts = []string{cfg.Host}
	}

	timeout := time.Duration(cfg.ConnectTimeout) * time.Second
	if timeout <= 0 {
		timeout = defaultConnectTimeout
	}

	return selectHost(hosts, cfg.Port, timeout)
}

func buildSSHArgs(cfg *Server, host string) []string {
	args := []string{fmt.Sprintf("%s@%s", cfg.User, host)}

	args = append(args, cfg.SSHOptions...)
	if len(cfg.SSHEnv) > 0 {
		// ponytail: values with spaces need ssh_config quoting, add when someone needs it
		args = append(args, "-o", "SetEnv="+strings.Join(cfg.SSHEnv, " "))
	}
	if cfg.Identity != "" {
		args = append(args, "-i", cfg.Identity)
	}
	for _, f := range cfg.Forward {
		args = append(args, "-L", f)
	}
	if cfg.Port != "" {
		args = append(args, "-p", cfg.Port)
	}
	if cfg.Command != "" {
		args = append(args, "-t", cfg.Command)
	}
	return args
}

func serverLabel(s *Server) string {
	host := s.Host
	if host == "" && len(s.Hosts) > 0 {
		host = s.Hosts[0]
	}
	return fmt.Sprintf("%-22s [%s] %s@%s", s.Name, strings.Join(serverAliases(s), ","), s.User, host)
}

// runFuzzy is the interactive picker; a variable so tests can stub it.
var runFuzzy = func(items []*Server, query string) (int, error) {
	return fuzzyfinder.Find(items, func(i int) string {
		return serverLabel(items[i])
	}, fuzzyfinder.WithQuery(query))
}

func pickServer(query string) *Server {
	if len(serversMapping.all) == 0 {
		exit(fmt.Errorf("server not found for alias or path: %v (no servers configured)", query))
	}
	idx, err := runFuzzy(serversMapping.all, query)
	if err != nil {
		exit(fmt.Errorf("server not found for alias or path: %v", query))
	}
	return serversMapping.all[idx]
}

func printAliases() {
	seen := map[string]bool{}
	for _, s := range serversMapping.all {
		for _, a := range serverAliases(s) {
			if !seen[a] {
				seen[a] = true
				fmt.Println(a)
			}
		}
	}
	paths := make([]string, 0, len(serversMapping.byPath))
	for p := range serversMapping.byPath {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		fmt.Println(p)
	}
}

func printCompletion(args []string) {
	shell := ""
	if len(args) > 0 {
		shell = args[0]
	}
	cmd := "taketo-go"
	if len(args) > 1 {
		cmd = args[1]
	}

	switch shell {
	case "zsh":
		fmt.Printf(`#compdef %[1]s
_%[1]s() {
  local -a items
  items=(${(f)"$(taketo-go aliases)"})
  compadd -a items
}
compdef _%[1]s %[1]s
`, cmd)
	case "bash":
		fmt.Printf(`_%[1]s() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  COMPREPLY=( $(compgen -W "$(taketo-go aliases)" -- "$cur") )
}
complete -F _%[1]s %[1]s
`, cmd)
	case "fish":
		fmt.Printf("complete -c %[1]s -f -a \"(taketo-go aliases)\"\n", cmd)
	default:
		exit(fmt.Errorf("usage: taketo-go completion <zsh|bash|fish> [command-name]"))
	}
}

func main() {
	log.SetFlags(0)

	configPath := fmt.Sprintf("%s/.taketo.yml", os.Getenv("HOME"))

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "ls", "list":
			listServers(configPath)
			return
		case "aliases":
			loadConfig(configPath)
			printAliases()
			return
		case "completion":
			printCompletion(os.Args[2:])
			return
		}
	}

	serverAlias, overrideCommand := parseArguments()

	cfg, err := readConf(configPath, serverAlias, overrideCommand)
	if err != nil {
		exit(err)
	}

	host := chooseHost(cfg)
	args := buildSSHArgs(cfg, host)

	cmd := execCommand("ssh", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			osExit(exitErr.ExitCode())
		}
		exit(fmt.Errorf("failed to run ssh: %w", err))
	}
}
