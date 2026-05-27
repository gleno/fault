package fault

import (
	"fmt"
	"strings"
	"testing"
)

func TestFromEmptyMessageNoLeadingSeparator(t *testing.T) {
	got := From(Errorf("root cause"), "").Error()

	if strings.HasPrefix(got, ": ") || strings.Contains(got, ": :") || strings.HasSuffix(got, ": ") {
		t.Fatalf("From(root, \"\").Error() has a malformed separator: %q", got)
	}
}

func TestBuilderFromEmptyMessageNoDoubledSeparator(t *testing.T) {
	got := Sentinel("x").From(Errorf("root cause"), "").Error()

	if strings.Contains(got, ": :") {
		t.Fatalf("Sentinel.From(root, \"\").Error() has a doubled separator: %q", got)
	}
}

func TestWrapEmptyMessageOnPlainErrorAddsStack(t *testing.T) {
	wrapped := Wrap(fmt.Errorf("plain"), "")

	if wrapped.Error() != "plain" {
		t.Errorf("empty-message Wrap must not alter the message, got %q", wrapped.Error())
	}
	if GetStackTrace(wrapped) == nil {
		t.Errorf("empty-message Wrap must still attach a stack trace")
	}
}

func TestWrapEmptyMessageReusesAncestorStack(t *testing.T) {
	base := Errorf("base")
	wrapped := Wrap(base, "")

	if wrapped != base {
		t.Errorf("empty-message Wrap of an error rooted in this frame should return it unchanged")
	}
}
