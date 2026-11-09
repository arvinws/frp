package validation

import (
	"fmt"
	"net"
	"strings"

	v1 "github.com/fatedier/frp/pkg/config/v1"
)

// IsPublicBindAddr reports whether addr accepts connections from outside the
// local machine. Empty addresses are not treated as public so callers that
// have not run Complete yet keep their existing behavior. 0.0.0.0 and :: are
// public because they listen on every interface.
func IsPublicBindAddr(addr string) bool {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return false
	}
	addr = strings.Trim(addr, "[]")
	if strings.EqualFold(addr, "localhost") {
		return false
	}
	ip := net.ParseIP(addr)
	if ip == nil {
		return true
	}
	return !ip.IsLoopback()
}

func validatePublicManagementAuth(kind, addr, user, password string, port int) error {
	if port <= 0 || !IsPublicBindAddr(addr) {
		return nil
	}
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("%s.password is required when %s.addr %q is not a loopback address", kind, kind, addr)
	}
	if strings.EqualFold(strings.TrimSpace(user), "admin") && password == "admin" {
		return fmt.Errorf("%s rejects the default admin/admin credentials on non-loopback address %q; set a different password", kind, addr)
	}
	return nil
}

func validateTokenForPublicBind(c *v1.ServerConfig) error {
	if c.Auth.Method != v1.AuthMethodToken || !IsPublicBindAddr(c.BindAddr) {
		return nil
	}
	if strings.TrimSpace(c.Auth.Token) == "" && c.Auth.TokenSource == nil {
		return fmt.Errorf("auth.token is required when bindAddr %q is not a loopback address; set auth.token or bindAddr = \"127.0.0.1\"", c.BindAddr)
	}
	return nil
}

func validateTokenTimeDiff(c *v1.ServerConfig) error {
	if c.Auth.Method != v1.AuthMethodToken || c.Auth.MaxTimeDiffSeconds == 0 {
		return nil
	}
	if c.Auth.MaxTimeDiffSeconds < 30 || c.Auth.MaxTimeDiffSeconds > 86400 {
		return fmt.Errorf("auth.maxTimeDiffSeconds must be between 30 and 86400")
	}
	return nil
}
