package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

type DirMatch func(dir string) (bool, error)

func ParseDirectory(matches DirMatch, dir string) (isRoot bool, subdirs []string, stopErr error) {
	if stat, err := os.Stat(dir); err != nil {
		stopErr = fmt.Errorf("Could not read root directory: %w", err)
		return
	} else if !stat.IsDir() {
		stopErr = fmt.Errorf("'%s' is not a directory", dir)
		return
	}

	isMatch, err := matches(dir)
	if err != nil {
		stopErr = fmt.Errorf("Running command in directory '%s': %w", dir, err)
		return
	} else if !isMatch {
		// Recurse within
		files, dirErr := ioutil.ReadDir(dir)
		if dirErr != nil {
			stopErr = fmt.Errorf("Reading directory '%s': %w", dir, dirErr)
			return
		}

		subdirs := make([]string, 0)
		for _, fileInfo := range files {
			if fileInfo.IsDir() {
				nextDir := filepath.Join(dir, fileInfo.Name())
				subdirs = append(subdirs, nextDir)
			}
		}
		return false, subdirs, nil
	}

	return true, nil, nil
}

func RunInDir(directory string, command ...string) (result string, stderrText string, err error) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = directory

	// Create Pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", fmt.Errorf("Failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", "", fmt.Errorf("Failed to create stderr pipe: %w", err)
	}

	// Start Command
	if err = cmd.Start(); err != nil {
		return "", "", fmt.Errorf("Failed to start command: %w", err)
	}

	// Read From Pipes
	stdoutBytes, stdoutErr := ioutil.ReadAll(stdout)
	if stdoutErr != nil {
		return "", "", fmt.Errorf("Failed reading stdout (%w) (perhaps another error: %v)", stdoutErr, err)
	}
	outputStr := string(stdoutBytes)

	stderrBytes, stderrErr := ioutil.ReadAll(stderr)
	if stderrErr != nil {
		return outputStr, "", fmt.Errorf("Failed reading stderr (%w) while trying to deal with command failure: %v", stderrErr, err)
	}

	// Finish Command
	err = cmd.Wait()
	return outputStr, string(stderrBytes), err
}
