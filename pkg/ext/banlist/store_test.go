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

func TestMemoryStoreProxyDisableSources(t *testing.T) {
	store := NewMemoryStore()

	changed := store.DisableProxySource("alice.tcp", ProxyDisableSourceManual, "manual", "ops")
	if !changed {
		t.Fatalf("first proxy disable source should change state")
	}
	if !store.IsProxyDisabled("alice.tcp") {
		t.Fatalf("proxy should be disabled after adding first source")
	}

	changed = store.DisableProxySource("alice.tcp", "schedule:workday", "schedule", "system")
	if !changed {
		t.Fatalf("second proxy disable source should change state")
	}

	record := store.GetProxy("alice.tcp")
	if record.Status != StatusDisabled {
		t.Fatalf("expected disabled status, got %q", record.Status)
	}
	if len(record.Sources) != 2 {
		t.Fatalf("expected 2 proxy disable sources, got %d", len(record.Sources))
	}

	removed := store.EnableProxySource("alice.tcp", ProxyDisableSourceManual)
	if !removed {
		t.Fatalf("expected manual source removal to succeed")
	}
	if !store.IsProxyDisabled("alice.tcp") {
		t.Fatalf("proxy should remain disabled while schedule source exists")
	}

	removed = store.EnableProxySource("alice.tcp", "schedule:workday")
	if !removed {
		t.Fatalf("expected schedule source removal to succeed")
	}
	if store.IsProxyDisabled("alice.tcp") {
		t.Fatalf("proxy should be enabled after removing all sources")
	}

	record = store.GetProxy("alice.tcp")
	if record.Status != StatusEnabled {
		t.Fatalf("expected enabled status after removing all sources, got %q", record.Status)
	}
}
