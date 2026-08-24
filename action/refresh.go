package action

import (
	"fmt"
	"os/exec"
	"strings"
)

type RefreshAction struct{}

var _ Action = &RefreshAction{}

func (r *RefreshAction) Name() string {
	return "-refresh"
}

func (r *RefreshAction) Summary() string {
	return "git pull if the repo is clean enough (e.g. only untracked changes), otherwise git fetch --all"
}

func (r *RefreshAction) Run(repoPath string) (string, error) {
	changedFiles, err := r.changedFiles(repoPath)
	if err != nil {
		return "", fmt.Errorf("checking repo for changes: %w", err)
	}

	if len(changedFiles) > 0 {
		explanation := fmt.Sprintf("Fetching because %s has changes...", strings.Join(changedFiles, ", "))
		output, fetchErr := r.runGit(repoPath, "fetch", "--all")
		return explanation + "\n" + output, fetchErr
	}

	output, err := r.runGit(repoPath, "pull")
	if err != nil {
		return output, fmt.Errorf("pulling: %w", err)
	}
	return output, nil
}

func (r *RefreshAction) changedFiles(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)
	for _, line := range strings.Split(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0:1] == "??" {
			continue
		}
		files = append(files, line[3:])
	}
	return files, nil
}

func (r *RefreshAction) runGit(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}
