package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoginFailureResponseHidesVersion(t *testing.T) {
	resp := loginFailureResponse(errAuthorization, true)
	require.Empty(t, resp.Version)
	require.Equal(t, "authorization failed", resp.Error)

	resp = loginFailureResponse(errTest("port already used"), true)
	require.NotEmpty(t, resp.Version)
	require.Contains(t, resp.Error, "port already used")
}

type errTest string

func (e errTest) Error() string { return string(e) }

func TestLoginFailureLimiterBlocksAfterLimit(t *testing.T) {
	now := time.Unix(1_000, 0)
	limiter := newLoginFailureLimiter(func() time.Time { return now })
	limiter.limit = 3
	for range 2 {
		limiter.fail("203.0.113.5")
		require.False(t, limiter.blocked("203.0.113.5"))
	}
	limiter.fail("203.0.113.5")
	require.True(t, limiter.blocked("203.0.113.5"))
	limiter.success("203.0.113.5")
	require.False(t, limiter.blocked("203.0.113.5"))
}
