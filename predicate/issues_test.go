package predicate

import (
	"os/exec"
	"testing"

	"github.com/adam000/foreach-git-dir/gh"
)

var _ gh.Runner = fakeRunner{}

type fakeRunner struct {
	lookPathFn func(string) (string, error)
	commandFn  func(string, ...string) *exec.Cmd
}

func (f fakeRunner) LookPath(file string) (string, error) {
	return f.lookPathFn(file)
}

func (f fakeRunner) Command(name string, args ...string) *exec.Cmd {
	return f.commandFn(name, args...)
}

func TestHasIssuesIgnoresDisabledIssues(t *testing.T) {
	runner = fakeRunner{
		lookPathFn: func(file string) (string, error) { return "/usr/bin/gh", nil },
		commandFn: func(name string, args ...string) *exec.Cmd {
			if len(args) >= 2 && args[0] == "repo" && args[1] == "view" {
				return exec.Command("sh", "-c", "exit 0")
			}
			if len(args) >= 2 && args[0] == "issue" && args[1] == "list" {
				return exec.Command("sh", "-c", "printf \"the 'lua/lua' repository has disabled issues\" 1>&2; exit 1")
			}
			t.Fatalf("unexpected command: %s %v", name, args)
			return nil
		},
	}

	result, err := HasIssues("/tmp")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result {
		t.Fatalf("expected false for disabled-issues repo, got true")
	}
}

func TestHasIssuesDetectsOpenIssues(t *testing.T) {
	runner = fakeRunner{
		lookPathFn: func(file string) (string, error) { return "/usr/bin/gh", nil },
		commandFn: func(name string, args ...string) *exec.Cmd {
			if len(args) >= 2 && args[0] == "repo" && args[1] == "view" {
				return exec.Command("sh", "-c", "exit 0")
			}
			if len(args) >= 2 && args[0] == "issue" && args[1] == "list" {
				return exec.Command("sh", "-c", "printf '123 open issue\n'")
			}
			t.Fatalf("unexpected command: %s %v", name, args)
			return nil
		},
	}

	result, err := HasIssues("/tmp")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result {
		t.Fatalf("expected true for open issues, got false")
	}
}

func TestHasIssuesDetectsNoOpenIssues(t *testing.T) {
	runner = fakeRunner{
		lookPathFn: func(file string) (string, error) { return "/usr/bin/gh", nil },
		commandFn: func(name string, args ...string) *exec.Cmd {
			if len(args) >= 2 && args[0] == "repo" && args[1] == "view" {
				return exec.Command("sh", "-c", "exit 0")
			}
			if len(args) >= 2 && args[0] == "issue" && args[1] == "list" {
				return exec.Command("sh", "-c", "echo \"no open issues in adam000/foreach-git-dir\"")
			}
			t.Fatalf("unexpected command: %s %v", name, args)
			return nil
		},
	}

	result, err := HasIssues("/tmp")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result {
		t.Fatalf("expected false for no open issues, got true")
	}
}
