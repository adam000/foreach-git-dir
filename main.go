package main

import (
	"fmt"
	"os"
	"sync"

	docopt "github.com/docopt/docopt-go"
)

type job struct {
	Directory string
}

type repository struct {
	root    string
	isClean bool
}

func main() {
	usage := `
Description: find all the git repositories under the current directory and report if they have any
outstanding changes (meaning the working directory is not clean).

Usage:
	check-git-dirs <root-dir>
`

	arguments, err := docopt.Parse(usage, nil, true, "check-git-dirs", false)
	if err != nil {
		panic(fmt.Errorf("Could not parse arguments: %v", err))
	}

	var waitGroup sync.WaitGroup
	jobs := make(chan job, 8)
	repos := make(chan repository, 8)

	rootJob := job{arguments["<root-dir>"].(string)}
	jobs <- rootJob

	for i := 0; i < 4; i++ {
		waitGroup.Add(1)
		go parseDirectoriesUntilDone(jobs, repos, &waitGroup)
	}

	waitGroup.Wait()
}

func parseDirectoriesUntilDone(jobs chan job, repos chan<- repository, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	job := <-jobs

	if stat, err := os.Stat(job.Directory); err != nil {
		panic(fmt.Errorf("Could not read root directory: %v", err))
	} else if !stat.IsDir() {
		panic(fmt.Errorf("'%v' is not a directory", job.Directory))
	}

	// Try `git rev-parse --show-toplevel`
	// if it's the current directory, we have a repository. Send it, along with any info we got about it.
}
