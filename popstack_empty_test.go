package fault

import "testing"

func TestPopStackEmptyStackSlice(t *testing.T) {
	// A stack error whose captured stack is empty must survive popStack without panicking.
	err := &_stackError{msg: "boom", stack: stack{}}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("popStack panicked on an empty stack slice: %v", r)
		}
	}()

	popStack(err)
}
