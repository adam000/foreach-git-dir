package parsing

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/adam000/foreach-git-dir/predicate"
)

func ParseCommandLine(args []string) (string, bool, predicate.Predicate, []string, error) {
	if len(args) == 0 {
		return "", false, predicate.Id, []string{}, fmt.Errorf("No arguments provided")
	}

	argIndex := 0

	// First argument is <root-dir>
	rootDir, err := filepath.Abs(args[argIndex])
	if err != nil {
		return rootDir, false, predicate.Id, []string{}, fmt.Errorf("Error finding root dir: %w", err)
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
		return rootDir, verbose, predicates, []string{}, fmt.Errorf("Error parsing predicates: %w", err)
	}

	// Look for all actions (args after --)
	actions, err := parseActions(args, argIndex)

	return rootDir, verbose, predicates, actions, nil
}
