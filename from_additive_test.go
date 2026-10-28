package fault_test

import (
	"errors"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/gleno/fault"
)

func TestFromKeepsReceiver(t *testing.T) {
	cases := map[string]func(fault.Fault, error) fault.Fault{
		"sentinel": func(base fault.Fault, _ error) fault.Fault { return base },
		"message":  func(base fault.Fault, _ error) fault.Fault { return base.WithMessage("lookup") },
		"tag":      func(base fault.Fault, _ error) fault.Fault { return base.AsRetryable("temporary") },
		"tag and message": func(base fault.Fault, _ error) fault.Fault {
			return base.AsRetryable("temporary").WithMessage("lookup")
		},
		"existing cause": func(base fault.Fault, cause error) fault.Fault { return base.From(cause, "first attempt") },
		"coalesced":      func(base fault.Fault, cause error) fault.Fault { return fault.Coalesce(cause, base).(fault.Fault) },
	}
	for name, makeReceiver := range cases {
		t.Run(name, func(t *testing.T) {
			sentinel := fault.Sentinel("operation failed")
			oldCause := errors.New("first failure")
			newCause := errors.New("second failure")
			receiver := makeReceiver(sentinel, oldCause)
			message, trace := receiver.Error(), fault.GetStackTrace(receiver)

			derived := receiver.From(newCause, "second attempt")
			for _, target := range []error{receiver, sentinel, newCause} {
				if !errors.Is(derived, target) {
					t.Errorf("From lost %T: %v", target, target)
				}
			}
			if errors.Is(receiver, oldCause) && !errors.Is(derived, oldCause) {
				t.Error("From replaced the receiver's existing cause")
			}
			if !strings.Contains(derived.Error(), message) || !strings.Contains(derived.Error(), "second attempt") {
				t.Errorf("From lost context: %q", derived.Error())
			}
			if fault.IsRetryable(receiver) && (!fault.IsRetryable(derived) || fault.GetTagMessage(derived) != "temporary") {
				t.Error("From lost the receiver's classification or reason")
			}
			if receiver.Error() != message || !slices.Equal(fault.GetStackTrace(receiver), trace) || errors.Is(receiver, newCause) {
				t.Error("From mutated its receiver")
			}
		})
	}
}

func TestFromRetainsEveryCauseAcrossDecorations(t *testing.T) {
	base := fault.Sentinel("operation failed")
	oldCause := &os.PathError{Op: "open", Path: "config", Err: errors.New("first failure")}
	newCause := &strconv.NumError{Func: "Atoi", Num: "bad", Err: strconv.ErrSyntax}
	first := base.From(oldCause, "first attempt")
	decorated := first.AsRetryable("temporary").WithMessage("loading profile")
	derived := decorated.From(newCause, "second attempt").WithMessage("returning")

	for _, target := range []error{base, first, decorated, oldCause, newCause, strconv.ErrSyntax} {
		if !errors.Is(derived, target) {
			t.Errorf("From lost %T: %v", target, target)
		}
	}
	var pathErr *os.PathError
	var numErr *strconv.NumError
	if !errors.As(derived, &pathErr) || pathErr != oldCause || !errors.As(derived, &numErr) || numErr != newCause {
		t.Error("errors.As must find both concrete failures")
	}
	for _, text := range []string{"operation failed", "first attempt", "loading profile", "second attempt", "returning", "temporary"} {
		if strings.Count(derived.Error(), text) != 1 {
			t.Errorf("want %q exactly once in %q", text, derived.Error())
		}
	}
	if !fault.IsRetryable(derived) || fault.GetTagMessage(derived) != "temporary" {
		t.Error("From lost the classification after decoration")
	}
}

func TestFromKeepsReceiverTagPrecedence(t *testing.T) {
	base := fault.Sentinel("operation failed").AsRetryable("temporary").WithMessage("lookup")
	cause := fault.MakeAuthErrorf("credentials")
	derived := base.From(cause, "request")
	if !fault.IsRetryable(derived) || fault.IsAuthFault(derived) || fault.GetTagMessage(derived) != "temporary" {
		t.Error("the new cause must not override the receiver's tag")
	}
	if !errors.Is(derived, cause) {
		t.Error("the differently tagged cause must remain discoverable")
	}
	if !fault.IsAuthFault(fault.Sentinel("untagged").From(cause, "request")) {
		t.Error("an untagged receiver must still expose the new cause's tag")
	}
}

func TestFromNilKeepsReceiverAndAddsContext(t *testing.T) {
	base := fault.Sentinel("operation failed")
	cause := errors.New("failure")
	receiver := base.From(cause, "first").AsRetryable("temporary").WithMessage("lookup")
	for _, message := range []string{"", "extra context"} {
		derived := receiver.From(nil, message)
		for _, target := range []error{base, receiver, cause} {
			if !errors.Is(derived, target) {
				t.Errorf("From(nil) lost %v", target)
			}
		}
		if !strings.Contains(derived.Error(), receiver.Error()) || !strings.Contains(derived.Error(), message) {
			t.Errorf("From(nil) lost context: %q", derived.Error())
		}
		if !fault.IsRetryable(derived) {
			t.Error("From(nil) lost retryability")
		}
	}
}
