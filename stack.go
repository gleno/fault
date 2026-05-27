package fault

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
)

// StackTrace is a captured call stack, innermost (origin) frame first.
type StackTrace []Frame

type StackTracer interface {
	StackTrace() StackTrace
}

// Frame is a single program counter from a captured stack.
type Frame uintptr

func (f Frame) pc() uintptr { return uintptr(f) - 1 }

func (f Frame) location() (function, file string, line int) {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "", "", 0
	}
	file, line = fn.FileLine(f.pc())
	return fn.Name(), file, line
}

// Format renders a single frame. "%+v" gives "<func>\n\t<file>:<line>"; "%v"/"%s" give
// the file:line; "%d" the line.
func (f Frame) Format(s fmt.State, verb rune) {
	function, file, line := f.location()
	switch verb {
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, trimFuncName(function))
			io.WriteString(s, "\n\t")
		}
		fmt.Fprintf(s, "%s:%d", file, line)
	case 's':
		io.WriteString(s, file)
	case 'd':
		fmt.Fprintf(s, "%d", line)
	}
}

// Format renders the whole trace for "%+v", one frame per block, omitting Go runtime
// machinery so the trace stays legible.
func (st StackTrace) Format(s fmt.State, verb rune) {
	if verb != 'v' || !s.Flag('+') {
		return
	}
	for _, f := range st {
		function, _, _ := f.location()
		if isMachineryFrame(function) {
			continue
		}
		io.WriteString(s, "\n")
		f.Format(s, verb)
	}
}

func isMachineryFrame(function string) bool {
	return function == "" || strings.HasPrefix(function, "runtime.")
}

func trimFuncName(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}

type stack []uintptr

func (s stack) trace() StackTrace {
	trace := make(StackTrace, len(s))
	for i := range s {
		trace[i] = Frame(s[i])
	}
	return trace
}

// capture records the call stack with its top frame at the caller of the constructor that
// invokes it (skipping runtime.Callers, capture itself and that constructor). The public
// entry points then popStack their own frame so the trace begins at user code. Kept
// noinline so the frame arithmetic holds regardless of inlining decisions.
//
//go:noinline
func capture() stack {
	const depth = 32
	var pcs = make([]uintptr, depth)
	var n = runtime.Callers(3, pcs)
	return stack(pcs[:n])
}

//go:noinline
func newError(msg string, cause error) *_stackError {
	return &_stackError{msg: msg, cause: cause, stack: capture()}
}

//go:noinline
func withStackError(cause error) *_stackError {
	return &_stackError{cause: cause, stack: capture()}
}

// popStack removes the top frame from a stack error, dropping our own infrastructure frame
// so the trace begins at the code that called us. It is a no-op on errors without a stack.
func popStack(err error) error {
	if err == nil {
		return err
	}
	se, ok := err.(*_stackError)
	if !ok || len(se.stack) == 0 {
		return err
	}
	se.stack = se.stack[1:]
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
func ancestorOfCause(ourStack []uintptr, causeStack StackTrace) bool {
	// Stack traces are ordered such that the deepest frame is first. We'll want to check
	// for prefix matching in reverse.

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

// Errorf produces a stack-traced error, interpolating message parameters. Use this when
// you want the stack trace to start at the place you create the error.
func Errorf(msg string, args ...any) error {
	return popStack(newError(fmt.Sprintf(msg, args...), nil))
}

// Wrap creates a new error from a cause, decorating the original error message with a
// prefix. It idempotently creates a stack trace, meaning we won't create another stack
// trace when there is already one present that matches our current program position.
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
			if msg == "" {
				return cause // already carries our stack, nothing to decorate
			}
			return &_messageError{msg: msg, cause: cause} // no stack added, no pop required
		}
	}

	// An empty message adds no prefix; we still ensure a stack trace is present.
	if msg == "" {
		return popStack(withStackError(cause))
	}

	// Otherwise we can't see a stack trace that represents ourselves, so let's add one.
	return popStack(newError(msg, cause))
}

func Wrapf(cause error, msg string, args ...any) error {
	if cause == nil {
		return nil
	}
	return Wrap(cause, fmt.Sprintf(msg, args...))
}

func GetStackTrace(err error) StackTrace {
	var stackTracer = new(StackTracer)
	if errors.As(err, stackTracer) {
		return (*stackTracer).StackTrace()
	}
	return nil
}
