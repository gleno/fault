package fault

import (
	"strings"
	"testing"
)

func TestGetFullErrorNoBlankLineBeforeStack(t *testing.T) {
	var err = Errorf("boom")
	var full = GetFullError(err)

	if strings.Contains(full, "\n\n") {
		t.Errorf("GetFullError emitted a blank line between message and stack trace: %q", full)
	}
	if !strings.Contains(full, "boom") {
		t.Errorf("expected message in output, got %q", full)
	}
	if !strings.Contains(full, "getfullerror_blankline_test.go") {
		t.Errorf("expected stack frame in output, got %q", full)
	}
}
