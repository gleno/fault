package fault

import (
	"strings"
	"testing"
)

func TestRetriableErrors(t *testing.T) {
	var base = Errorf("base error")
	var retriable = TagAsRetryable(base, "reason")

	if !IsRetryable(retriable) {
		t.Errorf("expected error to be retriable")
	}

	if IsRetryable(base) {
		t.Errorf("expected error to not be retriable")
	}
}

func TestDeepRetriableError(t *testing.T) {
	var base = Errorf("base error")
	var retriable = TagAsRetryable(base, "reason")
	var retriable2 = TagAsRetryable(retriable, "reason")

	if !IsRetryable(retriable2) {
		t.Errorf("expected error to be retriable")
	}

	if IsRetryable(base) {
		t.Errorf("expected error to not be retriable")
	}
}

func TestTagAsUserFault(t *testing.T) {
	var base = Errorf("bad input")
	var tagged = TagAsUserFault(base, "validation failed")

	if !IsUserFault(tagged) {
		t.Errorf("expected user fault")
	}
}

func TestMarkAsNotFound(t *testing.T) {
	var base = Errorf("sql: no rows")
	var tagged = TagAsNotFound(base, "user not found")

	if !IsNotFoundFault(tagged) {
		t.Errorf("expected not found fault")
	}
}

func TestMarkAsAuthError(t *testing.T) {
	var base = Errorf("token expired")
	var tagged = TagAsAuthError(base, "unauthorized")

	if !IsAuthFault(tagged) {
		t.Errorf("expected auth fault")
	}
}

func TestMakeUserFaultf(t *testing.T) {
	var tagged = MakeUserErrorf("field %q is required", "name")

	if !IsUserFault(tagged) {
		t.Errorf("expected user fault")
	}
}

func TestGetTagMessage(t *testing.T) {
	var base = Errorf("root")
	var tagged = TagAsRetryable(base, "temporary outage")

	var msg = GetTagMessage(tagged)
	if msg != "temporary outage" {
		t.Errorf("expected tag message 'temporary outage', got %q", msg)
	}
}

func TestGetTagMessageReturnsEmptyForUntagged(t *testing.T) {
	var err = Errorf("plain error")

	var msg = GetTagMessage(err)
	if msg != "" {
		t.Errorf("expected empty tag message for untagged error, got %q", msg)
	}
}

func TestTagNilReturnsNil(t *testing.T) {
	var result = TagAsRetryable(nil, "reason")
	if result != nil {
		t.Errorf("expected nil when tagging nil error")
	}
}

func TestUntaggedErrorDefaultsToInternal(t *testing.T) {
	var err = Errorf("internal problem")

	if extractTag(err) != InternalFault {
		t.Errorf("expected InternalFault for untagged error, got %d", extractTag(err))
	}
}

func TestMissingValue(t *testing.T) {
	var err = MissingValue("user")

	if !IsNotFoundFault(err) {
		t.Errorf("expected not found fault")
	}
}

func TestMakeAuthErrorf(t *testing.T) {
	var tagged = MakeAuthErrorf("user %q is not allowed", "bob")

	if !IsAuthFault(tagged) {
		t.Errorf("expected auth fault")
	}
}

func TestMakeRetryableErrorf(t *testing.T) {
	var tagged = MakeRetryableErrorf("attempt %d failed", 2)

	if !IsRetryable(tagged) {
		t.Errorf("expected retryable fault")
	}
}

func TestTaggedErrorFrom(t *testing.T) {
	var tagged = TagAsRetryable(Errorf("base"), "reason")
	var derived = tagged.From(Errorf("downstream"), "while fetching")

	if !strings.Contains(derived.Error(), "while fetching") {
		t.Errorf("expected wrapper message in error, got %q", derived.Error())
	}
	if !strings.Contains(derived.Error(), "downstream") {
		t.Errorf("expected cause message in error, got %q", derived.Error())
	}
}

func TestTaggedErrorAsUserFault(t *testing.T) {
	var tagged = TagAsRetryable(Errorf("base"), "reason")
	var f = tagged.AsUserFault("bad input")

	if !IsUserFault(f) {
		t.Errorf("expected user fault")
	}
}

func TestTaggedErrorAsValueMissing(t *testing.T) {
	var tagged = TagAsRetryable(Errorf("base"), "reason")
	var f = tagged.AsValueMissing("missing record")

	if !IsNotFoundFault(f) {
		t.Errorf("expected not found fault")
	}
}

func TestTaggedErrorAsRetryable(t *testing.T) {
	var tagged = TagAsUserFault(Errorf("base"), "reason")
	var f = tagged.AsRetryable("will retry")

	if !IsRetryable(f) {
		t.Errorf("expected retryable fault")
	}
}

func TestTaggedErrorAsAuthFault(t *testing.T) {
	var tagged = TagAsRetryable(Errorf("base"), "reason")
	var f = tagged.AsAuthFault("forbidden")

	if !IsAuthFault(f) {
		t.Errorf("expected auth fault")
	}
}
