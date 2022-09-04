package parsing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adam000/foreach-git-dir/predicate"
)

type Directives struct {
	RootDir    string
	Verbose    bool
	Predicates predicate.Predicate
	Actions    []string
	//ListOnly   bool
}

func ParseCommandLine(args []string) (Directives, error) {
	directives := Directives{}
	if len(args) == 0 {
		return Directives{}, fmt.Errorf("no arguments provided")
	}

	argIndex := 0

	// First argument is <root-dir>
	{
		rootDir, err := filepath.Abs(args[argIndex])
		if err != nil {
			return Directives{}, fmt.Errorf("error finding root dir: %w", err)
		}
		if fileInfo, err := os.Stat(rootDir); err != nil {
			if os.IsNotExist(err) {
				return Directives{}, fmt.Errorf("error finding root dir - does not exist (%s): %w", rootDir, err)
			}
			return Directives{}, fmt.Errorf("error accessing root dir (%s): %w", rootDir, err)
		} else if !fileInfo.IsDir() {
			return Directives{}, fmt.Errorf("root dir '%s' is not a directory", rootDir)
		}
		directives.RootDir = rootDir
		argIndex++
	}

	if argIndex == len(args) {
		//directives.ListOnly = true
		return directives, nil
	}

	// Next argument may be --verbose or -v
	{
		verboseArg := strings.ToLower(args[argIndex])
		directives.Verbose = verboseArg == "--verbose" || verboseArg == "-v"
		if directives.Verbose {
			argIndex++
		}
	}

	// Look for all predicates (args before --)
	{
		predicates, newArgIndex, err := parsePredicates(args, argIndex)
		if err != nil {
			return Directives{}, fmt.Errorf("error parsing predicates: %w", err)
		}
		argIndex = newArgIndex
		directives.Predicates = predicates
	}

	// Look for all actions (args after --)
	{
		actions, err := parseActions(args, argIndex)
		if err != nil {
			return Directives{}, fmt.Errorf("error parsing actions: %w", err)
		}
		directives.Actions = actions
	}

	return directives, nil
}
