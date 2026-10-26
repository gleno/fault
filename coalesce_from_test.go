package fault

import (
	"errors"
	"fmt"
	"runtime"
	"slices"
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

func TestCoalesceFromAttachesCallerStack(t *testing.T) {
	for _, withCause := range []bool{false, true} {
		t.Run(fmt.Sprintf("withCause=%t", withCause), func(t *testing.T) {
			first := errors.New("first")
			last := &testTypedError{Code: 7}
			group := Coalesce(first, last).(Fault)
			var cause error
			wantMessage := "context: first; code 7"
			if withCause {
				cause = &testTypedError{Code: 42}
				wantMessage = "context: code 42; first; code 7"
			}

			_, file, line, _ := runtime.Caller(0)
			derived := group.From(cause, "context")
			trace := GetStackTrace(derived)
			if len(trace) == 0 {
				t.Fatal("From did not attach a stack to the stackless combined cause")
			}
			if got, want := fmt.Sprintf("%v", trace[0]), fmt.Sprintf("%s:%d", file, line+1); got != want {
				t.Errorf("stack origin = %s, want caller %s", got, want)
			}
			if got := derived.Error(); got != wantMessage {
				t.Errorf("Error() = %q, want %q", got, wantMessage)
			}
			for _, member := range []error{first, last, cause} {
				if member != nil && !errors.Is(derived, member) {
					t.Errorf("From lost member %v", member)
				}
			}
			var typed *testTypedError
			if !errors.As(derived, &typed) || typed != last {
				t.Errorf("errors.As selected %v, want last group member %v", typed, last)
			}
		})
	}
}

func TestCoalesceFromReusesOriginAndTagPrecedence(t *testing.T) {
	origin := MakeRetryableErrorf("last")
	trace := GetStackTrace(origin)
	group := Coalesce(MakeAuthErrorf("first"), origin).(Fault)
	for _, cause := range []error{nil, MakeUserErrorf("new cause")} {
		derived := group.From(cause, "context")
		if got := GetStackTrace(derived); !slices.Equal(got, trace) {
			t.Errorf("From replaced the same-path origin stack: got %+v, want %+v", got, trace)
		}
		if !IsRetryable(derived) || GetTagMessage(derived) != "last" {
			t.Error("From changed last-member tag precedence")
		}
	}
}
