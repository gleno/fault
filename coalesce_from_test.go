package fault

import (
	"errors"
	"testing"
)

func TestCoalesceFromPreservesMembers(t *testing.T) {
	var e1 = Errorf("first")
	var e2 = Errorf("second")
	var coalesced = Coalesce(e1, e2).(*_CoalesceError)

	var derived = coalesced.From(Errorf("root cause"), "batch context")

	if !errors.Is(derived, e1) {
		t.Errorf("expected errors.Is to still find first member e1 after From")
	}
	if !errors.Is(derived, e2) {
		t.Errorf("expected errors.Is to still find second member e2 after From")
	}
}
