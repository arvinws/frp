package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtectWritesBlocksCrossSitePost(t *testing.T) {
	handler := ProtectWrites(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	blocked := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:7500/api/admin/server/restart", nil)
	blocked.Host = "127.0.0.1:7500"
	blocked.Header.Set("Origin", "http://evil.example")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, blocked)
	require.Equal(t, http.StatusForbidden, recorder.Code)

	allowed := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:7500/api/admin/server/restart", nil)
	allowed.Host = "127.0.0.1:7500"
	allowed.Header.Set("Origin", "http://127.0.0.1:7500")
	allowed.Header.Set("Sec-Fetch-Site", "same-origin")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, allowed)
	require.Equal(t, http.StatusNoContent, recorder.Code)

	script := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:7500/api/admin/server/restart", nil)
	script.Host = "127.0.0.1:7500"
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, script)
	require.Equal(t, http.StatusNoContent, recorder.Code)
}
