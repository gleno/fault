package fault

import (
	"errors"
	"testing"
)

//go:noinline
func descend(depth int, leaf func()) {
	if depth == 0 {
		leaf()
		return
	}
	descend(depth-1, leaf)
}

//go:noinline
func climb(height int, origin func() error) error {
	if height == 0 {
		return origin()
	}
	return climb(height-1, origin)
}

func TestWrapKeepsOriginStackAtAnyDepth(t *testing.T) {
	for _, depth := range []int{0, 20, 40, 64, 200} {
		for _, height := range []int{0, 40} {
			descend(depth, func() {
				var origin = climb(height, failDeep)
				var originTop = GetStackTrace(origin)[0]

				var cases = []struct {
					name string
					err  error
				}{
					{"Wrap", Wrap(origin, "CONTEXT")},
					{"Wrapf", Wrapf(origin, "CONTEXT %d", depth)},
					{"From", From(origin, "CONTEXT")},
					{"Sentinel.From", Sentinel("SENTINEL").From(origin, "CONTEXT")},
					{"TagAsRetryable", TagAsRetryable(origin, "REASON")},
				}

				for _, c := range cases {
					if top := GetStackTrace(c.err)[0]; top != originTop {
						t.Errorf("depth %d, origin height %d, %s: want origin frame %+v, got %+v", depth, height, c.name, originTop, top)
					}
				}
			})
		}
	}
}

func TestWrapAddsStackForCauseFromAnotherCallPath(t *testing.T) {
	var fromGoroutine = make(chan error)
	go func() { fromGoroutine <- failDeep() }()

	var stale error
	descend(0, func() { stale = failDeep() })

	var causes = []struct {
		name string
		err  error
	}{
		{"another goroutine", <-fromGoroutine},
		{"a returned call", stale},
	}

	for _, c := range causes {
		descend(0, func() {
			var wrapped = Wrap(c.err, "CONTEXT")
			if errors.Unwrap(wrapped) != c.err {
				t.Fatalf("%s: expected Wrap to wrap the cause directly", c.name)
			}
			if GetStackTrace(wrapped)[0] == GetStackTrace(c.err)[0] {
				t.Errorf("%s: expected a fresh stack at the wrap site, got the cause's origin", c.name)
			}
		})
	}
}

func TestWrapAddsStackForCauseFromAnotherGoroutineSharingFrames(t *testing.T) {
	for height := 0; height <= 40; height++ {
		var crossed = make(chan error)
		go descend(6, func() { crossed <- climb(height, failDeep) })

		descend(6, func() {
			var cause = <-crossed
			if GetStackTrace(Wrap(cause, "CONTEXT"))[0] == GetStackTrace(cause)[0] {
				t.Errorf("origin height %d: a cause from another goroutine kept its own stack instead of gaining the wrap site", height)
			}
		})
	}
}
