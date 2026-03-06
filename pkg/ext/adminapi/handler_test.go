package adminapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"github.com/fatedier/frp/pkg/ext/audit"
	"github.com/fatedier/frp/pkg/ext/banlist"
	"github.com/fatedier/frp/pkg/ext/clientmgr"
	httppkg "github.com/fatedier/frp/pkg/util/http"
)

type fakeController struct {
	closeCalls int
}

func (f *fakeController) Close() error {
	f.closeCalls++
	return nil
}

func TestDisableAndDisconnect(t *testing.T) {
	store := banlist.NewMemoryStore()
	sessionMgr := clientmgr.NewManager()
	recorder := audit.NewMemoryRecorder()
	handler := NewHandler(store, sessionMgr, recorder)

	ctl := &fakeController{}
	sessionMgr.Register("run-1", "client-a", "user-a", "127.0.0.1", ctl)

	ctx := newTestContext(http.MethodPost,
		"/api/admin/clients/client-a/disable-and-disconnect",
		`{"reason":"manual block","operator":"ops"}`,
		map[string]string{"clientID": "client-a"},
	)

	out, err := handler.DisableAndDisconnect(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp := out.(DisableAndDisconnectResponse)
	if resp.Status != banlist.StatusDisabled {
		t.Fatalf("expected status %q, got %q", banlist.StatusDisabled, resp.Status)
	}
	if resp.DisconnectResult != "disconnected" {
		t.Fatalf("expected disconnect result disconnected, got %q", resp.DisconnectResult)
	}
	if ctl.closeCalls != 1 {
		t.Fatalf("expected close to be called once, got %d", ctl.closeCalls)
	}

	disabled, _ := store.IsDisabled("client-a")
	if !disabled {
		t.Fatalf("expected client to be disabled after orchestration")
	}
}

func TestDisconnectSessionAlreadyOffline(t *testing.T) {
	handler := NewHandler(banlist.NewMemoryStore(), clientmgr.NewManager(), audit.NewMemoryRecorder())

	ctx := newTestContext(
		http.MethodPost,
		"/api/admin/sessions/run-404/disconnect",
		`{"reason":"manual","operator":"ops"}`,
		map[string]string{"runID": "run-404"},
	)

	out, err := handler.DisconnectSession(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp := out.(DisconnectResponse)
	if resp.Result != "already_offline" {
		t.Fatalf("expected already_offline, got %q", resp.Result)
	}
}

func newTestContext(method, path, body string, vars map[string]string) *httppkg.Context {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req = mux.SetURLVars(req, vars)
	w := httptest.NewRecorder()
	return httppkg.NewContext(w, req)
}
