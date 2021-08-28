package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/adam000/foreach-git-dir/action"
	"github.com/adam000/foreach-git-dir/parsing"
	"github.com/adam000/foreach-git-dir/predicate"
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
	foreach-git-dir <root-dir> [--verbose|-v] [<predicate>...] -- <action>...

Predicates:
	-isDirty		The repository is not in a clean state
	-hasStash		The repository has any stashes

Predicates can be joined with parentheses, -not, -or, and -and.
`

	logger := log.New(os.Stdout, "", 0)

	rootDir, verbose, predicates, actions, err := parsing.ParseCommandLine(os.Args[1:])
	if err != nil {
		logger.Printf("Failure parsing command line: %v", err)
		logger.Fatalf(usage)
	}
	sem := make(chan struct{}, 16)

	processDirectory(logger, sem, verbose, rootDir, predicates, actions)
}

// processDirectory recursively searches a directory for Git repositories and
// outputs their status. The given semaphore is used to limit concurrent work.
func processDirectory(logger *log.Logger, sem chan struct{}, verbose bool, dir string, tester predicate.Predicate, runner action.Action) {
	sem <- struct{}{} // acquire semaphore
	isRoot, subdirs, err := shell.ParseDirectory(git.IsGitRoot, dir)
	if err != nil {
		logger.Printf("ERROR: %v", err)
		<-sem // release semaphore
		return
	}
	if isRoot {
		shouldRun, err := tester(dir)
		if err != nil {
			logger.Printf("ERROR: could not test repository %s: %v", dir, err)
			return
		}

		var output strings.Builder

		if shouldRun || (!shouldRun && verbose) {
			fmt.Fprintf(&output, "\nRepository root: %s\n", dir)
		}

		if shouldRun {
			runner(dir, &output)
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
	wg.Add(len(subdirs))
	for _, subdir := range subdirs {
		subdir := subdir // capture loop variable for closure
		go func() {
			defer wg.Done()
			processDirectory(logger, sem, verbose, subdir, tester, runner)
		}()
	}
	wg.Wait()
}
