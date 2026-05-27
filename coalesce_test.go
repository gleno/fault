package fault

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestCoalesceAllNilReturnsNil(t *testing.T) {
	var result = Coalesce(nil, nil, nil)

	if result != nil {
		t.Errorf("expected nil when all errors are nil")
	}
}

func TestCoalesceEmptyReturnsNil(t *testing.T) {
	var result = Coalesce()

	if result != nil {
		t.Errorf("expected nil for empty input")
	}
}

func TestCoalesceSingleErrorReturnsThatError(t *testing.T) {
	var err = Errorf("only one")
	var result = Coalesce(nil, err, nil)

	if result != err {
		t.Errorf("expected single non-nil error to be returned directly")
	}
}

func TestCoalesceErrorMessage(t *testing.T) {
	var e1 = fmt.Errorf("first")
	var e2 = fmt.Errorf("second")
	var e3 = fmt.Errorf("third")
	var result = Coalesce(e1, e2, e3)

	var msg = result.Error()
	if msg != "first; second; third" {
		t.Errorf("expected joined message, got %q", msg)
	}
}

func TestCoalesceIsFindsAllErrors(t *testing.T) {
	var e1 = Errorf("first")
	var e2 = Errorf("second")
	var e3 = Errorf("third")
	var result = Coalesce(e1, e2, e3)

	if !errors.Is(result, e1) {
		t.Errorf("expected errors.Is to find first error")
	}
	if !errors.Is(result, e2) {
		t.Errorf("expected errors.Is to find second error")
	}
	if !errors.Is(result, e3) {
		t.Errorf("expected errors.Is to find third error")
	}
}

func TestCoalesceIsDoesNotMatchUnrelated(t *testing.T) {
	var e1 = Errorf("first")
	var e2 = Errorf("second")
	var unrelated = Errorf("unrelated")
	var result = Coalesce(e1, e2)

	if errors.Is(result, unrelated) {
		t.Errorf("expected errors.Is to not match unrelated error")
	}
}

func TestCoalesceIsFindsSentinels(t *testing.T) {
	var errNotFound = Sentinel("not found")
	var errTimeout = Sentinel("timeout")

	var e1 = errNotFound.From(Errorf("db"), "query")
	var e2 = errTimeout.From(Errorf("network"), "fetch")
	var result = Coalesce(e1, e2)

	if !errors.Is(result, errNotFound) {
		t.Errorf("expected errors.Is to find not-found sentinel")
	}
	if !errors.Is(result, errTimeout) {
		t.Errorf("expected errors.Is to find timeout sentinel")
	}
}

func TestCoalesceIsFindsSentinelNotInGroup(t *testing.T) {
	var errNotFound = Sentinel("not found")
	var errAuth = Sentinel("unauthorized")

	var e1 = errNotFound.From(Errorf("db"), "query")
	var result = Coalesce(e1, Errorf("other"))

	if errors.Is(result, errAuth) {
		t.Errorf("expected errors.Is to not match sentinel not in group")
	}
}

type testTypedError struct {
	Code int
}

func (e *testTypedError) Error() string {
	return fmt.Sprintf("code %d", e.Code)
}

func TestCoalesceAsFindsTypedError(t *testing.T) {
	var e1 = fmt.Errorf("plain")
	var e2 = &testTypedError{Code: 42}
	var result = Coalesce(e1, e2)

	var target *testTypedError
	if !errors.As(result, &target) {
		t.Fatal("expected errors.As to find testTypedError")
	}
	if target.Code != 42 {
		t.Errorf("expected Code=42, got %d", target.Code)
	}
}

func TestCoalesceAsFindsTypedErrorInAnyPosition(t *testing.T) {
	var e1 = &testTypedError{Code: 99}
	var e2 = fmt.Errorf("plain")
	var result = Coalesce(e1, e2)

	var target *testTypedError
	if !errors.As(result, &target) {
		t.Fatal("expected errors.As to find testTypedError in first position")
	}
	if target.Code != 99 {
		t.Errorf("expected Code=99, got %d", target.Code)
	}
}

func TestCoalesceAsDoesNotMatchUnrelatedType(t *testing.T) {
	var e1 = fmt.Errorf("plain")
	var e2 = Errorf("traced")
	var result = Coalesce(e1, e2)

	var target *testTypedError
	if errors.As(result, &target) {
		t.Errorf("expected errors.As to not match unrelated type")
	}
}

func TestCoalesceAsFindsTaggedError(t *testing.T) {
	var e1 = Errorf("first")
	var e2 = TagAsRetryable(Errorf("flaky"), "temporary")
	var result = Coalesce(e1, e2)

	if !IsRetryable(result) {
		t.Errorf("expected coalesced error to be detected as retryable")
	}
}

