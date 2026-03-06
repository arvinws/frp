package clientmgr

import "testing"

type fakeController struct {
	closeCalls int
}

func (f *fakeController) Close() error {
	f.closeCalls++
	return nil
}

func TestManagerRegisterLookupDisconnect(t *testing.T) {
	mgr := NewManager()
	ctl := &fakeController{}

	mgr.Register("run-1", "client-a", "user-a", "127.0.0.1", ctl)

	byRun, ok := mgr.GetByRunID("run-1")
	if !ok {
		t.Fatalf("expected session by runID")
	}
	if byRun.ClientID != "client-a" {
		t.Fatalf("expected clientID client-a, got %q", byRun.ClientID)
	}

	byClient, ok := mgr.GetByClientID("client-a")
	if !ok {
		t.Fatalf("expected session by clientID")
	}
	if byClient.RunID != "run-1" {
		t.Fatalf("expected runID run-1, got %q", byClient.RunID)
	}

	if ok := mgr.DisconnectByRunID("run-1"); !ok {
		t.Fatalf("disconnect by runID should succeed")
	}
	if ctl.closeCalls != 1 {
		t.Fatalf("expected close to be called once, got %d", ctl.closeCalls)
	}

	mgr.Unregister("run-1")
	if _, ok := mgr.GetByRunID("run-1"); ok {
		t.Fatalf("session should be removed after unregister")
	}
	if _, ok := mgr.GetByClientID("client-a"); ok {
		t.Fatalf("client index should be removed after unregister")
	}
}
