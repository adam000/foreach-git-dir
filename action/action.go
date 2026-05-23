package action

type Action interface {
	Name() string
	Summary() string
	Run(repoPath string) (string, error)
}
