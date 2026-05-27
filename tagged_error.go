package fault

import (
	"net/http"

	"github.com/pkg/errors"
)

type Tag int

const (
	InternalFault  Tag = http.StatusInternalServerError
	RetryableFault     = http.StatusGatewayTimeout
	UserFault          = http.StatusBadRequest
	AuthFault          = http.StatusUnauthorized
	NotFoundFault      = http.StatusNotFound
)

type _ErrorWithTag struct {
	_ErrorWithCause
	tag Tag
}

var _ Fault = (*_ErrorWithTag)(nil)

func tag(err error, tag Tag, reason string) Fault {

	if err == nil {
		return nil
	}

	var wrr = popStack(popStack(Wrap(err, reason)))

	return &_ErrorWithTag{
		_ErrorWithCause{
			_Error: _Error{msg: reason},
			cause:  wrr,
		}, tag,
	}
}

func (s *_ErrorWithTag) WithMessage(msg string) Fault {
	return &_ErrorWithCause{_Error: _Error{msg: msg}, cause: s}
}

func (s *_ErrorWithTag) From(err error, message string) Fault {
	return &_ErrorWithTag{
		_ErrorWithCause{_Error: s._Error, cause: Wrap(err, message)},
		s.tag,
	}
}

func (s *_ErrorWithTag) AsUserFault(msg string) Fault {
	return tag(s, UserFault, msg)
}

func (s *_ErrorWithTag) AsValueMissing(msg string) Fault {
	return tag(s, NotFoundFault, msg)
}

func (s *_ErrorWithTag) AsRetryable(msg string) Fault {
	return tag(s, RetryableFault, msg)
}

func (s *_ErrorWithTag) AsAuthFault(msg string) Fault {
	return tag(s, AuthFault, msg)
}

func extractTag(err error) Tag {

	var tagged *_ErrorWithTag
	if errors.As(err, &tagged) {
		return tagged.tag
	}

	return InternalFault
}

// Public API

func IsRetryable(err error) bool {
	return extractTag(err) == RetryableFault
}
func TagAsRetryable(err error, reason string) Fault {
	return tag(err, RetryableFault, reason)
}

func TagAsUserFault(err error, reason string) Fault {
	return tag(err, UserFault, reason)
}

func TagAsNotFound(err error, reason string) Fault {
	return tag(err, NotFoundFault, reason)
}

func TagAsAuthError(err error, reason string) Fault {
	return tag(err, AuthFault, reason)
}

func IsUserFault(err error) bool {
	return extractTag(err) == UserFault
}

func IsAuthFault(err error) bool { return extractTag(err) == AuthFault }

func IsNotFoundFault(err error) bool { return extractTag(err) == NotFoundFault }

func GetTagMessage(err error) string {
	var tagged *_ErrorWithTag
	if errors.As(err, &tagged) {
		return tagged.msg
	}

	return ""
}
