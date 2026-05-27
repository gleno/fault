package fault

import (
	"strings"
	"testing"
)

func TestGetFullErrorNoMessageDuplication(t *testing.T) {
	chain := From(From(Errorf("DEEPESTTOKEN"), "midmsg"), "topmsg")

	full := GetFullError(chain)

	if deepCount := strings.Count(full, "DEEPESTTOKEN"); deepCount != 1 {
		t.Errorf("deepest token duplicated: appears %d times, want 1\n%s", deepCount, full)
	}
	if midCount := strings.Count(full, "midmsg"); midCount != 1 {
		t.Errorf("mid message duplicated: appears %d times, want 1\n%s", midCount, full)
	}
}
