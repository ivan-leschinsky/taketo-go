package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Create a testContext to hold our testing state
type testContext struct {
	exitCode int
	exited   bool
}

func TestVersionFlag(t *testing.T) {
	// Save original args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	testCases := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name: "long version flag",
			args: []string{"program", "--version"},
			expected: []string{
				"taketo-go version " + version,
				"Git commit: " + commit,
				"Built: " + date,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up test args
			os.Args = tc.args

			// Capture panic from displayVersion
			defer func() {
				if r := recover(); r == nil {
					t.Error("Expected panic from displayVersion, got none")
				}

				// Close pipe and read output
				w.Close()
				var buf bytes.Buffer
				_, _ = buf.ReadFrom(r)
				output := buf.String()

				// Check each expected line
				for _, expected := range tc.expected {
					if !strings.Contains(output, expected) {
						t.Errorf("Expected output to contain %q", expected)
					}
				}
			}()

			// Call the function being tested
			parseArguments()
		})
	}
}

func TestMainExecutesSSHCommand(t *testing.T) {
	// Save original args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Create a temporary config file
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
	// Save original HOME env var and set it to temp directory
	originalHome := os.Getenv("HOME")
	tmpDir := os.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create .taketo.yml in temp HOME directory
	configPath := tmpDir + "/.taketo.yml"

	if err := os.WriteFile(configPath, []byte(tmpConfig), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	defer func() {
		if err := os.Remove(configPath); err != nil {
			t.Logf("Failed to remove test config: %v", err)
		}
	}()

	// Save original execCommand
	originalExecCommand := execCommand
	defer func() { execCommand = originalExecCommand }()

	testCases := []struct {
		name         string
		args         []string
		expectedArgs []string
	}{
		{
			name: "basic server connection",
			args: []string{"program", "test"},
			expectedArgs: []string{
				"testuser@example.com",
				"-p",
				"2222",
				"-t",
				"ls -la",
			},
		},
		{
			name: "basic server connection",
			args: []string{"program", "testproject:testserver"},
			expectedArgs: []string{
				"testuser@example.com",
				"-p",
				"2222",
				"-t",
				"ls -la",
			},
		},
		{
			name: "server with override command",
			args: []string{"program", "test", "-c", "echo hello"},
			expectedArgs: []string{
				"testuser@example.com",
				"-p",
				"2222",
				"-t",
				"echo hello",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up test args
			os.Args = tc.args

			var executedCommand string
			var executedArgs []string

			// Mock execCommand
			execCommand = func(command string, args ...string) *exec.Cmd {
				executedCommand = command
				executedArgs = args
				return &exec.Cmd{
					Stdout: os.Stdout,
					Stderr: os.Stderr,
					Stdin:  os.Stdin,
				}
			}

			// Run main
			main()

			// Verify the command
			if executedCommand != "ssh" {
				t.Errorf("Expected command 'ssh', got %s", executedCommand)
			}

			// Verify arguments
			if len(executedArgs) != len(tc.expectedArgs) {
				t.Errorf("Expected %d arguments, got %d\nExpected: %v\nGot: %v",
					len(tc.expectedArgs), len(executedArgs),
					tc.expectedArgs, executedArgs)
			}

			for i, arg := range tc.expectedArgs {
				if i >= len(executedArgs) {
					t.Errorf("Missing expected argument at position %d: %s", i, arg)
					continue
				}
				if executedArgs[i] != arg {
					t.Errorf("Expected argument %d to be %s, got %s", i, arg, executedArgs[i])
				}
			}
		})
	}
}
