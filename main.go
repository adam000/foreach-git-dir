package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	docopt "github.com/docopt/docopt-go"
)

type repository struct {
	root    string
	isClean bool
	stashes string
	status  string
}

type CommandLineOptions struct {
	RootDir string `docopt:"<root-dir>"`
	Verbose bool
	Brief   bool
	Stashes bool
}

func main() {
	usage := `
Description: find all the git repositories under the current directory and report if they have any
outstanding changes (meaning the working directory is not clean). Can also show stashes with -s

Usage:
	check-git-dirs [(--verbose|-v)|(--brief|-b)] [(--stashes|-s)] <root-dir>

Options:
	--verbose -v  List all git repositories found, even if they don't contain changes
	--brief -b    Only list repositories with changes; do not include stashes or dirty working directories
	--stashes -s  stashes: List all repositories with stashes and 'git stash list' output
`

	arguments, err := docopt.ParseDoc(usage)
	if err != nil {
		panic(fmt.Errorf("Could not parse arguments: %v", err))
	}
	var options CommandLineOptions
	arguments.Bind(&options)

	var waitGroup sync.WaitGroup
	repos := make(chan repository)
	errors := make(chan error)
	recall := make(chan struct{})

	jobStack := JobStack{
		inbox:  make(chan string),
		outbox: make(chan string),
	}
	defer jobStack.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go jobStack.Run(ctx)

	numGoroutines := 16
	for i := 0; i < numGoroutines; i++ {
		go processDirectories(&jobStack, options, errors, repos, recall, &waitGroup)
	}

	if rootDir, err := filepath.Abs(options.RootDir); err != nil {
		log.Fatalf("Could not make root-dir absolute: %v", err)
	} else {
		waitGroup.Add(1)
		jobStack.inbox <- rootDir
	}

	printWait := sync.WaitGroup{}

	printWait.Add(1)
	go printRepositoryInfo(options, repos, &printWait)
	printWait.Add(1)
	go printErrorInfo(errors, &printWait)

	waitGroup.Wait()
	close(repos)
	close(errors)
	for i := 0; i < numGoroutines; i++ {
		recall <- struct{}{}
	}

	printWait.Wait()
}

func processDirectories(jobStack *JobStack, options CommandLineOptions, errors chan<- error, repos chan<- repository, recall <-chan struct{}, waitGroup *sync.WaitGroup) {
	for {
		select {
		case dir := <-jobStack.outbox:
			if isRoot, subdirs, err := ParseDirectory(IsGitRoot, dir); err != nil {
				errors <- err
			} else {
				if isRoot {
					repo, err := getRepositoryInfo(options, dir)
					if err != nil {
						errors <- fmt.Errorf("Could not get info for repository '%s': %w", dir, err)
					} else {
						repos <- repo
					}
				} else {
					for _, subdir := range subdirs {
						waitGroup.Add(1)
						jobStack.inbox <- subdir
					}
				}
			}
			waitGroup.Done()
		case <-recall:
			break
		}
	}
}

func getRepositoryInfo(options CommandLineOptions, pwd string) (repo repository, err error) {
	isClean := true
	stashes := ""
	status := ""

	if options.Stashes {
		// Run `git stash list` and store results in stashes
		cmd := exec.Command("git", "stash", "list")
		cmd.Dir = pwd
		output, err := cmd.Output()
		if err != nil {
			return repository{}, err
		}
		stashes = strings.TrimSpace(string(output))
		isClean = isClean && len(stashes) == 0
	}

	// Run `git diff --quiet` to see if the working directory is dirty
	workingDirectoryIsClean := true
	{
		cmd := exec.Command("git", "diff", "--quiet")
		cmd.Dir = pwd
		_, err := cmd.Output()
		// Assume that this means there's a diff
		workingDirectoryIsClean = err == nil
	}

	if !workingDirectoryIsClean {
		isClean = false

		if !options.Brief {
			// Run `git status -sb` and store results in `status`
			cmd := exec.Command("git", "status", "-sb")
			cmd.Dir = pwd
			output, err := cmd.Output()
			if err != nil {
				return repository{}, err
			}
			status = strings.TrimSpace(string(output))
		}
	}

	return repository{pwd, isClean, stashes, status}, nil
}

func printRepositoryInfo(options CommandLineOptions, repos <-chan repository, printWait *sync.WaitGroup) {
	defer printWait.Done()
	for repo := range repos {
		if repo.isClean && !options.Verbose {
			continue
		}
		fmt.Printf("Repository root: %s\n", repo.root)
		if !options.Brief {
			if len(repo.status) != 0 {
				fmt.Println("\tStatus:")
				for _, line := range strings.Split(repo.status, "\n") {
					fmt.Printf("\t\t%s\n", line)
				}
			}

			if options.Stashes && len(repo.stashes) != 0 {
				fmt.Println("\tStashes:")
				for _, line := range strings.Split(repo.stashes, "\n") {
					fmt.Printf("\t\t%s\n", line)
				}
			}
		}
	}
}

func printErrorInfo(errors <-chan error, printWait *sync.WaitGroup) {
	defer printWait.Done()
	for err := range errors {
		fmt.Printf("ERROR: %v\n", err)
	}
}
