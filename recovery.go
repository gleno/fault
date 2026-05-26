package fault

import (
	"fmt"

	"github.com/pkg/errors"
)

// RecoverPanic turns a panic into an error, adjusting the stacktrace so it originates at
// the line that caused it.
//
// Example:
//
//	func Do() (err error) {
//	  defer func() {
//	    errors.RecoverPanic(recover(), &err)
//	  }()
//	}
func RecoverPanic(r any, errPtr *error) {
	var err error
	if r != nil {
		if panicErr, ok := r.(error); ok {
			err = errors.Wrap(panicErr, "caught panic")
		} else {
			err = errors.New(fmt.Sprintf("caught panic: %v", r))
		}
	}

	if err != nil {
		// Pop twice: once for the errors package, then again for the defer function we must
		// run this under. We want the stacktrace to originate at the source of the panic, not
		// in the infrastructure that catches it.
		err = popStack(err) // errors.go
		err = popStack(err) // defer

		*errPtr = err
	}
}
