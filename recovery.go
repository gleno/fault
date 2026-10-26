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
//
//go:noinline
func RecoverPanic(r any, errPtr *error) {
	if r == nil || errPtr == nil {
		return
	}

	site := panicOrigin(callers(0))
	if panicErr, ok := r.(error); ok {
		*errPtr = &_stackError{msg: "caught panic", cause: panicErr, stack: site}
	} else {
		*errPtr = &_stackError{msg: fmt.Sprintf("caught panic: %v", r), stack: site}
	}
}

// panicOrigin removes recovery helpers and the runtime's panic machinery. Find
// the panic boundary rather than assuming how many helpers called RecoverPanic.
// Direct calls outside a panic keep their caller's stack.
func panicOrigin(site stack) stack {
	for i, pc := range site {
		function, _, _ := Frame(pc).location()
		if function != "runtime.gopanic" {
			continue
		}
		for i++; i < len(site); i++ {
			function, _, _ = Frame(site[i]).location()
			if !isMachineryFrame(function) {
				return site[i:]
			}
		}
		break
	}
	return site
}
