package server

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/pkg/util/version"
)

const (
	loginFailureLimit  = 10
	loginFailureWindow = time.Minute
)

// errAuthorization is returned to the connection handler so the client receives
// a generic failure. The detailed reason is logged on the server only.
var errAuthorization = errors.New("authorization failed")

func loginFailureResponse(err error, detailed bool) *msg.LoginResp {
	if errors.Is(err, errAuthorization) {
		return &msg.LoginResp{Error: "authorization failed"}
	}
	return &msg.LoginResp{
		Version: version.Full(),
		Error:   util.GenerateResponseErrorString("register control error", err, detailed),
	}
}

func clientIP(conn net.Conn) string {
	if conn == nil || conn.RemoteAddr() == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		return conn.RemoteAddr().String()
	}
	return host
}

type loginFailureLimiter struct {
	mu       sync.Mutex
	failures map[string]loginFailureState
	now      func() time.Time
	limit    int
	window   time.Duration
}

type loginFailureState struct {
	count        int
	windowStart  time.Time
	blockedUntil time.Time
}

func newLoginFailureLimiter(now func() time.Time) *loginFailureLimiter {
	if now == nil {
		now = time.Now
	}
	return &loginFailureLimiter{
		failures: make(map[string]loginFailureState),
		now:      now,
		limit:    loginFailureLimit,
		window:   loginFailureWindow,
	}
}

func (svr *Service) loginGuard() *loginFailureLimiter {
	svr.loginFailOnce.Do(func() {
		if svr.loginFailures == nil {
			svr.loginFailures = newLoginFailureLimiter(time.Now)
		}
	})
	return svr.loginFailures
}

func (l *loginFailureLimiter) blocked(ip string) bool {
	if l == nil || ip == "" {
		return false
	}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	state := l.failures[ip]
	return now.Before(state.blockedUntil)
}

func (l *loginFailureLimiter) fail(ip string) {
	if l == nil || ip == "" {
		return
	}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	state := l.failures[ip]
	if state.windowStart.IsZero() || now.Sub(state.windowStart) >= l.window {
		state = loginFailureState{windowStart: now}
	}
	state.count++
	if state.count >= l.limit {
		state.blockedUntil = now.Add(l.window)
		state.count = 0
		state.windowStart = now
	}
	l.failures[ip] = state
}

func (l *loginFailureLimiter) success(ip string) {
	if l == nil || ip == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, ip)
}
