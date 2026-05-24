package parsing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adam000/foreach-git-dir/action"
	"github.com/adam000/foreach-git-dir/config"
	"github.com/adam000/foreach-git-dir/predicate"
	"github.com/adam000/foreach-git-dir/shell"
)

type Directives struct {
	RootDir    string
	Excludes   []string
	Verbose    bool
	Predicates predicate.Predicate
	Actions    []action.Action
}

// ParseCommandLine parses the command line taking optional defaults for root and excludes.
// default parameters will only be used if the command line doesn't provide a value for them.
func ParseCommandLine(args []string, cfg config.Config) (Directives, error) {
	directives := Directives{
		RootDir:  cfg.RootDir,
		Excludes: cfg.Excludes,
	}

	if len(args) == 0 {
		if directives.RootDir == "" {
			return Directives{}, fmt.Errorf("no arguments provided and no default root dir provided")
		}
		return directives, nil
	}

	argIndex := 0

	// First argument is <root-dir> if it doesn't start with a `-`
	rootDir := directives.RootDir
	if strings.HasPrefix(args[argIndex], "-") {
		if directives.RootDir == "" {
			return Directives{}, fmt.Errorf("expected first argument to be root dir, but got '%s' and no default root dir provided", args[argIndex])
		}
	} else {
		rootDir = args[argIndex]
		argIndex++
	}
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return Directives{}, fmt.Errorf("finding root dir: %w", err)
	}
	if fileInfo, err := os.Stat(rootDir); err != nil {
		if os.IsNotExist(err) {
			return Directives{}, fmt.Errorf("finding root dir - does not exist (%s): %w", rootDir, err)
		}
		return Directives{}, fmt.Errorf("accessing root dir (%s): %w", rootDir, err)
	} else if !fileInfo.IsDir() {
		return Directives{}, fmt.Errorf("root dir '%s' is not a directory", rootDir)
	}
	directives.RootDir = rootDir

	if argIndex == len(args) {
		// root dir was the only arg provided, so we're done
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

	sh := cfg.Shell
	if len(sh) == 0 {
		sh = shell.GetDefault()
	}
	// Look for all predicates (args before --)
	{
		predicates, excludes, newArgIndex, err := parsePredicates(args, argIndex, sh)
		if err != nil {
			return Directives{}, fmt.Errorf("error parsing predicates: %w", err)
		}
		argIndex = newArgIndex
		directives.Predicates = predicates
		if len(excludes) != 0 {
			directives.Excludes = excludes
		}
	}

	// Look for all actions (args after --)
	{
		actions, err := parseActions(args, argIndex, sh)
		if err != nil {
			return Directives{}, fmt.Errorf("error parsing actions: %w", err)
		}
		directives.Actions = actions
	}

	return directives, nil
}
