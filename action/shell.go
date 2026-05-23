package action

import (
	"os/exec"
	"strings"
)

type ShellAction struct {
	name    string
	command string
}

var _ Action = &ShellAction{}

func (s *ShellAction) Name() string {
	return s.name
}

func (s *ShellAction) Summary() string {
	return s.command
}

func (s *ShellAction) Run(repoPath string) (string, error) {
	words := strings.Fields(s.command)
	cmd := exec.Command(words[0], words[1:]...)
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func DefaultShellActions() map[string]Action {
	return map[string]Action{
		"-status": &ShellAction{
			name:    "-status",
			command: "git status",
		},
		"-shortstatus": &ShellAction{
			name:    "-shortStatus",
			command: "git status -sb",
		},
		"-stashes": &ShellAction{
			name:    "-stashes",
			command: "git stash list",
		},
		"-fetch": &ShellAction{
			name:    "-fetch",
			command: "git fetch",
		},
		"-fetchall": &ShellAction{
			name:    "-fetchAll",
			command: "git fetch --all",
		},
		"-issues": &ShellAction{
			name:    "-issues",
			command: "gh issue list",
		},
	}
}
