package parsing

import (
	"fmt"
	"strings"

	"github.com/adam000/foreach-git-dir/action"
)

func tokenizeActions(args []string, argIndex int, shell []string) ([]action.Action, int, error) {
	numArgs := len(args)
	actions := make([]action.Action, 0, numArgs-argIndex)

	actionOptions := action.DefaultShellActions()
	for numArgs != argIndex {
		thisArg := strings.Trim(strings.ToLower(args[argIndex]), " \t")
		if entry, ok := actionOptions[thisArg]; ok {
			actions = append(actions, entry)
		} else if thisArg == "-custom" {
			argIndex++
			if numArgs == argIndex {
				return actions, argIndex, fmt.Errorf("expected command after -custom but no more arguments found")
			}
			customCmd := args[argIndex]
			actions = append(actions, action.NewCustomAction(customCmd, shell))
		} else {
			return actions, argIndex, fmt.Errorf("unknown action flag '%s'", args[argIndex])
		}
		argIndex++
	}

	return actions, argIndex, nil
}

func parseActions(args []string, argIndex int, shell []string) ([]action.Action, error) {
	actions, argIndex, err := tokenizeActions(args, argIndex, shell)

	if err != nil {
		return actions, fmt.Errorf("error tokenizing actions: %w", err)
	}
	if len(args) != argIndex {
		return actions, fmt.Errorf("failed to parse all the actions (%d/%d)", argIndex, len(args))
	}

	return actions, nil
}
