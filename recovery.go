package fault

import "fmt"

// RecoverPanic turns a panic into an error, adjusting the stacktrace so it originates at
// the line that caused it.
//
// Example:
//
//	func Do() (err error) {
//	  defer func() {
//	    fault.RecoverPanic(recover(), &err)
//	  }()
//	}
func RecoverPanic(r any, errPtr *error) {
	var err error
	if r != nil {
		if panicErr, ok := r.(error); ok {
			err = newError("caught panic", panicErr)
		} else {
			err = newError(fmt.Sprintf("caught panic: %v", r), nil)
		}
	}

	if err != nil && errPtr != nil {
		// Pop twice: once for newError, then again for the defer function we must run this
		// under. We want the stacktrace to originate at the source of the panic, not in the
		// infrastructure that catches it.
		err = popStack(err) // newError
		err = popStack(err) // defer

		*errPtr = err
	}
}
