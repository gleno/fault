package fault

import "testing"

//go:noinline
func failDeep() error {
	return Errorf("BOOM")
}

func TestTagKeepsOriginStack(t *testing.T) {
	var deep = failDeep()
	var local = Errorf("BOOM")
	var deepTop, localTop = GetStackTrace(deep)[0], GetStackTrace(local)[0]

	var cases = []struct {
		name   string
		err    error
		origin Frame
	}{
		{"TagAsRetryable over deeper origin", TagAsRetryable(deep, "REASON"), deepTop},
		{"TagAsRetryable over local origin", TagAsRetryable(local, "REASON"), localTop},
		{"From.AsUserFault", From(deep, "CONTEXT").AsUserFault("REASON"), deepTop},
		{"Sentinel.From.AsAuthFault", Sentinel("SENTINEL").From(deep, "CONTEXT").AsAuthFault("REASON"), deepTop},
		{"tagged.AsValueMissing", TagAsRetryable(deep, "REASON").AsValueMissing("RETAG"), deepTop},
		{"tagged.From keeps receiver origin", TagAsRetryable(deep, "REASON").From(local, "FETCHING"), deepTop},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if top := GetStackTrace(c.err)[0]; top != c.origin {
				t.Errorf("tagging should keep the origin frame on top: want %+v, got %+v", c.origin, top)
			}
		})
	}

	if GetStackTrace(deep)[0] != deepTop || GetStackTrace(local)[0] != localTop {
		t.Errorf("tagging mutated the cause stack")
	}
}
