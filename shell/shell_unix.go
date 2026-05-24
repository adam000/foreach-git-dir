//go:build !windows

package shell

func GetDefault() []string {
	return []string{"sh", "-c"}
}
