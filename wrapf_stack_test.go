package fault

import (
	"fmt"
	"strings"
	"testing"
)

func TestWrapStackPointsAtCaller(t *testing.T) {
	var base = fmt.Errorf("plain")
	var wrapped = Wrap(base, "context")
	var st = GetStackTrace(wrapped)

	var topFrame = fmt.Sprintf("%+v", st[0])
	if !strings.Contains(topFrame, "wrapf_stack_test.go") {
		t.Errorf("Wrap: expected top frame in this test file (the caller), got %s", topFrame)
	}
}

func TestWrapfStackPointsAtCaller(t *testing.T) {
	var base = fmt.Errorf("plain")
	var wrapped = Wrapf(base, "context %d", 1)
	var st = GetStackTrace(wrapped)

	var topFrame = fmt.Sprintf("%+v", st[0])
	if !strings.Contains(topFrame, "wrapf_stack_test.go") {
		t.Errorf("Wrapf: expected top frame in this test file (the caller), got %s", topFrame)
	}
}

func TestWrapfIdempotentStackTrace(t *testing.T) {
	var base = Errorf("base")
	var baseLen = len(GetStackTrace(base))

	var wrapped = Wrapf(base, "layer %d", 1)
	var wrappedLen = len(GetStackTrace(wrapped))

	if wrappedLen != baseLen {
		t.Errorf("Wrapf must reuse the existing stack trace (len %d), got len %d", baseLen, wrappedLen)
	}
}

func TestWrapEmptyMsgDoesNotMutateCauseStack(t *testing.T) {
	var base = Errorf("base")
	var baseLen = len(GetStackTrace(base))

	var wrapped = Wrap(base, "")

	if got := len(GetStackTrace(base)); got != baseLen {
		t.Errorf("Wrap(base, \"\") mutated the cause stack: was %d, now %d", baseLen, got)
	}
	if got := len(GetStackTrace(wrapped)); got != baseLen {
		t.Errorf("Wrap(base, \"\") returned a shortened stack: want %d, got %d", baseLen, got)
	}
}
