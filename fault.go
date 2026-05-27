package fault

import (
	"fmt"

	"github.com/pkg/errors"
)

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
	return Sentinel(msg).AsValueMissing(msg)
}

func From(err error, message string) Fault {
	return &_ErrorWithCause{
		_Error: _Error{msg: message},
		cause:  err,
	}
}

func MakeUserErrorf(format string, args ...any) Fault {
	var msg = fmt.Sprintf(format, args...)
	return TagAsUserFault(errors.New(msg), msg)
}

func MakeAuthErrorf(format string, args ...any) Fault {
	var msg = fmt.Sprintf(format, args...)
	return TagAsAuthError(errors.New(msg), msg)
}

func MakeRetryableErrorf(format string, args ...any) Fault {
	var msg = fmt.Sprintf(format, args...)
	return TagAsRetryable(errors.New(msg), msg)
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
		return fmt.Sprintf("%+v\n%s", stackTrace, err.Error())
	}

	return err.Error()
}
