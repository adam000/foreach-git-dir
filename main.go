package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	go jobStack.Run(recall)

	numThreads := 16
	for i := 0; i < numThreads; i++ {
		go func() {
			for {
				select {
				case job := <-jobStack.outbox:
					parseDirectory(options, job, jobStack.inbox, repos, errors, &waitGroup)
				case <-recall:
					break
				}
			}
		}()
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
	for i := 0; i < numThreads+1; i++ {
		recall <- struct{}{}
	}

	printWait.Wait()
	time.Sleep(2 * time.Second)
}

func parseDirectory(options CommandLineOptions, dir string, outJobs chan<- string, repos chan<- repository, errors chan<- error, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	if stat, err := os.Stat(dir); err != nil {
		errors <- fmt.Errorf("Could not read root directory: %w", err)
		return
	} else if !stat.IsDir() {
		errors <- fmt.Errorf("'%s' is not a directory", dir)
		return
	}

	// Try `git rev-parse --show-toplevel` at our current directory
	result, isGitRoot, err := RunInDir(dir, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		errors <- fmt.Errorf("Failed to run git rev-parse: %w", err)
		return
	} else if !isGitRoot {
		// Recurse within
		files, dirErr := ioutil.ReadDir(dir)
		if dirErr != nil {
			errors <- fmt.Errorf("Reading directory '%s': %w", dir, dirErr)
		}

		for _, fileInfo := range files {
			if fileInfo.IsDir() {
				nextDir := filepath.Join(dir, fileInfo.Name())
				waitGroup.Add(1)
				outJobs <- nextDir
			}
		}
		return
	}

	// Make sure we're in the base -- if we ever hit this error, we have... big problems
	// Not 100% sure this is an exactly accurate way to find this out, might need os.SameFile
	if strings.TrimSpace(result) != dir {
		errors <- fmt.Errorf("Job directory '%s' is not the base git directory '%s'", dir, result)
		return
	}

	// We're in the root of a repo
	repo, err := getRepositoryInfo(options, dir)
	if err != nil {
		errors <- fmt.Errorf("Could not get info for repository '%s': %w", dir, err)
		return
	}

	repos <- repo
}

func RunInDir(directory string, command ...string) (result string, isGitRoot bool, err error) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = directory

	// Create Pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", false, fmt.Errorf("Failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", false, fmt.Errorf("Failed to create stderr pipe: %w", err)
	}

	// Start Command
	if err = cmd.Start(); err != nil {
		return "", false, fmt.Errorf("Failed to start command: %w", err)
	}

	// Read From Pipes
	stdoutBytes, stdoutErr := ioutil.ReadAll(stdout)
	if stdoutErr != nil {
		return "", false, fmt.Errorf("Failed reading stdout (%w) (perhaps another error: %v)", stdoutErr, err)
	}
	outputStr := string(stdoutBytes)

	stderrBytes, stderrErr := ioutil.ReadAll(stderr)
	if stderrErr != nil {
		return outputStr, false, fmt.Errorf("Failed reading stderr (%w) while trying to deal with command failure: %v", stderrErr, err)
	}

	// Finish Command
	if err = cmd.Wait(); err != nil {
		if strings.Contains(string(stderrBytes), "not a git repository") {
			return outputStr, false, nil
		} else {
			return outputStr, false, err
		}
	}
	return outputStr, true, nil
}

func printRepositoryInfo(options CommandLineOptions, repos <-chan repository, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	for repo := range repos {
		//fmt.Printf("Repo: %#v\n", repo)
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

func printErrorInfo(errors <-chan error, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()
	for err := range errors {
		fmt.Printf("ERROR: %v\n", err)
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
			panic(err)
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
			output, _ := cmd.Output()
			status = strings.TrimSpace(string(output))
		}
	}

	return repository{pwd, isClean, stashes, status}, nil
}
