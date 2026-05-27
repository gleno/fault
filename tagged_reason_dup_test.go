package fault

import (
	"strings"
	"testing"
)

func TestTaggedErrorReasonNotDuplicated(t *testing.T) {
	base := Errorf("connection refused")
	tagged := TagAsRetryable(base, "upstream unavailable")

	got := tagged.Error()
	n := strings.Count(got, "upstream unavailable")
	if n != 1 {
		t.Errorf("tag reason should appear exactly once, appeared %d times: %q", n, got)
	}
}
