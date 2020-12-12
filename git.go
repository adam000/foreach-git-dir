package main

import (
	"fmt"
	"strings"
)

func IsGitRoot(dir string) (bool, error) {
	// Try `git rev-parse --show-toplevel` at our current directory
	result, stderrText, err := RunInDir(dir, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		if !strings.Contains(stderrText, "not a git repository") {
			return false, fmt.Errorf("Failed to run git rev-parse: %w", err)
		}
		return false, nil
	}

	// Make sure we're in the base -- if we ever hit this error, we have... big problems
	// Not 100% sure this is an exactly accurate way to find this out, might need os.SameFile
	if strings.TrimSpace(result) != dir {
		return false, fmt.Errorf("Job directory '%s' is not the base git directory '%s'", dir, result)
	}

	return true, nil
}
