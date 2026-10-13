package fault

import "fmt"

type Fault interface {
	error

	From(err error, message string) Fault
	WithMessage(message string) Fault

	AsUserFault(message string) Fault
	AsValueMissing(message string) Fault
	AsRetryable(message string) Fault
	AsAuthFault(message string) Fault

	Is(target error) bool
	Unwrap() error
}

// Sentinel is used to create compile-time errors that are intended to be value only, with
// no associated stack trace.
func Sentinel(msg string) Fault {
	return &_Error{id: _sentinelIDs.Add(1), msg: msg}
}

func MissingValue(msg string) Fault {
	return newTagged(NotFoundFault, msg)
}

//go:noinline
func From(err error, message string) Fault {
	return &_ErrorWithCause{
		_Error: _Error{msg: message},
		cause:  wrap(err, "", callers(1)),
	}
}

func MakeUserErrorf(format string, args ...any) Fault {
	return newTagged(UserFault, fmt.Sprintf(format, args...))
}

func MakeAuthErrorf(format string, args ...any) Fault {
	return newTagged(AuthFault, fmt.Sprintf(format, args...))
}

func MakeRetryableErrorf(format string, args ...any) Fault {
	return newTagged(RetryableFault, fmt.Sprintf(format, args...))
}

// GetFullError will write all .Error() messages in possibly wrapped error,
// and also attach stack trace if it exists
func GetFullError(err error) string {

	if err == nil {
		return ""
	}

	// err.Error() already renders the whole wrapped chain, so the stack trace is
	// all we add — re-walking Unwrap here would just repeat each layer's message.
	if stackTrace := GetStackTrace(err); stackTrace != nil {
		return fmt.Sprintf("%s%+v", err.Error(), stackTrace)
	}

	return err.Error()
}
