package parsing

import "testing"

func TestEmptyCommandLine(t *testing.T) {
	args := []string{}

	_, _, _, _, err := ParseCommandLine(args)

	if err == nil {
		t.Errorf("Expected error parsing empty command line, but that didn't happen")
	}
}
