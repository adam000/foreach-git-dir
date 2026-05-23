package action

import (
	"os/exec"
	"strings"
)

type CustomAction struct {
	name    string
	command string
}

var _ Action = &CustomAction{}

func (c *CustomAction) Name() string {
	return c.name
}

func (c *CustomAction) Summary() string {
	return c.command
}

func (c *CustomAction) Run(repoPath string) (string, error) {
	cmd := exec.Command("sh", "-c", c.command)
	cmd.Dir = repoPath
	output, err := cmd.Output()

	return strings.Trim(string(output), " \t"), err
}

func NewCustomAction(command string) *CustomAction {
	return &CustomAction{
		name:    "-custom",
		command: command,
	}
}
