package fault

import (
	"errors"
	"strings"
)

type _CoalesceError struct {
	errs []error
}

var _ Fault = (*_CoalesceError)(nil)

func (s *_CoalesceError) Error() string {
	var msgs = make([]string, len(s.errs))
	for i, err := range s.errs {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "; ")
}

func (s *_CoalesceError) Is(target error) bool {
	if len(s.errs) == 0 {
		return false
	}
	return errors.Is(s.errs[len(s.errs)-1], target)
}

func (s *_CoalesceError) As(target any) bool {
	if len(s.errs) == 0 {
		return false
	}
	return errors.As(s.errs[len(s.errs)-1], target)
}

func (s *_CoalesceError) Unwrap() error {
	if len(s.errs) <= 1 {
		return nil
	}
	return &_CoalesceError{errs: s.errs[:len(s.errs)-1]}
}

func (s *_CoalesceError) Errors() []error {
	return s.errs
}

func (s *_CoalesceError) From(err error, message string) Fault {
	return &_ErrorWithCause{_Error: _Error{msg: message}, cause: Wrap(err, s.Error())}
}

func (s *_CoalesceError) WithMessage(msg string) Fault {
	return &_ErrorWithCause{_Error: _Error{msg: msg}, cause: s}
}

func (s *_CoalesceError) AsUserFault(msg string) Fault {
	return tag(s, UserFault, msg)
}

func (s *_CoalesceError) AsValueMissing(msg string) Fault {
	return tag(s, NotFoundFault, msg)
}

func (s *_CoalesceError) AsRetryable(msg string) Fault {
	return tag(s, RetryableFault, msg)
}

func (s *_CoalesceError) AsAuthFault(msg string) Fault {
	return tag(s, AuthFault, msg)
}

func Coalesce(errs ...error) error {
	var nonNilErrs = make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			nonNilErrs = append(nonNilErrs, err)
		}
	}

	switch len(nonNilErrs) {
	case 0:
		return nil
	case 1:
		return nonNilErrs[0]
	default:
		return &_CoalesceError{errs: nonNilErrs}
	}
}
