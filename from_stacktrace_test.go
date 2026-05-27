package fault

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestFromAttachesStackToStacklessCause(t *testing.T) {
	base := errors.New("root")
	f := From(base, "context")

	if st := GetStackTrace(f); st == nil {
		t.Fatalf("From(errors.New, ...) carries no stack trace; GetStackTrace returned nil")
	}
}

func TestFromStackPointsAtCaller(t *testing.T) {
	f := From(errors.New("root"), "context")
	st := GetStackTrace(f)

	topFrame := fmt.Sprintf("%+v", st[0])
	if !strings.Contains(topFrame, "from_stacktrace_test.go") {
		t.Errorf("From: expected top frame in this test file (the caller), got %s", topFrame)
	}
}

func TestFromPreservesErrorMessage(t *testing.T) {
	f := From(errors.New("root"), "context")
	if got := f.Error(); got != "context: root" {
		t.Errorf("From error message changed: want %q, got %q", "context: root", got)
	}
}

func TestFromIdempotentStackTrace(t *testing.T) {
	base := Errorf("root")
	baseLen := len(GetStackTrace(base))

	f := From(base, "context")
	if got := len(GetStackTrace(f)); got != baseLen {
		t.Errorf("From over an already-traced cause must reuse its stack (len %d), got len %d", baseLen, got)
	}
}

func TestFromNilCauseStaysNil(t *testing.T) {
	f := From(nil, "context")
	if f.Unwrap() != nil {
		t.Errorf("From(nil, ...) should keep a nil cause, got %v", f.Unwrap())
	}
	if got := f.Error(); got != "context" {
		t.Errorf("From(nil, ...) message: want %q, got %q", "context", got)
	}
}

func TestErrorFromStackPointsAtCaller(t *testing.T) {
	f := Sentinel("sentinel").From(errors.New("root"), "context")

	top := fmt.Sprintf("%+v", GetStackTrace(f)[0])
	if !strings.Contains(top, "from_stacktrace_test.go") {
		t.Errorf("_Error.From: expected top frame at the caller, got %s", top)
	}
}

func TestErrorWithCauseFromStackPointsAtCaller(t *testing.T) {
	parent := From(errors.New("base"), "outer")
	f := parent.From(errors.New("root"), "context")

	top := fmt.Sprintf("%+v", GetStackTrace(f)[0])
	if !strings.Contains(top, "from_stacktrace_test.go") {
		t.Errorf("_ErrorWithCause.From: expected top frame at the caller, got %s", top)
	}
}

func TestMethodFromIdempotentStackTrace(t *testing.T) {
	base := Errorf("root")
	baseLen := len(GetStackTrace(base))

	f := Sentinel("sentinel").From(base, "context")
	if got := len(GetStackTrace(f)); got != baseLen {
		t.Errorf("method From over an already-traced cause must reuse its stack (len %d), got len %d", baseLen, got)
	}
}
