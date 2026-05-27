package fault

import "testing"

func TestMakeUserErrorfTagMessageIsFormatted(t *testing.T) {
	var tagged = MakeUserErrorf("field %q is required", "name")

	var msg = GetTagMessage(tagged)
	if msg != `field "name" is required` {
		t.Errorf("expected formatted tag message, got %q", msg)
	}
}
