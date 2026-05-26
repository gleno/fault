package fault

import (
	"errors"
	"testing"
)

func TestSentinelError(t *testing.T) {
	var s = Sentinel("something went wrong")

	if s.Error() != "something went wrong" {
		t.Errorf("expected 'something went wrong', got %q", s.Error())
	}
}

func TestSentinelUnwrapReturnsNil(t *testing.T) {
	var s = Sentinel("err")

	if s.Unwrap() != nil {
		t.Errorf("expected nil unwrap for sentinel")
	}
}

func TestSentinelIsMatchesSelf(t *testing.T) {
	var s = Sentinel("not found")

	if !errors.Is(s, s) {
		t.Errorf("expected sentinel to match itself")
	}
}

func TestSentinelIsDoesNotMatchDifferent(t *testing.T) {
	var a = Sentinel("not found")
	var b = Sentinel("timeout")

	if errors.Is(a, b) {
		t.Errorf("expected different sentinels to not match")
	}
}

func TestSentinelFrom(t *testing.T) {
	var s = Sentinel("not found")
	var cause = Errorf("db failure")
	var wrapped = s.From(cause, "lookup user")

	if wrapped.Error() != "not found: lookup user: db failure" {
		t.Errorf("unexpected error message: %q", wrapped.Error())
	}
}

func TestSentinelCauseIsMatchesSentinel(t *testing.T) {
	var s = Sentinel("not found")
	var cause = Errorf("db failure")
	var wrapped = s.From(cause, "lookup user")

	if !errors.Is(wrapped, s) {
		t.Errorf("expected sentinelCause to match its sentinel via errors.Is")
	}
}

func TestSentinelCauseIsDoesNotMatchOtherSentinel(t *testing.T) {
	var s = Sentinel("not found")
	var other = Sentinel("timeout")
	var wrapped = s.From(Errorf("cause"), "msg")

	if errors.Is(wrapped, other) {
		t.Errorf("expected sentinelCause to not match a different sentinel")
	}
}

func TestSentinelWithMessage(t *testing.T) {
	var base = Sentinel("base")
	var wrapped = base.WithMessage("extra context")

	if !errors.Is(wrapped, base) {
		t.Errorf("expected WithMessage result to match sentinel via errors.Is")
	}
}

func TestSentinelAsRetryable(t *testing.T) {
	var s = Sentinel("temporary")
	var f = s.AsRetryable("retrying")

	if !IsRetryable(f) {
		t.Errorf("expected fault to be retryable")
	}
}

func TestSentinelAsUserFault(t *testing.T) {
	var s = Sentinel("bad input")
	var f = s.AsUserFault("invalid field")

	if !IsUserFault(f) {
		t.Errorf("expected fault to be user fault")
	}
}

func TestSentinelAsValueMissing(t *testing.T) {
	var s = Sentinel("missing")
	var f = s.AsValueMissing("record not found")

	if extractTag(f) != NotFoundFault {
		t.Errorf("expected NotFoundFault, got %d", extractTag(f))
	}
}

func TestSentinelCauseAsRetryable(t *testing.T) {
	var s = Sentinel("temporary")
	var cause = Errorf("connection reset")
	var f = s.From(cause, "fetch failed").AsRetryable("will retry")

	if !IsRetryable(f) {
		t.Errorf("expected fault to be retryable")
	}

	if !errors.Is(f, s) {
		t.Errorf("expected retryable fault to still match sentinel via errors.Is")
	}
}

func TestSentinelCauseAsUserFault(t *testing.T) {
	var s = Sentinel("bad request")
	var f = s.From(Errorf("validation"), "parse").AsUserFault("invalid")

	if !IsUserFault(f) {
		t.Errorf("expected fault to be user fault")
	}
}

func TestSentinelCauseAsValueMissing(t *testing.T) {
	var s = Sentinel("not found")
	var f = s.From(Errorf("sql: no rows"), "query").AsValueMissing("user missing")

	if extractTag(f) != NotFoundFault {
		t.Errorf("expected NotFoundFault, got %d", extractTag(f))
	}
}

func TestFaultBuilderFrom(t *testing.T) {
	var cause = Errorf("root cause")
	var f = From(cause, "context")

	if f.Error() != "context: root cause" {
		t.Errorf("unexpected error: %q", f.Error())
	}
}

func TestFaultBuilderChaining(t *testing.T) {
	var cause = Errorf("root")
	var f = From(cause, "step1").WithMessage("step2").AsRetryable("retrying").WithMessage("sdfa")

	if !IsRetryable(f) {
		t.Errorf("expected retryable")
	}
}
