package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/adam000/foreach-git-dir/action"
	"github.com/adam000/foreach-git-dir/parsing"
	"github.com/adam000/goutils/git"
	"github.com/adam000/goutils/shell"
)

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

	logger := log.New(os.Stdout, "", 0)

	// Load config defaults from XDG_CONFIG_HOME if it exists
	cfgRoot := ""
	cfgExcludes := make([]string, 0)

	homeDir, _ := os.UserHomeDir()
	xdgConfigHome := filepath.Join(homeDir, ".config")
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		xdgConfigHome = x
	}
	cfgPath := filepath.Join(xdgConfigHome, "foreach-git-dir", "config.json")
	if cfgPath != "" {
		if _, err := os.Stat(cfgPath); err == nil {
			if b, err := os.ReadFile(cfgPath); err == nil {
				var cfg struct {
					RootDir  string   `json:"rootDir"`
					Excludes []string `json:"excludes"`
				}
				if err := json.Unmarshal(b, &cfg); err == nil {
					cfgRoot = cfg.RootDir
					if cfg.Excludes != nil {
						cfgExcludes = cfg.Excludes
					}
				}
			}
		}
	}

	directives, err := parsing.ParseCommandLine(os.Args[1:], cfgRoot, cfgExcludes)
	if err != nil {
		var predicates strings.Builder
		for _, pred := range parsing.PredicateInfo() {
			predicates.WriteString(fmt.Sprintf("%20s  %-58s\n", pred.Name, pred.Description))
		}
		var actions strings.Builder
		for _, action := range action.DefaultShellActions() {
			actions.WriteString(fmt.Sprintf("%20s  %-58s\n", action.Name(), action.Summary()))
		}
		logger.Printf(usage, predicates.String(), actions.String())
		logger.Fatalf("Failure parsing command line: %v", err)
	}
	sem := make(chan struct{}, 16)

	processDirectory(logger, sem, directives.RootDir, directives)
}

// processDirectory recursively searches a directory for Git repositories and
// outputs their status. The given semaphore is used to limit concurrent work.
func processDirectory(logger *log.Logger, sem chan struct{}, dir string, directives parsing.Directives) {
	sem <- struct{}{} // acquire semaphore
	normalizedDir := filepath.ToSlash(dir)
	isRoot, subdirs, err := shell.ParseDirectory(git.IsGitRoot, normalizedDir)
	if err != nil {
		logger.Printf("ERROR: %v", err)
		<-sem // release semaphore
		return
	}
	if isRoot {
		shouldRun := true
		if directives.Predicates != nil {
			var err error
			shouldRun, err = directives.Predicates(dir)
			if err != nil {
				logger.Printf("ERROR: could not test repository %s: %v", dir, err)
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

		if output.Len() != 0 {
			logger.Print(&output)
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
			processDirectory(logger, sem, subdir, directives)
		}(subdir)
	}
	wg.Wait()
}
