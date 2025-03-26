package standalone

import "testing"

func TestStandaloneCommand(t *testing.T) {
	// Stops the linter from complaining
	_ = NewStandaloneCommand()
}
