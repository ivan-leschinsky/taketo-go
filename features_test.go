package main

import (
	"os"
	"testing"
	"time"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := dir + "/.taketo.yml"
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return p
}

func TestMultipleAliases(t *testing.T) {
	cfg := `projects:
  - name: p
    servers:
      - name: s
        alias: main
        aliases: [a1, a2]
        host: h
        user: u
`
	loadConfig(writeTempConfig(t, cfg))

	for _, a := range []string{"main", "a1", "a2"} {
		if got := findServer(a); got == nil || got.Name != "s" {
			t.Errorf("alias %q did not resolve to server s", a)
		}
	}
}

func TestTargetsInheritAndOverride(t *testing.T) {
	cfg := `projects:
  - name: poly
    defaults:
      user: root
      shell: fish -l
    servers:
      - name: vm5
        alias: vm5
        host: 1.2.3.4
        targets:
          - alias: p
            location: /root/btc
          - aliases: [logs, lg]
            location: /root/btc
            command: ./log
`
	loadConfig(writeTempConfig(t, cfg))

	base := findServer("p")
	if base == nil {
		t.Fatal("target alias p not found")
	}
	if base.Host != "1.2.3.4" || base.User != "root" || base.Shell != "fish -l" {
		t.Errorf("target p did not inherit parent/defaults: %+v", base)
	}
	if base.Location != "/root/btc" {
		t.Errorf("target p location override failed: %q", base.Location)
	}

	for _, a := range []string{"logs", "lg"} {
		tg := findServer(a)
		if tg == nil {
			t.Fatalf("target alias %q not found", a)
		}
		if tg.Command != "./log" || tg.Host != "1.2.3.4" || tg.User != "root" {
			t.Errorf("target %q wrong: %+v", a, tg)
		}
	}
}

func TestDefaultsCommandAndEnv(t *testing.T) {
	cfg := `projects:
  - name: p
    defaults:
      user: u
      command: uptime
      env: [A=1, B=2]
    servers:
      - name: s
        alias: s
        host: h
`
	loadConfig(writeTempConfig(t, cfg))

	s := findServer("s")
	if s.Command != "uptime" {
		t.Errorf("default command not applied: %q", s.Command)
	}
	if len(s.Env) != 2 || s.Env[0] != "A=1" || s.Env[1] != "B=2" {
		t.Errorf("default env not applied: %v", s.Env)
	}
}

func TestNamedCommandResolution(t *testing.T) {
	cfg := `commands:
  - name: log
    command: tail -f log/production.log
projects:
  - name: p
    servers:
      - name: s
        alias: s
        host: h
        user: u
        command: log
`
	loadConfig(writeTempConfig(t, cfg))

	s := findServer("s")
	if s.Command != "tail -f log/production.log" {
		t.Errorf("named command not resolved: %q", s.Command)
	}
}

func TestSelectHostSingleNoProbe(t *testing.T) {
	orig := probeHost
	defer func() { probeHost = orig }()
	probeHost = func(addr string, to time.Duration) bool {
		t.Fatalf("probe must not run for single host, got %q", addr)
		return false
	}

	if got := selectHost([]string{"only"}, "22", time.Second); got != "only" {
		t.Errorf("got %q, want only", got)
	}
}

func TestSelectHostFallbackPicksFirstReachable(t *testing.T) {
	orig := probeHost
	defer func() { probeHost = orig }()
	probeHost = func(addr string, to time.Duration) bool {
		return addr == "up2:22" || addr == "up3:22"
	}

	got := selectHost([]string{"down1", "up2", "up3"}, "22", time.Second)
	if got != "up2" {
		t.Errorf("got %q, want up2", got)
	}
}

func TestSelectHostAllUnreachableUsesFirst(t *testing.T) {
	orig := probeHost
	defer func() { probeHost = orig }()
	probeHost = func(addr string, to time.Duration) bool { return false }

	got := selectHost([]string{"a", "b"}, "22", time.Second)
	if got != "a" {
		t.Errorf("got %q, want a", got)
	}
}

