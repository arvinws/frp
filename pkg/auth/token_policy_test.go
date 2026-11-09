package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/util"
)

func TestTokenLoginWindowAndReplay(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	verifier := newTokenAuth(nil, "secret", 10*time.Minute, func() time.Time { return now })
	fresh := &msg.Login{
		Hostname:  "host-a",
		Timestamp: now.Unix(),
	}
	fresh.PrivilegeKey = util.GetAuthKey("secret", fresh.Timestamp)
	require.NoError(t, verifier.VerifyLogin(fresh))
	for range 3 {
		require.NoError(t, verifier.VerifyLogin(fresh))
	}
	require.ErrorContains(t, verifier.VerifyLogin(fresh), "replayed login")

	stale := &msg.Login{Hostname: "host-a", Timestamp: 1}
	stale.PrivilegeKey = util.GetAuthKey("secret", stale.Timestamp)
	require.ErrorContains(t, verifier.VerifyLogin(stale), "outside the allowed window")

	other := &msg.Login{Hostname: "host-b", Timestamp: now.Unix()}
	other.PrivilegeKey = util.GetAuthKey("secret", other.Timestamp)
	require.NoError(t, verifier.VerifyLogin(other))
}
