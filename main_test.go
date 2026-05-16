package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	originalOsExit := osExit
	defer func() { osExit = originalOsExit }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	var exitCode int
	osExit = func(code int) {
		exitCode = code
		panic("os.Exit called")
	}

	os.Args = []string{"program", "--version"}

	func() {
		defer func() { recover() }() //nolint:errcheck
		parseArguments()
	}()

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	expected := []string{
		"taketo-go version " + version,
		"Git commit: " + commit,
		"Built: " + date,
	}
	for _, want := range expected {
		if !strings.Contains(output, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}

func TestVersionShortFlag(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	originalOsExit := osExit
	defer func() { osExit = originalOsExit }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	var exitCode int
	osExit = func(code int) {
		exitCode = code
		panic("os.Exit called")
	}

	os.Args = []string{"program", "-v"}

	func() {
		defer func() { recover() }() //nolint:errcheck
		parseArguments()
	}()

	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	if !strings.Contains(output, "taketo-go version") {
		t.Errorf("expected version string in output, got:\n%s", output)
	}
}

func TestMainExecutesSSHCommand(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tmpConfig := `projects:
  - name: testproject
    servers:
      - name: testserver
        alias: test
        host: example.com
        user: testuser
        port: "2222"
        command: "ls -la"
`
	originalHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	configPath := tmpDir + "/.taketo.yml"
	if err := os.WriteFile(configPath, []byte(tmpConfig), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()

	testCases := []struct {
		name         string
		args         []string
		expectedArgs []string
	}{
		{
			name: "connection by alias",
			args: []string{"program", "test"},
			expectedArgs: []string{
				"testuser@example.com",
				"-p", "2222",
				"-t", "ls -la",
			},
		},
		{
			name: "connection by path",
			args: []string{"program", "testproject:testserver"},
			expectedArgs: []string{
				"testuser@example.com",
				"-p", "2222",
				"-t", "ls -la",
			},
		},
		{
			name: "connection with override command",
			args: []string{"program", "test", "-c", "echo hello"},
			expectedArgs: []string{
				"testuser@example.com",
				"-p", "2222",
				"-t", "echo hello",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Args = tc.args

			var executedCommand string
			var executedArgs []string

			execCommand = func(command string, args ...string) *exec.Cmd {
				executedCommand = command
				executedArgs = args
				return &exec.Cmd{
					Stdout: os.Stdout,
					Stderr: os.Stderr,
					Stdin:  os.Stdin,
				}
			}

			main()

			if executedCommand != "ssh" {
				t.Errorf("expected command 'ssh', got %s", executedCommand)
			}

			if len(executedArgs) != len(tc.expectedArgs) {
				t.Errorf("expected %d arguments, got %d\nexpected: %v\ngot:      %v",
					len(tc.expectedArgs), len(executedArgs), tc.expectedArgs, executedArgs)
				return
			}

			for i, want := range tc.expectedArgs {
				if executedArgs[i] != want {
					t.Errorf("arg[%d]: expected %q, got %q", i, want, executedArgs[i])
				}
			}
		})
	}
}
