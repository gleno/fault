package fault

import (
	"fmt"
	"strings"
	"testing"
)

func TestErrorfHasStackTrace(t *testing.T) {
	var err = Errorf("something broke")
	var st = GetStackTrace(err)

	if st == nil {
		t.Fatal("expected Errorf to produce a stack trace")
	}

	if len(st) == 0 {
		t.Fatal("expected non-empty stack trace")
	}
}

func TestErrorfStackTracePointsAtCaller(t *testing.T) {
	var err = Errorf("boom")
	var st = GetStackTrace(err)

	var topFrame = fmt.Sprintf("%+v", st[0])
	if !strings.Contains(topFrame, "stack_test.go") {
		t.Errorf("expected top frame to be in stack_test.go, got %s", topFrame)
	}
}

func TestSentinelHasNoStackTrace(t *testing.T) {
	var s = Sentinel("not found")
	var st = GetStackTrace(s)

	if st != nil {
		t.Errorf("expected sentinel to have no stack trace, got %+v", st)
	}
}

func TestWrapAddsStackTrace(t *testing.T) {
	var plain = fmt.Errorf("plain error")
	var wrapped = Wrap(plain, "context")

	var st = GetStackTrace(wrapped)
	if st == nil {
		t.Fatal("expected Wrap to add a stack trace to a plain error")
	}
}

func TestWrapIdempotentStackTrace(t *testing.T) {
	var base = Errorf("base")
	var baseTrace = GetStackTrace(base)

	var wrapped = Wrap(base, "layer 1")
	var wrappedTrace = GetStackTrace(wrapped)

	if len(wrappedTrace) != len(baseTrace) {
		t.Errorf(
			"expected Wrap to reuse existing stack trace (len %d), but got a different one (len %d)",
			len(baseTrace), len(wrappedTrace),
		)
	}
}

func TestWrapDoubleWrapDoesNotDuplicateTrace(t *testing.T) {
	var base = Errorf("base")
	var w1 = Wrap(base, "layer 1")
	var w2 = Wrap(w1, "layer 2")

	var baseLen = len(GetStackTrace(base))
	var w2Len = len(GetStackTrace(w2))

	if w2Len != baseLen {
		t.Errorf("expected double-wrap to keep original stack trace length %d, got %d", baseLen, w2Len)
	}
}

func TestWrapNilReturnsNil(t *testing.T) {
	var result = Wrap(nil, "context")
	if result != nil {
		t.Errorf("expected Wrap(nil) to return nil")
	}
}

func TestWrapfNilReturnsNil(t *testing.T) {
	var result = Wrapf(nil, "context %d", 1)
	if result != nil {
		t.Errorf("expected Wrapf(nil) to return nil")
	}
}

func TestWrapfFormatsMessage(t *testing.T) {
	var base = Errorf("root")
	var wrapped = Wrapf(base, "failed at step %d", 3)

	if !strings.Contains(wrapped.Error(), "failed at step 3") {
		t.Errorf("expected formatted message, got %q", wrapped.Error())
	}
}

func TestGetStackTraceReturnsNilForPlainError(t *testing.T) {
	var plain = fmt.Errorf("plain")
	var st = GetStackTrace(plain)

	if st != nil {
		t.Errorf("expected nil stack trace for plain error")
	}
}

func TestPopStackNilReturnsNil(t *testing.T) {
	if popStack(nil) != nil {
		t.Errorf("expected popStack(nil) to return nil")
	}
}

func TestPopStackStacklessErrorReturnsUnchanged(t *testing.T) {
	var stackless = Sentinel("no stack field here")

	var out error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("popStack panicked on a stackless error: %v", r)
			}
		}()
		out = popStack(stackless)
	}()

	if out != stackless {
		t.Errorf("expected popStack to return the stackless error unchanged")
	}
}

func panicWithError() (err error) {
	defer func() { RecoverPanic(recover(), &err) }()
	panic(fmt.Errorf("oh no"))
}

func panicWithString() (err error) {
	defer func() { RecoverPanic(recover(), &err) }()
	panic("something bad")
}

func noPanic() (err error) {
	defer func() { RecoverPanic(recover(), &err) }()
	return nil
}

func TestRecoverPanicWithError(t *testing.T) {
	var err = panicWithError()

	if err == nil {
		t.Fatal("expected RecoverPanic to capture the panic")
	}

	if !strings.Contains(err.Error(), "caught panic") {
		t.Errorf("expected 'caught panic' in message, got %q", err.Error())
	}

	if !strings.Contains(err.Error(), "oh no") {
		t.Errorf("expected original error message, got %q", err.Error())
	}

	var st = GetStackTrace(err)
	if st == nil {
		t.Fatal("expected recovered panic to have a stack trace")
	}
}

func TestRecoverPanicWithString(t *testing.T) {
	var err = panicWithString()

	if err == nil {
		t.Fatal("expected RecoverPanic to capture the string panic")
	}

	if !strings.Contains(err.Error(), "something bad") {
		t.Errorf("expected panic string in message, got %q", err.Error())
	}

	var st = GetStackTrace(err)
	if st == nil {
		t.Fatal("expected recovered string panic to have a stack trace")
	}
}

func TestRecoverPanicWithNil(t *testing.T) {
	var err = noPanic()

	if err != nil {
		t.Errorf("expected no error when there is no panic, got %v", err)
	}
}

func TestGetFullErrorIncludesStackTrace(t *testing.T) {
	var err = Errorf("root cause")
	var full = GetFullError(err)

	if !strings.Contains(full, "root cause") {
		t.Errorf("expected error message in full output, got %q", full)
	}

	if !strings.Contains(full, "stack_test.go") {
		t.Errorf("expected stack trace file reference in full output, got %q", full)
	}
}

func TestGetFullErrorWithWrappedChain(t *testing.T) {
	var base = Errorf("db timeout")
	var wrapped = Wrap(base, "query failed")
	var full = GetFullError(wrapped)

	if !strings.Contains(full, "query failed") {
		t.Errorf("expected wrapper message in full output")
	}
	if !strings.Contains(full, "db timeout") {
		t.Errorf("expected base message in full output")
	}
}

func TestGetFullErrorWithPlainError(t *testing.T) {
	var err = fmt.Errorf("plain error")
	var full = GetFullError(err)

	if !strings.Contains(full, "plain error") {
		t.Errorf("expected error message in output, got %q", full)
	}
}
