package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/adam000/foreach-git-dir/action"
	"github.com/adam000/foreach-git-dir/config"
	"github.com/adam000/foreach-git-dir/parsing"
	"github.com/adam000/goutils/git"
	"github.com/adam000/goutils/shell"
)

func newLogger() *slog.Logger {
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			stateHome = filepath.Join(homeDir, ".local", "state")
		}
	}

	if stateHome != "" {
		stateDir := filepath.Join(stateHome, "foreach-git-dir")
		if err := os.MkdirAll(stateDir, 0o755); err == nil {
			logPath := filepath.Join(stateDir, "foreach-git-dir.log")
			if logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
				return slog.New(slog.NewTextHandler(logFile, nil))
			}
		}
	}

	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func main() {
	usage := `
Description: find all the git repositories under the <root-dir> and run some
<predicates> on them, taking some <actions> if the repository matches. For
<predicates>, this command takes inspiration from find(1) and allows boolean -or
and -and combination of predicates.

Usage:
	foreach-git-dir <root-dir> [--verbose|-v] [<predicate>...] [-- <action>...]

Predicates:
%s
Predicates can be joined with parentheses, -not, -or, and -and.

Actions:
%s
Every action is executed on every repository that matches the predicate(s).

If no actions are given, prints every repository that matches <predicates>, or all
repositories if no predicates are found.

------

`

	logger := newLogger()
	slog.SetDefault(logger)

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
	}

	directives, err := parsing.ParseCommandLine(os.Args[1:], cfg)
	if err != nil {
		var predicates strings.Builder
		for _, pred := range parsing.PredicateInfo() {
			predicates.WriteString(fmt.Sprintf("%20s  %-58s\n", pred.Name, pred.Description))
		}
		var actions strings.Builder
		for _, action := range action.DefaultShellActions() {
			actions.WriteString(fmt.Sprintf("%20s  %-58s\n", action.Name(), action.Summary()))
		}
		fmt.Printf(usage, predicates.String(), actions.String())
		fmt.Printf("Failure parsing command line: %v\n", err)
		os.Exit(1)
	}
	sem := make(chan struct{}, 16)

	results := make(chan result)
	go func() {
		processDirectory(sem, directives.RootDir, directives, results)
		close(results)
	}()

	errorResults := make([]result, 0)
	validResults := make([]result, 0)
	for r := range results {
		if r.err != nil {
			errorResults = append(errorResults, r)
		} else if r.output != "" {
			validResults = append(validResults, r)
		}
	}

	for _, r := range errorResults {
		fmt.Printf("Error processing repository %s: %v\n", r.dir, r.err)
	}

	for _, r := range validResults {
		fmt.Print(r.output)
	}

	fmt.Println()
	fmt.Printf("%d repositories with %d errors\n", len(validResults), len(errorResults))
}

type result struct {
	dir    string
	output string
	err    error
}

// processDirectory recursively searches a directory for Git repositories and
// outputs their status. The given semaphore is used to limit concurrent work.
func processDirectory(sem chan struct{}, dir string, directives parsing.Directives, results chan result) {
	sem <- struct{}{} // acquire semaphore

	normalizedDir := filepath.ToSlash(dir)
	isRoot, subdirs, err := shell.ParseDirectory(git.IsGitRoot, normalizedDir)
	if err != nil {
		results <- result{
			dir:    normalizedDir,
			output: "",
			err:    fmt.Errorf("processing directory %s: %w", normalizedDir, err),
		}
		<-sem // release semaphore
		return
	}
	if isRoot {
		shouldRun := true
		if directives.Predicates != nil {
			var err error
			shouldRun, err = directives.Predicates(dir)
			if err != nil {
				results <- result{
					dir:    dir,
					output: "",
					err:    fmt.Errorf("testing predicates on repository %s: %w", dir, err),
				}
				<-sem // release semaphore
				return
			}
		}

		var output strings.Builder

		if len(directives.Actions) == 0 {
			if shouldRun {
				fmt.Fprintln(&output, dir)
			}
		} else {
			if shouldRun || (!shouldRun && directives.Verbose) {
				fmt.Fprintf(&output, "\nRepository root: %s\n", dir)
			}

			if shouldRun {
				for _, action := range directives.Actions {
					stdout, err := action.Run(dir)

					if err != nil {
						fmt.Fprintf(&output, "Error while running %s: %s\n", action, err)
					}
					fmt.Fprintf(&output, "%s\n", stdout)
				}
			}
		}

		results <- result{
			dir:    dir,
			output: output.String(),
			err:    nil,
		}
		<-sem // release semaphore
		return
	}

	// Descend into subdirectories.
	// Release the semaphore to permit work to continue.
	<-sem
	var wg sync.WaitGroup
SubdirsLoop:
	for _, subdir := range subdirs {
		for _, ex := range directives.Excludes {
			relPath, _ := filepath.Rel(directives.RootDir, subdir)
			if relPath == ex {
				continue SubdirsLoop
			}
		}

		wg.Add(1)
		go func(subdir string) {
			defer wg.Done()
			processDirectory(sem, subdir, directives, results)
		}(subdir)
	}
	wg.Wait()
}
