package banlist

import "testing"

func TestMemoryStoreDisableEnable(t *testing.T) {
	store := NewMemoryStore()

	record, changed := store.Disable("client-a", "incident", "alice")
	if !changed {
		t.Fatalf("first disable should mark state changed")
	}
	if record.Status != StatusDisabled {
		t.Fatalf("expected status %q, got %q", StatusDisabled, record.Status)
	}

	record, changed = store.Disable("client-a", "incident", "alice")
	if changed {
		t.Fatalf("same disable request should be idempotent")
	}
	if record.Status != StatusDisabled {
		t.Fatalf("expected status %q, got %q", StatusDisabled, record.Status)
	}

	_, changed = store.Enable("client-a", "alice")
	if !changed {
		t.Fatalf("enable should change disabled client status")
	}
	disabled, _ := store.IsDisabled("client-a")
	if disabled {
		t.Fatalf("client should be enabled after enable call")
	}

	_, changed = store.Enable("client-a", "alice")
	if changed {
		t.Fatalf("enabling already enabled client should be idempotent")
	}
}
