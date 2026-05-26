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
	return &_Error{msg}
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
	return TagAsUserFault(errors.Errorf(format, args...), format)
}

func MakeAuthErrorf(format string, args ...any) Fault {
	return TagAsAuthError(errors.Errorf(format, args...), format)
}

func MakeRetryableErrorf(format string, args ...any) Fault {
	return TagAsRetryable(errors.Errorf(format, args...), format)
}

// GetFullError will write all .Error() messages in possibly wrapped error,
// and also attach stack trace if it exists
func GetFullError(err error) string {

	var fullError string
	var stackTrace = GetStackTrace(err)
	if stackTrace != nil {
		fullError = fmt.Sprintf("%+v\n", stackTrace)
	}

	fullError = fmt.Sprintf("%s\n%s", fullError, err.Error())
	for cause := errors.Unwrap(err); cause != nil; cause = errors.Unwrap(cause) {
		fullError = fmt.Sprintf("%s\n%s", fullError, cause.Error())
	}

	return fullError
}
