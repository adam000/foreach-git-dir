package gh

import "os/exec"

type Runner interface {
	LookPath(file string) (string, error)
	Command(name string, args ...string) *exec.Cmd
}

type ExecRunner struct{}

func (ExecRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (ExecRunner) Command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
