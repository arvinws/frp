package validation

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

func TestPublicBindRequiresToken(t *testing.T) {
	cfg := &v1.ServerConfig{BindAddr: "0.0.0.0", Auth: v1.AuthServerConfig{Method: v1.AuthMethodToken}}
	require.NoError(t, cfg.Complete())
	_, err := NewConfigValidator(nil).ValidateServerConfig(cfg)
	require.ErrorContains(t, err, "auth.token is required")

	cfg.BindAddr = "127.0.0.1"
	require.NoError(t, cfg.Complete())
	_, err = NewConfigValidator(nil).ValidateServerConfig(cfg)
	require.NoError(t, err)

	cfg.BindAddr = "0.0.0.0"
	cfg.Auth.Token = "not-empty"
	require.NoError(t, cfg.Complete())
	_, err = NewConfigValidator(nil).ValidateServerConfig(cfg)
	require.NoError(t, err)
}

func TestPublicDashboardRejectsEmptyAndDefaultCredentials(t *testing.T) {
	for _, tc := range []struct {
		name    string
		addr    string
		user    string
		pass    string
		wantErr string
	}{
		{name: "public empty", addr: "0.0.0.0", wantErr: "webServer.password is required"},
		{name: "public admin", addr: "0.0.0.0", user: "admin", pass: "admin", wantErr: "default admin/admin"},
		{name: "loopback admin", addr: "127.0.0.1", user: "admin", pass: "admin"},
		{name: "public custom", addr: "0.0.0.0", user: "ops", pass: "not-admin"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &v1.ServerConfig{
				BindAddr: "127.0.0.1",
				Auth:     v1.AuthServerConfig{Method: v1.AuthMethodToken},
				WebServer: v1.WebServerConfig{
					Addr: tc.addr, Port: 7500, User: tc.user, Password: tc.pass,
				},
			}
			require.NoError(t, cfg.Complete())
			_, err := NewConfigValidator(nil).ValidateServerConfig(cfg)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
