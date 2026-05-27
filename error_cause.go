package fault

import "fmt"

type _ErrorWithCause struct {
	_Error
	cause error
}

var _ Fault = (*_ErrorWithCause)(nil)

func (s *_ErrorWithCause) Error() string {
	if s.cause == nil {
		return s.msg
	}
	if s.msg == "" {
		return s.cause.Error()
	}
	return fmt.Sprintf("%s: %s", s.msg, s.cause.Error())
}

func (s *_ErrorWithCause) Unwrap() error {
	return s.cause
}

func (s *_ErrorWithCause) From(err error, message string) Fault {
	return &_ErrorWithCause{_Error: s._Error, cause: Wrap(err, message)}
}

func (s *_ErrorWithCause) WithMessage(msg string) Fault {
	return &_ErrorWithCause{_Error: _Error{msg: msg}, cause: s}
}

func (s *_ErrorWithCause) AsUserFault(msg string) Fault {
	return tag(s, UserFault, msg)
}

func (s *_ErrorWithCause) AsValueMissing(msg string) Fault {
	return tag(s, NotFoundFault, msg)
}

func (s *_ErrorWithCause) AsRetryable(msg string) Fault {
	return tag(s, RetryableFault, msg)
}

func (s *_ErrorWithCause) AsAuthFault(msg string) Fault {
	return tag(s, AuthFault, msg)
}
