package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/adam000/foreach-git-dir/action"
	"github.com/adam000/foreach-git-dir/config"
	"github.com/adam000/foreach-git-dir/parsing"
	"github.com/adam000/goutils/git"
	"github.com/adam000/goutils/shell"
)

type result struct {
	dir    string
	output string
	err    error
}

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
	foreach-git-dir [<root-dir>] [--verbose|-v] [<predicate>...] [-- <action>...]

<root-dir>
	The directory to scan. If not provided, relies on a config file at
	$XDG_CONFIG_HOME/foreach-git-dir/config.yaml (defaulting that variable to ~/.config)

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
	statusCh := make(chan statusEvent)

	baseDirectoriesWithoutRepos := make([]string, 0)
	// Find all the directories immediately under rootDir
	// that are not in excludes.
	{
		entries, err := os.ReadDir(directives.RootDir)
		if err != nil {
			slog.Error("Error reading root directory", "rootDir", directives.RootDir, "error", err)
		} else {
			for _, entry := range entries {
				if entry.IsDir() {
					dirPath := filepath.Join(directives.RootDir, entry.Name())
					if !slices.Contains(directives.Excludes, dirPath) {
						baseDirectoriesWithoutRepos = append(baseDirectoriesWithoutRepos, dirPath)
					}
				}
			}
		}
	}

	errorResults := make([]result, 0)
	validResults := make([]result, 0)
	doneResults := make(chan struct{})
	deletedDirectories := 0

	go func() {
		for r := range results {
			// If r.dir is the dir or a subdir of any of the base directories,
			// delete it from baseDirectoriesWithoutRepos. Then it will only
			// contain directories that don't have a git repo in them.
			for i := 0; i < len(baseDirectoriesWithoutRepos)-deletedDirectories; i++ {
				baseDir := baseDirectoriesWithoutRepos[i]
				if r.dir == baseDir || strings.HasPrefix(r.dir, baseDir+"/") {
					// Remove baseDir from baseDirectoriesWithoutRepos by swapping it with
					// the last element. We will truncate the slice at the end of the loop.
					end := len(baseDirectoriesWithoutRepos) - 1 - deletedDirectories
					baseDirectoriesWithoutRepos[i], baseDirectoriesWithoutRepos[end] = baseDirectoriesWithoutRepos[end], baseDirectoriesWithoutRepos[i]
					deletedDirectories++
					break
				}
			}

			if r.err != nil {
				errorResults = append(errorResults, r)
			} else if r.output != "" {
				validResults = append(validResults, r)
			}
		}
		close(doneResults)
	}()

	go func() {
		processDirectory(sem, directives.RootDir, directives, results, statusCh)
		close(results)
		close(statusCh)
	}()

	if err := startStatusUI(statusCh); err != nil {
		slog.Error("Bubbletea UI failed", "error", err)
	}

	<-doneResults

	// Warn the user if any directories in the excludes list do not exist.
	for _, ex := range directives.Excludes {
		if _, err := os.Stat(ex); err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("Warning: excluded directory '%s' does not exist\n", ex)
			}
		}
	}

	baseDirectoriesWithoutRepos = baseDirectoriesWithoutRepos[:len(baseDirectoriesWithoutRepos)-deletedDirectories]
	if len(baseDirectoriesWithoutRepos) > 0 {
		fmt.Println("Directories without git repositories:")
		for _, dir := range baseDirectoriesWithoutRepos {
			fmt.Printf("  %s\n", dir)
		}
		fmt.Println()
		fmt.Println("Note: the above directories do not contain any git repos.")
		fmt.Println("You can ignore these repos with -exclude <dirname> or in the config file.")
		fmt.Println()
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

// processDirectory recursively searches a directory for Git repositories and
// outputs their status. The given semaphore is used to limit concurrent work.
func processDirectory(sem chan struct{}, dir string, directives parsing.Directives, results chan result, statusCh chan statusEvent) {
	statusCh <- statusEvent{dir: filepath.ToSlash(dir), state: statusStarted}
	sem <- struct{}{} // acquire semaphore

	normalizedDir := filepath.ToSlash(dir)

	// Check for svn / hg / p4 metadata and skip if found.
	otherRepoExists := false
	if _, err := os.Stat(filepath.Join(normalizedDir, ".svn")); err == nil {
		otherRepoExists = true
	} else if _, err := os.Stat(filepath.Join(normalizedDir, ".hg")); err == nil {
		otherRepoExists = true
	} else if _, err := os.Stat(filepath.Join(normalizedDir, "p4config.txt")); err == nil {
		otherRepoExists = true
	}
	if otherRepoExists {
		statusCh <- statusEvent{dir: normalizedDir, state: statusComplete, err: nil}
		results <- result{
			dir: normalizedDir,
		}
		<-sem // release semaphore
		return
	}

	isRoot, subdirs, err := shell.ParseDirectory(git.IsGitRoot, normalizedDir)
	if err != nil {
		results <- result{
			dir:    normalizedDir,
			output: "",
			err:    fmt.Errorf("processing directory %s: %w", normalizedDir, err),
		}
		statusCh <- statusEvent{dir: normalizedDir, state: statusError, err: err}
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
				statusCh <- statusEvent{dir: normalizedDir, state: statusError, err: err}
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
						fmt.Fprintf(&output, "Error while running %s: %s\n", action.Name(), err)
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
		statusCh <- statusEvent{dir: normalizedDir, state: statusSuccess}
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
			if subdir == ex {
				continue SubdirsLoop
			}
		}

		wg.Add(1)
		go func(subdir string) {
			defer wg.Done()
			processDirectory(sem, subdir, directives, results, statusCh)
		}(subdir)
	}
	wg.Wait()
}
