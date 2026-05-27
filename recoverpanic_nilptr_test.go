package fault

import (
	"fmt"
	"testing"
)

func TestRecoverPanicNilErrPtr(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RecoverPanic panicked with nil errPtr: %v", r)
		}
	}()

	RecoverPanic(fmt.Errorf("oh no"), nil)
}
