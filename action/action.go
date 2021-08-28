package action

import (
	"fmt"
	"os/exec"
	"strings"
)

type Action func(string, *strings.Builder) error

func Id(_ string, _ *strings.Builder) error {
	return nil
}

func PrintBriefStatus(root string, output *strings.Builder) error {
	// Run `git status -sb` and store results in `status`
	cmd := exec.Command("git", "status", "-sb")
	cmd.Dir = root
	stdout, err := cmd.Output()
	if err != nil {
		return err
	}
	fmt.Fprintf(output, strings.TrimSpace(string(stdout)))

	return nil
}
