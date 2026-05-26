package fault

import (
	"fmt"
	"reflect"
	"runtime"
	"unsafe"

	"github.com/pkg/errors"
)

// StackTrace should be aliases rather than newtype'd, so it can work with any of the
// functions we export from pkg/errors.
type StackTrace = errors.StackTrace

type StackTracer interface {
	StackTrace() errors.StackTrace
}

// popStack removes the top of the stack from an errors stack trace.
func popStack(err error) error {
	if err == nil {
		return err
	}

	// We want to remove us, the internal/errors.Errorf function, from the error stack we just
	// produced. There's no official way of reaching into the error and adjusting this, as
	// the stack is stored as a private field on an unexported struct.
	//
	// This does some unsafe badness to adjust that field, which should not be repeated
	// anywhere else.
	var stackField = reflect.ValueOf(err).Elem().FieldByName("stack")
	if stackField.IsZero() {
		return err
	}
	var stackFieldPtr = (**[]uintptr)(unsafe.Pointer(stackField.UnsafeAddr()))

	// Remove the first of the frames, dropping 'us' from the error stack trace.
	frames := (**stackFieldPtr)[1:]

	// Assign to the internal stack field
	*stackFieldPtr = &frames

	return err
}

func callers(skip int) []uintptr {
	var pc = make([]uintptr, 32)        // assume we'll have at most 32 frames
	var n = runtime.Callers(skip+3, pc) // capture those frames, skipping runtime.Callers, ourselves and the calling function
	return pc[:n]                       // return everything that we captured
}

// ancestorOfCause returns true if the caller looks to be an ancestor of the given stack
// trace. We check this by seeing whether our stack prefix-matches the cause stack, which
// should imply the error was generated directly from our goroutine.
func ancestorOfCause(ourStack []uintptr, causeStack errors.StackTrace) bool {
	// Stack traces are ordered such that the deepest frame is first. We'll want to check
	// for prefix matching in reverse.
	//
	// As an example, imagine we have a prefix-matching stack for ourselves:
	// [
	//   "github.com/onsi/ginkgo/internal/leafnodes.(*runner).runSync",
	//   "github.com/incident-io/core/server/pkg/errors_test.TestSuite",
	//   "testing.tRunner",
	//   "runtime.goexit"
	// ]
	//
	// We'll want to compare this against an error cause that will have happened further
	// down the stack. An example stack trace from such an error might be:
	// [
	//   "github.com/incident-io/core/server/pkg/errors.Errorf",
	//   "github.com/incident-io/core/server/pkg/errors_test.glob..func1.2.2.2.1",,
	//   "github.com/onsi/ginkgo/internal/leafnodes.(*runner).runSync",
	//   "github.com/incident-io/core/server/pkg/errors_test.TestSuite",
	//   "testing.tRunner",
	//   "runtime.goexit"
	// ]
	//
	// They prefix match, but we'll have to handle the match carefully as we need to match
	// from back to forward.

	// We can't possibly prefix match if our stack is larger than the cause stack.
	if len(ourStack) > len(causeStack) {
		return false
	}

	// We know the sizes are compatible, so compare program counters from back to front.
	for idx := 0; idx < len(ourStack); idx++ {
		if ourStack[len(ourStack)-1-idx] != (uintptr)(causeStack[len(causeStack)-1-idx]) {
			return false
		}
	}

	// All comparisons checked out, these stacks match
	return true
}

// Public API

// Errorf acts as pkg/errors.New, producing a stack traced error, but supports
// interpolating of message parameters. Use this when you want the stack trace to start at
// the place you create the error.
func Errorf(msg string, args ...any) error {
	return popStack(errors.New(fmt.Sprintf(msg, args...)))
}

// Wrap creates a new error from a cause, decorating the original error message with a
// prefix.
//
// It differs from the pkg/errors Wrap/Wrapf by idempotently creating a stack trace,
// meaning we won't create another stack trace when there is already a stack trace present
// that matches our current program position.
func Wrap(cause error, msg string) error {

	if cause == nil {
		return nil
	}

	var causeStackTracer = new(StackTracer)
	if errors.As(cause, causeStackTracer) {
		// If our cause has set a stack trace, and that trace is a child of our own function
		// as inferred by prefix matching our current program counter stack, then we only want
		// to decorate the error message rather than add a redundant stack trace.
		if ancestorOfCause(callers(1), (*causeStackTracer).StackTrace()) {
			return errors.WithMessage(cause, msg) // no stack added, no pop required
		}
	}

	// Otherwise we can't see a stack trace that represents ourselves, so let's add one.
	return popStack(errors.Wrap(cause, msg))
}

func Wrapf(cause error, msg string, args ...any) error {
	if cause == nil {
		return nil
	}
	var message = fmt.Sprintf(msg, args...)
	return Wrap(cause, message)
}

func GetStackTrace(err error) StackTrace {
	var stackTracer = new(StackTracer)
	if errors.As(err, stackTracer) {
		return (*stackTracer).StackTrace()
	}
	return nil
}
