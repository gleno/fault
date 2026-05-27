package fault

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/pkg/errors"
)

func TestPopStackEmptyStackSlice(t *testing.T) {
	// errors.New produces a *fundamental with an embedded *stack pointing at a
	// real stack slice. Force that slice to length zero (non-nil pointer, empty
	// slice) and run it through popStack.
	err := errors.New("boom")

	stackField := reflect.ValueOf(err).Elem().FieldByName("stack")
	stackFieldPtr := (**[]uintptr)(unsafe.Pointer(stackField.UnsafeAddr()))
	empty := []uintptr{}
	*stackFieldPtr = &empty

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("popStack panicked on an empty stack slice: %v", r)
		}
	}()

	popStack(err)
}
