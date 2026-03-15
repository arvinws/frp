package governance

import (
	"testing"

	"github.com/fatedier/frp/pkg/ext/banlist"
	"github.com/fatedier/frp/pkg/ext/clientmgr"
	"github.com/fatedier/frp/pkg/msg"
)

type fakeRuntimeController struct {
	closed       []string
	controlCalls []struct {
		runID     string
		proxyName string
		action    string
	}
}

func (f *fakeRuntimeController) CloseProxyByName(name string) bool {
	f.closed = append(f.closed, name)
	return true
}

func (f *fakeRuntimeController) SendProxyControlByRunID(runID, proxyName, action string) bool {
	f.controlCalls = append(f.controlCalls, struct {
		runID     string
		proxyName string
		action    string
	}{runID: runID, proxyName: proxyName, action: action})
	return true
}

type fakeMetadataProvider struct {
	items map[string]ProxyMetadata
}

func (f fakeMetadataProvider) GetProxyMetadata(name string) (ProxyMetadata, bool) {
	item, ok := f.items[name]
	return item, ok
}

type fakeSessionController struct{}

func (fakeSessionController) Close() error { return nil }

func TestDisableAndEnableProxy(t *testing.T) {
	store := banlist.NewMemoryStore()
	sessionMgr := clientmgr.NewManager()
	sessionMgr.Register("run-1", "client-a", "alice", "127.0.0.1", fakeSessionController{})
	runtime := &fakeRuntimeController{}
	svc := NewService(store, sessionMgr, runtime, fakeMetadataProvider{
		items: map[string]ProxyMetadata{
			"alice.tcp": {ClientID: "client-a"},
		},
	})

	disableResp, err := svc.DisableProxy("alice.tcp", ManualProxySource(), "manual", "ops")
	if err != nil {
		t.Fatalf("disable proxy: %v", err)
	}
	if disableResp.Result != "disabled_and_closed" {
		t.Fatalf("unexpected disable result %q", disableResp.Result)
	}
	if len(runtime.controlCalls) != 1 || runtime.controlCalls[0].action != msg.ProxyControlActionDisable {
		t.Fatalf("expected one disable control call, got %#v", runtime.controlCalls)
	}
	if !store.IsProxyDisabled("alice.tcp") {
		t.Fatalf("proxy should be disabled")
	}

	enableResp, err := svc.EnableProxy("alice.tcp", ManualProxySource())
	if err != nil {
		t.Fatalf("enable proxy: %v", err)
	}
	if enableResp.Result != "enabled" {
		t.Fatalf("unexpected enable result %q", enableResp.Result)
	}
	if len(runtime.controlCalls) != 2 || runtime.controlCalls[1].action != msg.ProxyControlActionEnable {
		t.Fatalf("expected enable control call, got %#v", runtime.controlCalls)
	}
	if store.IsProxyDisabled("alice.tcp") {
		t.Fatalf("proxy should be enabled")
	}
}

func TestManualEnableClearsScheduleSources(t *testing.T) {
	store := banlist.NewMemoryStore()
	sessionMgr := clientmgr.NewManager()
	sessionMgr.Register("run-1", "client-a", "alice", "127.0.0.1", fakeSessionController{})
	runtime := &fakeRuntimeController{}
	svc := NewService(store, sessionMgr, runtime, fakeMetadataProvider{
		items: map[string]ProxyMetadata{
			"alice.tcp": {ClientID: "client-a"},
		},
	})

	if _, err := svc.DisableProxy("alice.tcp", ScheduleProxySource("task-1"), "schedule", "system"); err != nil {
		t.Fatalf("disable proxy by schedule: %v", err)
	}
	if _, err := svc.DisableProxy("alice.tcp", ScheduleProxySource("task-2"), "schedule", "system"); err != nil {
		t.Fatalf("disable proxy by second schedule: %v", err)
	}

	resp, err := svc.EnableProxy("alice.tcp", ManualProxySource())
	if err != nil {
		t.Fatalf("enable proxy: %v", err)
	}
	if resp.Result != "enabled" {
		t.Fatalf("expected enabled, got %q", resp.Result)
	}
	if resp.Disabled {
		t.Fatalf("proxy should be enabled after clearing schedule sources")
	}
	if store.IsProxyDisabled("alice.tcp") {
		t.Fatalf("proxy should be enabled after manual override")
	}
	if len(runtime.controlCalls) != 1 {
		t.Fatalf("expected one enable control call, got %#v", runtime.controlCalls)
	}
	if runtime.controlCalls[0].action != msg.ProxyControlActionEnable {
		t.Fatalf("expected enable control action, got %#v", runtime.controlCalls)
	}
}

func TestManualEnableRespectsOtherSources(t *testing.T) {
	store := banlist.NewMemoryStore()
	runtime := &fakeRuntimeController{}
	svc := NewService(store, clientmgr.NewManager(), runtime, fakeMetadataProvider{})

	if _, err := svc.DisableProxy("alice.tcp", ManualProxySource(), "manual", "ops"); err != nil {
		t.Fatalf("disable proxy: %v", err)
	}
	if _, err := svc.DisableProxy("alice.tcp", ScheduleProxySource("task-1"), "schedule", "system"); err != nil {
		t.Fatalf("disable proxy by schedule: %v", err)
	}
	if _, err := svc.DisableProxy("alice.tcp", "policy:lock", "policy", "system"); err != nil {
		t.Fatalf("disable proxy by policy: %v", err)
	}

	resp, err := svc.EnableProxy("alice.tcp", ManualProxySource())
	if err != nil {
		t.Fatalf("enable proxy: %v", err)
	}
	if resp.Result != "still_disabled" {
		t.Fatalf("expected still_disabled, got %q", resp.Result)
	}
	if !resp.Disabled {
		t.Fatalf("proxy should remain disabled")
	}
	if len(runtime.controlCalls) != 0 {
		t.Fatalf("expected no control call while proxy remains disabled, got %#v", runtime.controlCalls)
	}
}
