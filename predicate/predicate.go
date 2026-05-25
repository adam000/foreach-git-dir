package predicate

import (
	"os/exec"
	"strings"
)

type Predicate func(string) (bool, error)

func Id(_ string) (bool, error) {
	return true, nil
}

func And(p1, p2 Predicate) Predicate {
	return func(root string) (bool, error) {
		result, err := p1(root)
		if result && err == nil {
			result, err = p2(root)
		}
		return result, err
	}
}

func Or(p1, p2 Predicate) Predicate {
	return func(root string) (bool, error) {
		result, err := p1(root)
		if err != nil {
			return result, err
		}
		if !result {
			result, err = p2(root)
		}
		return result, nil
	}
}

func Custom(command string, shell []string) Predicate {
	return func(root string) (bool, error) {
		cmd := exec.Command(shell[0], append(shell[1:], command)...)
		cmd.Dir = root
		_, err := cmd.Output()

		return err == nil, nil
	}
}

func Not(pred Predicate) Predicate {
	return func(root string) (bool, error) {
		result, err := pred(root)
		return !result, err
	}
}

func IsDirty(root string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = root
	out, _ := cmd.Output()

	return len(out) != 0, nil
}

func Grep(pattern string) Predicate {
	return func(root string) (bool, error) {
		params := []string{"grep", "-q"}
		// Smart search: if the pattern is all lowercase, ignore case
		if strings.ToLower(pattern) == pattern {
			params = append(params, "-i")
		}
		params = append(params, pattern)

		cmd := exec.Command("git", params...)
		cmd.Dir = root
		err := cmd.Run()

		return err == nil, nil
	}
}
