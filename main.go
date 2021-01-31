package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	docopt "github.com/docopt/docopt-go"

	"github.com/adam000/goutils/git"
	"github.com/adam000/goutils/shell"
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
	foreach-git-dir [(--verbose|-v)|(--brief|-b)] [(--stashes|-s)] <root-dir>

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

	rootDir, err := filepath.Abs(options.RootDir)
	if err != nil {
		log.Fatalf("Could not make root-dir absolute: %v", err)
	}
	output := log.New(os.Stdout, "", 0)
	sem := make(chan struct{}, 16)
	processDirectory(output, options, sem, rootDir)
}

// processDirectory recursively searches a directory for Git repositories and
// outputs their status. The given semaphore is used to limit concurrent work.
func processDirectory(output *log.Logger, options CommandLineOptions, sem chan struct{}, dir string) {
	sem <- struct{}{} // acquire semaphore
	isRoot, subdirs, err := shell.ParseDirectory(git.IsGitRoot, dir)
	if err != nil {
		log.Printf("ERROR: %v", err)
		<-sem // release semaphore
		return
	}
	if isRoot {
		repo, err := getRepositoryInfo(options, dir)
		if err != nil {
			log.Printf("ERROR: could not get info for repository %q: %v", dir, err)
		}
		printRepositoryInfo(output, options, repo)
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
			processDirectory(output, options, sem, subdir)
		}()
	}
	wg.Wait()
}

func getRepositoryInfo(options CommandLineOptions, pwd string) (*repository, error) {
	isClean := true
	stashes := ""
	status := ""

	if options.Stashes {
		// Run `git stash list` and store results in stashes
		cmd := exec.Command("git", "stash", "list")
		cmd.Dir = pwd
		output, err := cmd.Output()
		if err != nil {
			return nil, err
		}
		stashes = strings.TrimSpace(string(output))
		isClean = isClean && len(stashes) == 0
	}

	// Run `git diff --quiet` to see if the working directory is dirty
	workingDirectoryIsClean := true
	{
		cmd := exec.Command("git", "status", "--porcelain")
		cmd.Dir = pwd
		out, _ := cmd.Output()
		workingDirectoryIsClean = len(out) == 0
	}

	if !workingDirectoryIsClean {
		isClean = false

		if !options.Brief {
			// Run `git status -sb` and store results in `status`
			cmd := exec.Command("git", "status", "-sb")
			cmd.Dir = pwd
			output, err := cmd.Output()
			if err != nil {
				return nil, err
			}
			status = strings.TrimSpace(string(output))
		}
	}

	return &repository{pwd, isClean, stashes, status}, nil
}

func printRepositoryInfo(output *log.Logger, options CommandLineOptions, repo *repository) {
	if repo.isClean && !options.Verbose {
		return
	}
	// Collect message in buffer and send output as single log event.
	buf := new(strings.Builder)
	defer output.Print(buf)

	fmt.Fprintf(buf, "Repository root: %s\n", repo.root)
	if options.Brief {
		return
	}
	if len(repo.status) != 0 {
		fmt.Fprintln(buf, "\tStatus:")
		for _, line := range strings.Split(repo.status, "\n") {
			fmt.Fprintf(buf, "\t\t%s\n", line)
		}
	}

	if options.Stashes && len(repo.stashes) != 0 {
		fmt.Fprintln(buf, "\tStashes:")
		for _, line := range strings.Split(repo.stashes, "\n") {
			fmt.Fprintf(buf, "\t\t%s\n", line)
		}
	}
}
