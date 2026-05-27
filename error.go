package fault

import "sync/atomic"

var _sentinelIDs atomic.Uint64

type _Error struct {
	id  uint64
	msg string
}

var _ Fault = (*_Error)(nil)

func (s *_Error) Error() string {
	return s.msg
}

// Is matches on the sentinel identity carried by the error, not on its message
// text. A zero id means the error is not a sentinel and matches nothing by value.
func (s *_Error) Is(target error) bool {
	t, ok := target.(*_Error)
	if !ok {
		return false
	}
	return s.id != 0 && s.id == t.id
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
