package fault

// _stackError carries a message, an optional cause, and the call stack captured at the
// moment it was created. It is the building block behind Errorf, Wrap and RecoverPanic;
// the public Fault types embed it as their cause to gain a stack trace.
type _stackError struct {
	msg   string
	cause error
	stack stack
}

func (e *_stackError) Error() string {
	if e.cause == nil {
		return e.msg
	}
	if e.msg == "" {
		return e.cause.Error()
	}
	return e.msg + ": " + e.cause.Error()
}

func (e *_stackError) Unwrap() error { return e.cause }

func (e *_stackError) StackTrace() StackTrace { return e.stack.trace() }

// _messageError prepends a message to a cause without capturing a new stack. Wrap uses it
// to decorate an error that already carries a stack rooted in the current call path, so we
// add the message without stacking a second, redundant trace. It deliberately does not
// implement StackTracer, leaving traversal to find the real stack deeper in the chain.
type _messageError struct {
	msg   string
	cause error
}

func (e *_messageError) Error() string {
	if e.msg == "" {
		return e.cause.Error()
	}
	return e.msg + ": " + e.cause.Error()
}

func (e *_messageError) Unwrap() error { return e.cause }
