package fault

import "testing"

func TestTaggedErrorFromPreservesTag(t *testing.T) {
	var retryable = TagAsRetryable(Errorf("base"), "transient outage")

	if !IsRetryable(retryable) {
		t.Fatalf("precondition failed: tagged error should be retryable")
	}

	var derived = retryable.From(Errorf("downstream"), "while fetching")

	if !IsRetryable(derived) {
		t.Errorf("derived error should remain retryable after .From() (extractTag=%d)", extractTag(derived))
	}
}
