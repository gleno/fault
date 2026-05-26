package fault

import (
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
