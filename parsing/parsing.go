package parsing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adam000/foreach-git-dir/predicate"
)

func ParseCommandLine(args []string) (string, bool, predicate.Predicate, []string, error) {
	if len(args) == 0 {
		return "", false, predicate.Id, []string{}, fmt.Errorf("no arguments provided")
	}

	argIndex := 0

	// First argument is <root-dir>
	rootDir, err := filepath.Abs(args[argIndex])
	if err != nil {
		return rootDir, false, predicate.Id, []string{}, fmt.Errorf("error finding root dir: %w", err)
	}
	if fileInfo, err := os.Stat(rootDir); err != nil {
		if os.IsNotExist(err) {
			return rootDir, false, predicate.Id, []string{}, fmt.Errorf("error finding root dir - does not exist (%s): %w", rootDir, err)
		}
		return rootDir, false, predicate.Id, []string{}, fmt.Errorf("error accessing root dir (%s): %w", rootDir, err)
	} else if !fileInfo.IsDir() {
		return rootDir, false, predicate.Id, []string{}, fmt.Errorf("root dir '%s' is not a directory", rootDir)
	}
	argIndex++

	// Next argument may be --verbose or -v
	verboseArg := strings.ToLower(args[argIndex])
	verbose := verboseArg == "--verbose" || verboseArg == "-v"
	if verbose {
		argIndex++
	}

	// Look for all predicates (args before --)
	predicates, argIndex, err := parsePredicates(args, argIndex)
	if err != nil {
		return rootDir, verbose, predicates, []string{}, fmt.Errorf("error parsing predicates: %w", err)
	}

	// Look for all actions (args after --)
	actions, err := parseActions(args, argIndex)
	if err != nil {
		return rootDir, verbose, predicates, []string{}, fmt.Errorf("error parsing actions: %w", err)
	}

	return rootDir, verbose, predicates, actions, nil
}
