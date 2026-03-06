// Copyright 2026 The frp Authors
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

package clientmgr

import (
	"sync"
	"time"
)

// SessionController is the minimal controller capability needed for immediate disconnect.
type SessionController interface {
	Close() error
}

// OnlineSession stores runtime metadata for a connected client session.
type OnlineSession struct {
	RunID      string    `json:"runID"`
	ClientID   string    `json:"clientID"`
	User       string    `json:"user,omitempty"`
	ConnectedAt time.Time `json:"connectedAt"`
	RemoteAddr string    `json:"remoteAddr,omitempty"`

	controller SessionController `json:"-"`
}

// Manager indexes online sessions by runID and clientID.
type Manager struct {
	mu             sync.RWMutex
	sessionsByRunID map[string]*OnlineSession
	runIDByClientID map[string]string
}

func NewManager() *Manager {
	return &Manager{
		sessionsByRunID: make(map[string]*OnlineSession),
		runIDByClientID: make(map[string]string),
	}
}

func (m *Manager) Register(runID, clientID, user, remoteAddr string, controller SessionController) {
	if runID == "" {
		return
	}

	session := &OnlineSession{
		RunID:       runID,
		ClientID:    clientID,
		User:        user,
		ConnectedAt: time.Now(),
		RemoteAddr:  remoteAddr,
		controller:  controller,
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.sessionsByRunID[runID]; ok && existing.ClientID != "" {
		if indexedRunID, exists := m.runIDByClientID[existing.ClientID]; exists && indexedRunID == runID {
			delete(m.runIDByClientID, existing.ClientID)
		}
	}

	m.sessionsByRunID[runID] = session
	if clientID != "" {
		m.runIDByClientID[clientID] = runID
	}
}

func (m *Manager) Unregister(runID string) {
	if runID == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessionsByRunID[runID]
	if !ok {
		return
	}
	delete(m.sessionsByRunID, runID)

	if session.ClientID == "" {
		return
	}
	if indexedRunID, exists := m.runIDByClientID[session.ClientID]; exists && indexedRunID == runID {
		delete(m.runIDByClientID, session.ClientID)
	}
}

func (m *Manager) GetByRunID(runID string) (OnlineSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessionsByRunID[runID]
	if !ok {
		return OnlineSession{}, false
	}
	return session.snapshot(), true
}

func (m *Manager) GetByClientID(clientID string) (OnlineSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	runID, ok := m.runIDByClientID[clientID]
	if !ok {
		return OnlineSession{}, false
	}
	session, ok := m.sessionsByRunID[runID]
	if !ok {
		return OnlineSession{}, false
	}
	return session.snapshot(), true
}

// DisconnectByRunID closes the current control connection of a session.
func (m *Manager) DisconnectByRunID(runID string) bool {
	m.mu.RLock()
	session, ok := m.sessionsByRunID[runID]
	var controller SessionController
	if ok {
		controller = session.controller
	}
	m.mu.RUnlock()

	if !ok || controller == nil {
		return false
	}

	_ = controller.Close()
	return true
}

func (s *OnlineSession) snapshot() OnlineSession {
	if s == nil {
		return OnlineSession{}
	}
	copy := *s
	copy.controller = nil
	return copy
}
