package parsing

import (
	"fmt"
	"strings"
)

type actionType int

type actionToken struct {
	typ  actionType
	text string
}

type actionInfo struct {
	Name   string
	Action string
}

func ActionInfo() map[string]actionInfo {
	// TODO implement storage of custom actions / exclusion of defaults.
	return defaultActionInfo()
}
func defaultActionInfo() map[string]actionInfo {
	return map[string]actionInfo{
		"-status": {
			Name:   "-status",
			Action: "git status",
		},
		"-shortstatus": {
			Name:   "-shortStatus",
			Action: "git status -sb",
		},
		"-stashes": {
			Name:   "-stashes",
			Action: "git stash list",
		},
	}
}

func tokenizeActions(args []string, argIndex int) ([]string, int, error) {
	numArgs := len(args)
	actions := make([]string, 0, numArgs-argIndex)

	actionOptions := ActionInfo()
	for numArgs != argIndex {
		thisArg := strings.Trim(strings.ToLower(args[argIndex]), " \t")
		if entry, ok := actionOptions[thisArg]; ok {
			actions = append(actions, entry.Action)
		} else {
			return actions, argIndex, fmt.Errorf("Unknown action flag '%s'", args[argIndex])
		}
		argIndex++
	}

	return actions, argIndex, nil
}

func parseActions(args []string, argIndex int) ([]string, error) {
	actions, argIndex, err := tokenizeActions(args, argIndex)

	if err != nil {
		return actions, fmt.Errorf("Error tokenizing actions: %w", err)
	}
	if len(args) != argIndex {
		return actions, fmt.Errorf("Failed to parse all the actions (%d/%d)", argIndex, len(args))
	}

	return actions, nil
}
