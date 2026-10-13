package fault

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestTaggedErrorRendersReasonOnce(t *testing.T) {
	var base = errors.New("BOOM")
	var tagged = TagAsRetryable(base, "REASON")

	var cases = []struct {
		name    string
		err     error
		message string
		reason  string
		tag     Tag
	}{
		{"TagAsRetryable", TagAsRetryable(base, "REASON"), "REASON: BOOM", "REASON", RetryableFault},
		{"TagAsUserFault", TagAsUserFault(base, "REASON"), "REASON: BOOM", "REASON", UserFault},
		{"TagAsNotFound", TagAsNotFound(base, "REASON"), "REASON: BOOM", "REASON", NotFoundFault},
		{"TagAsAuthError", TagAsAuthError(base, "REASON"), "REASON: BOOM", "REASON", AuthFault},
		{"TagAsRetryable over Wrap", TagAsRetryable(Wrap(base, "CALLING"), "REASON"), "REASON: CALLING: BOOM", "REASON", RetryableFault},
		{"MakeUserErrorf", MakeUserErrorf("MADE %d", 1), "MADE 1", "MADE 1", UserFault},
		{"MakeAuthErrorf", MakeAuthErrorf("MADE %d", 1), "MADE 1", "MADE 1", AuthFault},
		{"MakeRetryableErrorf", MakeRetryableErrorf("MADE %d", 1), "MADE 1", "MADE 1", RetryableFault},
		{"MissingValue", MissingValue("MISSING"), "MISSING", "MISSING", NotFoundFault},
		{"tagged.From", tagged.From(errors.New("DOWN"), "FETCHING"), "REASON: FETCHING: DOWN", "REASON", RetryableFault},
		{"tagged.WithMessage", tagged.WithMessage("OUTER"), "OUTER: REASON: BOOM", "REASON", RetryableFault},
		{"tagged.AsUserFault", tagged.AsUserFault("RETAG"), "RETAG: REASON: BOOM", "RETAG", UserFault},
		{"tagged.AsValueMissing", tagged.AsValueMissing("RETAG"), "RETAG: REASON: BOOM", "RETAG", NotFoundFault},
		{"tagged.AsRetryable", tagged.AsRetryable("RETAG"), "RETAG: REASON: BOOM", "RETAG", RetryableFault},
		{"tagged.AsAuthFault", tagged.AsAuthFault("RETAG"), "RETAG: REASON: BOOM", "RETAG", AuthFault},
		{"Sentinel.AsRetryable", Sentinel("SENTINEL").AsRetryable("RETAG"), "RETAG: SENTINEL", "RETAG", RetryableFault},
		{"From.AsUserFault", From(base, "CONTEXT").AsUserFault("RETAG"), "RETAG: CONTEXT: BOOM", "RETAG", UserFault},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.err.Error(); got != c.message {
				t.Errorf("Error(): want %q, got %q", c.message, got)
			}
			if got := GetTagMessage(c.err); got != c.reason {
				t.Errorf("GetTagMessage: want %q, got %q", c.reason, got)
			}
			if got := extractTag(c.err); got != c.tag {
				t.Errorf("tag: want %d, got %d", c.tag, got)
			}

			var stack = GetStackTrace(c.err)
			if len(stack) == 0 {
				t.Fatalf("expected a stack trace")
			}
			if top := fmt.Sprintf("%+v", stack[0]); !strings.Contains(top, "tagged_reason_dup_test.go") {
				t.Errorf("top frame should be the caller, got %s", top)
			}

			var full = GetFullError(c.err)
			if !strings.HasPrefix(full, c.message+"\n") || strings.Count(full, c.message) != 1 {
				t.Errorf("GetFullError should render the message once, then the stack: %q", full)
			}
		})
	}
}
