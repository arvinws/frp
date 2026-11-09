// Copyright 2020 guylewin, guy@lewin.co.il
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth

import (
	"fmt"
	"slices"
	"sync"
	"time"

	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/msg"
	"github.com/fatedier/frp/pkg/util/util"
)

const maxIdenticalLoginUses = 4

type TokenAuthSetterVerifier struct {
	additionalAuthScopes []v1.AuthScope
	token                string
	maxTimeDiff          time.Duration
	now                  func() time.Time

	mu   sync.Mutex
	seen map[string]loginUse
}

type loginUse struct {
	count   int
	expires time.Time
}

func NewTokenAuth(additionalAuthScopes []v1.AuthScope, token string) *TokenAuthSetterVerifier {
	return newTokenAuth(additionalAuthScopes, token, 0, time.Now)
}

// NewTokenAuthWithPolicy enables the login timestamp window. A zero window
// keeps the historical behavior used by tests and by the client setter.
func NewTokenAuthWithPolicy(additionalAuthScopes []v1.AuthScope, token string, maxTimeDiff time.Duration) *TokenAuthSetterVerifier {
	return newTokenAuth(additionalAuthScopes, token, maxTimeDiff, time.Now)
}

func newTokenAuth(additionalAuthScopes []v1.AuthScope, token string, maxTimeDiff time.Duration, now func() time.Time) *TokenAuthSetterVerifier {
	if now == nil {
		now = time.Now
	}
	return &TokenAuthSetterVerifier{
		additionalAuthScopes: additionalAuthScopes,
		token:                token,
		maxTimeDiff:          maxTimeDiff,
		now:                  now,
		seen:                 make(map[string]loginUse),
	}
}

func (auth *TokenAuthSetterVerifier) SetLogin(loginMsg *msg.Login) error {
	loginMsg.PrivilegeKey = util.GetAuthKey(auth.token, loginMsg.Timestamp)
	return nil
}

func (auth *TokenAuthSetterVerifier) SetPing(pingMsg *msg.Ping) error {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeHeartBeats) {
		return nil
	}

	pingMsg.Timestamp = time.Now().Unix()
	pingMsg.PrivilegeKey = util.GetAuthKey(auth.token, pingMsg.Timestamp)
	return nil
}

func (auth *TokenAuthSetterVerifier) SetNewWorkConn(newWorkConnMsg *msg.NewWorkConn) error {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}

	newWorkConnMsg.Timestamp = time.Now().Unix()
	newWorkConnMsg.PrivilegeKey = util.GetAuthKey(auth.token, newWorkConnMsg.Timestamp)
	return nil
}

func (auth *TokenAuthSetterVerifier) VerifyLogin(m *msg.Login) error {
	if !util.ConstantTimeEqString(util.GetAuthKey(auth.token, m.Timestamp), m.PrivilegeKey) {
		return fmt.Errorf("token in login doesn't match token from configuration")
	}
	if auth.maxTimeDiff <= 0 {
		return nil
	}
	now := auth.now()
	skew := time.Duration(m.Timestamp-now.Unix()) * time.Second
	if skew < 0 {
		skew = -skew
	}
	if skew > auth.maxTimeDiff {
		return fmt.Errorf("login timestamp is outside the allowed window")
	}
	if err := auth.rememberLogin(m, now); err != nil {
		return err
	}
	return nil
}

func (auth *TokenAuthSetterVerifier) rememberLogin(m *msg.Login, now time.Time) error {
	key := m.PrivilegeKey + "\n" + m.Hostname + "\n" + m.User + "\n" + m.ClientID + "\n" + m.RunID + "\n" + m.Os + "\n" + m.Arch
	auth.mu.Lock()
	defer auth.mu.Unlock()
	for id, entry := range auth.seen {
		if !entry.expires.After(now) {
			delete(auth.seen, id)
		}
	}
	entry := auth.seen[key]
	if !entry.expires.After(now) {
		entry = loginUse{expires: now.Add(auth.maxTimeDiff)}
	}
	if entry.count >= maxIdenticalLoginUses {
		return fmt.Errorf("replayed login")
	}
	entry.count++
	auth.seen[key] = entry
	return nil
}

func (auth *TokenAuthSetterVerifier) VerifyPing(m *msg.Ping) error {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeHeartBeats) {
		return nil
	}

	if !util.ConstantTimeEqString(util.GetAuthKey(auth.token, m.Timestamp), m.PrivilegeKey) {
		return fmt.Errorf("token in heartbeat doesn't match token from configuration")
	}
	return nil
}

func (auth *TokenAuthSetterVerifier) VerifyNewWorkConn(m *msg.NewWorkConn) error {
	if !slices.Contains(auth.additionalAuthScopes, v1.AuthScopeNewWorkConns) {
		return nil
	}

	if !util.ConstantTimeEqString(util.GetAuthKey(auth.token, m.Timestamp), m.PrivilegeKey) {
		return fmt.Errorf("token in NewWorkConn doesn't match token from configuration")
	}
	return nil
}
