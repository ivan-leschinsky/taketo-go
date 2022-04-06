package main

import (
	"testing"
)

func TestBuildCommand(t *testing.T) {
	testCases := []struct {
		Description string
		Input       Server
		Expected    string
	}{
		{
			Description: "empty Server",
			Input: Server{
				Shell:    "",
				Location: "",
				Command:  "",
				Env:      []string{},
			},
			Expected: "",
		},
		{
			Description: "only envs",
			Input: Server{
				Shell:    "",
				Location: "",
				Command:  "",
				Env:      []string{"VAR_1=123"},
			},
			Expected: "export VAR_1=123",
		},
		{
			Description: "only command",
			Input: Server{
				Shell:    "",
				Location: "",
				Command:  "ls -lha",
				Env:      []string{},
			},
			Expected: "ls -lha",
		},
		{
			Description: "command and envs",
			Input: Server{
				Shell:    "",
				Location: "",
				Command:  "ls -lha && df -h",
				Env:      []string{"VAR_1=123", "VAR_2=456"},
			},
			Expected: "export VAR_1=123 && export VAR_2=456 && ls -lha && df -h",
		},
		{
			Description: "only location",
			Input: Server{
				Shell:    "",
				Location: "/data",
				Command:  "",
				Env:      []string{},
			},
			Expected: "cd /data",
		},
		{
			Description: "location and envs",
			Input: Server{
				Shell:    "",
				Location: "/data",
				Command:  "",
				Env:      []string{"VAR_1=123"},
			},
			Expected: "export VAR_1=123 && cd /data",
		},
		{
			Description: "location and command",
			Input: Server{
				Shell:    "",
				Location: "/data",
				Command:  "ls -lha",
				Env:      []string{},
			},
			Expected: "cd /data && ls -lha",
		},
		{
			Description: "location, command and envs",
			Input: Server{
				Shell:    "",
				Location: "/data",
				Command:  "wg genkey | sudo tee /etc/wireguard/wg0.conf",
				Env:      []string{"VAR_1=123", "VAR_2=456"},
			},
			Expected: "export VAR_1=123 && export VAR_2=456 && cd /data && wg genkey | sudo tee /etc/wireguard/wg0.conf",
		},
		{
			Description: "only shell",
			Input: Server{
				Shell:    "zsh",
				Location: "",
				Command:  "",
				Env:      []string{},
			},
			Expected: "zsh",
		},
		{
			Description: "shell and envs",
			Input: Server{
				Shell:    "zsh",
				Location: "",
				Command:  "",
				Env:      []string{"VAR_1=123"},
			},
			Expected: "export VAR_1=123 && zsh",
		},
		{
			Description: "shell and command",
			Input: Server{
				Shell:    "zsh",
				Location: "",
				Command:  "ls -lha",
				Env:      []string{},
			},
			Expected: "zsh -c \"ls -lha\"",
		},
		{
			Description: "shell, command and envs",
			Input: Server{
				Shell:    "zsh",
				Location: "",
				Command:  "ls -1 | xargs rm -f",
				Env:      []string{"VAR_1=123"},
			},
			Expected: "export VAR_1=123 && zsh -c \"ls -1 | xargs rm -f\"",
		},
		{
			Description: "shell and location",
			Input: Server{
				Shell:    "zsh",
				Location: "/data",
				Command:  "",
				Env:      []string{},
			},
			Expected: "cd /data && zsh",
		},
		{
			Description: "shell, location and envs",
			Input: Server{
				Shell:    "zsh",
				Location: "/data",
				Command:  "",
				Env:      []string{"VAR_1=123"},
			},
			Expected: "export VAR_1=123 && cd /data && zsh",
		},
		{
			Description: "shell, location and command",
			Input: Server{
				Shell:    "zsh",
				Location: "/data",
				Command:  "ls -lha",
				Env:      []string{},
			},
			Expected: "cd /data && zsh -c \"ls -lha\"",
		},
		{
			Description: "shell, location, command and envs",
			Input: Server{
				Shell:    "zsh",
				Location: "/data",
				Command:  "ls -lha",
				Env:      []string{"VAR_1=123"},
			},
			Expected: "export VAR_1=123 && cd /data && zsh -c \"ls -lha\"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			got := buildCommand(&tc.Input)
			want := tc.Expected

			if got != want {
				t.Errorf("got: %q, want: %q, given: %v", got, want, tc.Input)
			}
		})
	}
}