func TestCoalesceUnwrapChain(t *testing.T) {
	var e1 = Errorf("first")
	var e2 = Errorf("second")
	var e3 = Errorf("third")
	var result = Coalesce(e1, e2, e3)

	var unwrapped = errors.Unwrap(result)
	if unwrapped == nil {
		t.Fatal("expected Unwrap to return remaining errors")
	}

	var inner, ok = unwrapped.(*_CoalesceError)
	if !ok {
		t.Fatalf("expected *_CoalesceError, got %T", unwrapped)
	}

	if len(inner.errs) != 2 {
		t.Errorf("expected 2 remaining errors, got %d", len(inner.errs))
	}
}

func TestCoalesceUnwrapSingleReturnsNil(t *testing.T) {
	var coalesced = &_CoalesceError{errs: []error{Errorf("only")}}
	var unwrapped = coalesced.Unwrap()

	if unwrapped != nil {
		t.Errorf("expected nil when unwrapping single-error coalesce, got %v", unwrapped)
	}
}

func TestCoalesceErrorsAccessor(t *testing.T) {
	var e1 = Errorf("first")
	var e2 = Errorf("second")
	var result = Coalesce(e1, e2)

	var coalesced = result.(*_CoalesceError)
	var errs = coalesced.Errors()

	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(errs))
	}
	if errs[0] != e1 || errs[1] != e2 {
		t.Errorf("expected original errors in order")
	}
}

func TestCoalesceWithMessage(t *testing.T) {
	var e1 = Errorf("first")
	var e2 = Errorf("second")
	var result = Coalesce(e1, e2)

	var wrapped = result.(*_CoalesceError).WithMessage("batch failed")

	if !strings.Contains(wrapped.Error(), "batch failed") {
		t.Errorf("expected 'batch failed' in message, got %q", wrapped.Error())
	}

	if !errors.Is(wrapped, e1) {
		t.Errorf("expected wrapped coalesced to still match first error")
	}
	if !errors.Is(wrapped, e2) {
		t.Errorf("expected wrapped coalesced to still match second error")
	}
}

func TestCoalesceAsRetryable(t *testing.T) {
	var e1 = Errorf("first")
	var e2 = Errorf("second")
	var result = Coalesce(e1, e2)

	var retryable = result.(*_CoalesceError).AsRetryable("batch can retry")

	if !IsRetryable(retryable) {
		t.Errorf("expected retryable")
	}

	if !errors.Is(retryable, e1) {
		t.Errorf("expected tagged coalesced to still match first error via errors.Is")
	}
	if !errors.Is(retryable, e2) {
		t.Errorf("expected tagged coalesced to still match second error via errors.Is")
	}
}

func TestCoalesceFiltersNils(t *testing.T) {
	var e1 = Errorf("real")
	var result = Coalesce(nil, e1, nil, nil)

	if result != e1 {
		t.Errorf("expected single non-nil to be returned directly")
	}
}

func TestCoalescePreservesWrappedErrors(t *testing.T) {
	var base = Errorf("base")
	var wrapped = Wrap(base, "context")
	var other = Errorf("other")
	var result = Coalesce(wrapped, other)

	if !errors.Is(result, base) {
		t.Errorf("expected errors.Is to traverse into wrapped error and find base")
	}
}

func TestCoalesceIsEmptyReturnsFalse(t *testing.T) {
	var empty = &_CoalesceError{}

	if errors.Is(empty, Errorf("anything")) {
		t.Errorf("expected empty coalesce to not match any error via errors.Is")
	}
}

func TestCoalesceAsEmptyReturnsFalse(t *testing.T) {
	var empty = &_CoalesceError{}

	var target *testTypedError
	if errors.As(empty, &target) {
		t.Errorf("expected empty coalesce to not match any type via errors.As")
	}
}

func TestCoalesceFrom(t *testing.T) {
	var result = Coalesce(Errorf("first"), Errorf("second")).(*_CoalesceError)
	var derived = result.From(Errorf("cause"), "batch context")

	if !strings.Contains(derived.Error(), "batch context") {
		t.Errorf("expected wrapper message in error, got %q", derived.Error())
	}
	if !strings.Contains(derived.Error(), "first; second") {
		t.Errorf("expected coalesced message preserved in cause, got %q", derived.Error())
	}
}

func TestCoalesceAsUserFault(t *testing.T) {
	var result = Coalesce(Errorf("first"), Errorf("second")).(*_CoalesceError)
	var f = result.AsUserFault("bad batch")

	if !IsUserFault(f) {
		t.Errorf("expected user fault")
	}
}

func TestCoalesceAsValueMissing(t *testing.T) {
	var result = Coalesce(Errorf("first"), Errorf("second")).(*_CoalesceError)
	var f = result.AsValueMissing("nothing found")

	if !IsNotFoundFault(f) {
		t.Errorf("expected not found fault")
	}
}

func TestCoalesceAsAuthFault(t *testing.T) {
	var result = Coalesce(Errorf("first"), Errorf("second")).(*_CoalesceError)
	var f = result.AsAuthFault("denied")

	if !IsAuthFault(f) {
		t.Errorf("expected auth fault")
	}
}
