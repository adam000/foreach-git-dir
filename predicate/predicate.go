package predicate

import (
	"os/exec"
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

func Custom(command string) Predicate {
	return func(root string) (bool, error) {
		// TODO
		return true, nil
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
