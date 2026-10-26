package fault

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"slices"
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

// Format renders a single frame. "%+v" gives "<func>\n\t<file>:<line>"; "%v" gives
// the file:line; "%s" the file; "%d" the line.
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

// callers captures the caller of its caller, plus any additional frames to skip.
// Capture through the goroutine root: ancestorOfCause compares stack suffixes,
// which only align when neither stack has been truncated.
//
//go:noinline
func callers(skip int) []uintptr {
	var initial [32]uintptr
	pc := initial[:]
	for {
		n := runtime.Callers(skip+3, pc)
		if n < len(pc) {
			return slices.Clone(pc[:n])
		}
		pc = make([]uintptr, len(pc)*2)
	}
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
//
//go:noinline
func Errorf(msg string, args ...any) error {
	return &_stackError{msg: fmt.Sprintf(msg, args...), stack: callers(0)}
}

// Wrap creates a new error from a cause, decorating the original error message with a
// prefix. It idempotently creates a stack trace, meaning we won't create another stack
// trace when there is already one present that matches our current program position.
//
//go:noinline
func Wrap(cause error, msg string) error {
	return wrap(cause, msg, callers(0))
}

//go:noinline
func Wrapf(cause error, msg string, args ...any) error {
	if cause == nil {
		return nil
	}
	return wrap(cause, fmt.Sprintf(msg, args...), callers(0))
}

func wrap(cause error, msg string, site []uintptr) error {
	if cause == nil {
		return nil
	}

	var causeStackTracer = new(StackTracer)
	if errors.As(cause, causeStackTracer) {
		// If our cause has set a stack trace, and that trace is a child of our own function
		// as inferred by prefix matching our current program counter stack, then we only want
		// to decorate the error message rather than add a redundant stack trace.
		if ancestorOfCause(site[1:], (*causeStackTracer).StackTrace()) {
			if msg == "" {
				return cause // already carries our stack, nothing to decorate
			}
			return &_messageError{msg: msg, cause: cause}
		}
	}

	// Otherwise we can't see a stack trace that represents ourselves, so let's add one.
	return &_stackError{msg: msg, cause: cause, stack: site}
}

func GetStackTrace(err error) StackTrace {
	var stackTracer = new(StackTracer)
	if errors.As(err, stackTracer) {
		return (*stackTracer).StackTrace()
	}
	return nil
}
