package action

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

type CustomAction struct {
	name    string
	command string
	shell   []string
}

var _ Action = &CustomAction{}

func (c *CustomAction) Name() string {
	return c.name
}

func (c *CustomAction) Summary() string {
	return c.command
}

func (c *CustomAction) Run(repoPath string) (string, error) {
	command := append(c.shell, c.command)
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = repoPath
	output, err := cmd.Output()

	if err != nil {
		slog.Error("Error occurred while running command", "shell", c.shell, "command", c.command, "error", fmt.Sprintf("%v", err))
	}

	return strings.Trim(string(output), " \t"), err
}

func NewCustomAction(command string, shell []string) *CustomAction {
	return &CustomAction{
		name:    "-custom",
		command: command,
		shell:   shell,
	}
}
