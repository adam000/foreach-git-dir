package parsing

import (
	"path/filepath"
	"testing"

	"github.com/adam000/foreach-git-dir/config"
)

func TestEmptyCommandLine(t *testing.T) {
	args := []string{}

	_, err := ParseCommandLine(args, config.Config{})

	if err == nil {
		t.Errorf("Expected error parsing empty command line (with simulated no config), but that didn't happen")
	}
}

func TestEmptyCommandLineWithConfig(t *testing.T) {
	args := []string{}

	directives, err := ParseCommandLine(args, config.Config{RootDir: "/my/default/root"})

	if err != nil {
		t.Errorf("Unexpected error parsing empty command line with config: %v", err)
	}

	if directives.RootDir != "/my/default/root" {
		t.Errorf("Expected root dir to be set to default, but got '%s'", directives.RootDir)
	}
}

func TestRelativeExcludeResolvedAgainstRootDir(t *testing.T) {
	root := t.TempDir()

	directives, err := ParseCommandLine([]string{root, "-exclude", "junk"}, config.Config{})
	if err != nil {
		t.Fatalf("Unexpected error parsing command line with relative exclude: %v", err)
	}

	want := filepath.Join(root, "junk")
	if len(directives.Excludes) != 1 || directives.Excludes[0] != want {
		t.Errorf("Expected excludes to be ['%s'], but got %v", want, directives.Excludes)
	}
}

func TestAbsoluteExcludeKeptAsIs(t *testing.T) {
	root := t.TempDir()

	directives, err := ParseCommandLine([]string{root, "-exclude", "/some/other/dir"}, config.Config{})
	if err != nil {
		t.Fatalf("Unexpected error parsing command line with absolute exclude: %v", err)
	}

	want := filepath.Clean("/some/other/dir")
	if len(directives.Excludes) != 1 || directives.Excludes[0] != want {
		t.Errorf("Expected excludes to be ['%s'], but got %v", want, directives.Excludes)
	}
}

func TestMixedExcludes(t *testing.T) {
	root := t.TempDir()

	directives, err := ParseCommandLine([]string{root, "-exclude", "junk", "-exclude", "/some/other/dir"}, config.Config{})
	if err != nil {
		t.Fatalf("Unexpected error parsing command line with mixed excludes: %v", err)
	}

	want := []string{filepath.Join(root, "junk"), filepath.Clean("/some/other/dir")}
	if len(directives.Excludes) != len(want) {
		t.Fatalf("Expected %d excludes, but got %d: %v", len(want), len(directives.Excludes), directives.Excludes)
	}
	for i := range want {
		if directives.Excludes[i] != want[i] {
			t.Errorf("Expected excludes[%d] to be '%s', but got '%s'", i, want[i], directives.Excludes[i])
		}
	}
}
