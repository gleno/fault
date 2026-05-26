package fault

type _Error struct {
	msg string
}

var _ Fault = (*_Error)(nil)

func (s *_Error) Error() string {
	return s.msg
}

func (s *_Error) Is(target error) bool {
	if t, ok := target.(*_Error); ok {
		return s.msg == t.msg
	}
	return false
}

func (s *_Error) Unwrap() error {
	return nil
}

func (s *_Error) AsUserFault(msg string) Fault {
	return tag(s, UserFault, msg)
}

func (s *_Error) AsValueMissing(msg string) Fault {
	return tag(s, NotFoundFault, msg)
}

func (s *_Error) AsRetryable(msg string) Fault {
	return tag(s, RetryableFault, msg)
}

func (s *_Error) AsAuthFault(msg string) Fault {
	return tag(s, AuthFault, msg)
}

func (s *_Error) From(err error, message string) Fault {
	return &_ErrorWithCause{_Error: *s, cause: Wrap(err, message)}
}

func (s *_Error) WithMessage(msg string) Fault {
	return &_ErrorWithCause{_Error: _Error{msg: msg}, cause: s}
}
