package fault

import (
	"errors"
	"testing"
)

func TestCoalesceErrorsEncapsulationLeak(t *testing.T) {
	a := errors.New("alpha")
	b := errors.New("beta")
	ce := Coalesce(a, b)

	before := ce.Error()
	if before != "alpha; beta" {
		t.Fatalf("unexpected initial Error(): %q", before)
	}

	got := ce.(*_CoalesceError).Errors()
	got[0] = errors.New("HIJACKED")

	after := ce.Error()
	if after != before {
		t.Fatalf("internal state mutated via Errors(): before=%q after=%q", before, after)
	}
}
