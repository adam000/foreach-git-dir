package parsing

import "testing"

func TestEmptyCommandLine(t *testing.T) {
	args := []string{}

	_, err := ParseCommandLine(args, "", nil)

	if err == nil {
		t.Errorf("Expected error parsing empty command line (with simulated no config), but that didn't happen")
	}
}

func TestEmptyCommandLineWithConfig(t *testing.T) {
	args := []string{}

	directives, err := ParseCommandLine(args, "/my/default/root", nil)

	if err != nil {
		t.Errorf("Unexpected error parsing empty command line with config: %v", err)
	}

	if directives.RootDir != "/my/default/root" {
		t.Errorf("Expected root dir to be set to default, but got '%s'", directives.RootDir)
	}
}
