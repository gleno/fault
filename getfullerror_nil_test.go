package fault

import "testing"

func TestGetFullErrorNil(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GetFullError(nil) panicked: %v", r)
		}
	}()
	_ = GetFullError(nil)
}
