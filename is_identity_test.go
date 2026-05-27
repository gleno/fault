package fault

import (
	"errors"
	"testing"
)

func TestIsDoesNotMatchUnrelatedSameMessage(t *testing.T) {
	notFound := Sentinel("not found")
	unrelated := From(Errorf("disk exploded"), "not found")

	if errors.Is(unrelated, notFound) {
		t.Errorf("unrelated error sharing the sentinel's message must not match it via errors.Is")
	}
}

func TestIsDoesNotMatchUnrelatedWithMessage(t *testing.T) {
	notFound := Sentinel("not found")
	unrelated := Sentinel("disk exploded").WithMessage("not found")

	if errors.Is(unrelated, notFound) {
		t.Errorf("WithMessage-wrapped unrelated error must not match a sentinel by message text")
	}
}

func TestIsMatchesSameSentinelIdentity(t *testing.T) {
	s := Sentinel("base")
	a := s.From(Errorf("a"), "context a")
	b := s.WithMessage("context b")

	if !errors.Is(a, s) {
		t.Errorf("error derived from a sentinel via From must match it via errors.Is")
	}
	if !errors.Is(b, s) {
		t.Errorf("error derived from a sentinel via WithMessage must match it via errors.Is")
	}
}

func TestIsConsistentBetweenCauseAndTagged(t *testing.T) {
	causeA := From(Sentinel("inner"), "same msg")
	causeB := From(Sentinel("inner"), "same msg")
	causeMatch := errors.Is(causeA, causeB)

	taggedA := TagAsUserFault(Sentinel("inner"), "same msg")
	taggedB := TagAsUserFault(Sentinel("inner"), "same msg")
	taggedMatch := errors.Is(taggedA, taggedB)

	if causeMatch != taggedMatch {
		t.Errorf("errors.Is must behave the same for cause and tagged pairs: cause=%v tagged=%v", causeMatch, taggedMatch)
	}
}