func TestBuildSSHArgs(t *testing.T) {
	s := &Server{
		User:       "u",
		Port:       "2222",
		Command:    "ls -la",
		Identity:   "/k",
		Forward:    []string{"8080:localhost:80"},
		SSHOptions: []string{"-o", "KexAlgorithms=+diffie-hellman-group1-sha1"},
		SSHEnv:     []string{"TERM=xterm-256color", "LANG=C.UTF-8"},
	}
	got := buildSSHArgs(s, "host")
	want := []string{
		"u@host",
		"-o", "KexAlgorithms=+diffie-hellman-group1-sha1",
		"-o", "SetEnv=TERM=xterm-256color LANG=C.UTF-8",
		"-i", "/k",
		"-L", "8080:localhost:80",
		"-p", "2222",
		"-t", "ls -la",
	}
	if len(got) != len(want) {
		t.Fatalf("len mismatch\n got: %v\nwant: %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d]: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestBuildSSHArgsMinimal(t *testing.T) {
	s := &Server{User: "u"}
	got := buildSSHArgs(s, "host")
	if len(got) != 1 || got[0] != "u@host" {
		t.Errorf("minimal args wrong: %v", got)
	}
}

func TestPickServerUsesTypedQuery(t *testing.T) {
	cfg := `projects:
  - name: p
    servers:
      - name: s
        alias: a1
        host: h
        user: u
`
	loadConfig(writeTempConfig(t, cfg))

	orig := runFuzzy
	defer func() { runFuzzy = orig }()
	var gotQuery string
	runFuzzy = func(items []*Server, query string) (int, error) {
		gotQuery = query
		return 0, nil
	}

	s := pickServer("typed")
	if gotQuery != "typed" {
		t.Errorf("picker query: got %q want typed", gotQuery)
	}
	if s == nil || s.Name != "s" {
		t.Errorf("picker returned wrong server: %+v", s)
	}
}

func TestFewerNames(t *testing.T) {
	cfg := `projects:
  - name: home
    defaults: { user: root }
    servers:
      - name: nvr                 # name only -> also the alias
        host: 100.75.112.42
      - aliases: [syno, s]        # aliases only -> name = first alias
        host: 192.168.1.235
        targets:
          - { name: logs, location: /var/log }   # target name -> alias
      - name: web                 # explicit alias wins, name is not claimed
        alias: w
        host: h
`
	p := writeTempConfig(t, cfg)
	loadConfig(p)

	if s := findServer("nvr"); s == nil || s.Host != "100.75.112.42" {
		t.Fatalf("name-only server not reachable by its name: %+v", s)
	}
	if s := findServer("s"); s == nil || s.Name != "syno" {
		t.Fatalf("aliases-only server should get name from first alias: %+v", s)
	}
	if s := findServer("home:syno"); s == nil {
		t.Fatal("path from defaulted name missing")
	}
	if s := findServer("logs"); s == nil || s.Location != "/var/log" || s.User != "root" {
		t.Fatalf("target name should act as alias and inherit: %+v", s)
	}
	if s := findServer("web"); s != nil {
		t.Fatalf("server with explicit alias must not also claim its name: %+v", s)
	}
}

func TestFewerNamesNameTakenIsNotFatal(t *testing.T) {
	// Two alias-less servers with the same name in different environments
	// (old configs) must still load; the first one keeps the bare name.
	cfg := `projects:
  - name: p
    environments:
      - name: stg
        servers: [{ name: web, host: a }]
      - name: prd
        servers: [{ name: web, host: b }]
`
	loadConfig(writeTempConfig(t, cfg))
	if s := findServer("web"); s == nil || s.Host != "a" {
		t.Fatalf("first server should own the bare name: %+v", s)
	}
	if s := findServer("p:prd:web"); s == nil || s.Host != "b" {
		t.Fatalf("second server still reachable by path: %+v", s)
	}
}

func TestSSHEnvFromDefaults(t *testing.T) {
	cfg := `projects:
  - name: p
    defaults: { user: u, ssh_env: [TERM=xterm-256color] }
    servers:
      - name: s
        host: h
        targets: [{ name: t, location: /x }]
`
	loadConfig(writeTempConfig(t, cfg))
	for _, a := range []string{"s", "t"} {
		if got := findServer(a).SSHEnv; len(got) != 1 || got[0] != "TERM=xterm-256color" {
			t.Errorf("%s: ssh_env not inherited: %v", a, got)
		}
	}
}
