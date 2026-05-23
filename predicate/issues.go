package predicate

import (
	"fmt"
	"os"
	"strings"

	"github.com/adam000/foreach-git-dir/gh"
)

var runner gh.Runner = gh.ExecRunner{}

func HasIssues(root string) (bool, error) {
	if _, err := runner.LookPath("gh"); err != nil {
		fmt.Fprintf(os.Stderr, "gh (GitHub CLI) is not installed. Please install it from https://cli.github.com\n")
		return false, nil
	}

	cmd := runner.Command("gh", "repo", "view")
	cmd.Dir = root
	_, err := cmd.Output()
	if err != nil {
		// If we can't view the repo, it's either not a GitHub repo or we don't have permission
		return false, nil
	}

	cmd = runner.Command("gh", "issue", "list")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		lowered := strings.ToLower(string(out))
		if strings.Contains(lowered, "has disabled issues") {
			return false, nil
		}
		if strings.Contains(lowered, "no open issues") {
			return false, nil
		}
		return false, err
	}

	return len(strings.TrimSpace(string(out))) != 0, nil
}
