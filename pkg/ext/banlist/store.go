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

package banlist

import (
	"sync"
	"time"
)

const (
	StatusEnabled  = "enabled"
	StatusDisabled = "disabled"
)

// ClientBanRecord is the external representation of a client's governance status.
type ClientBanRecord struct {
	ClientID  string    `json:"clientID"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
	Operator  string    `json:"operator,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

// Store defines banlist operations.
// The default state for an unknown clientID is "enabled".
type Store interface {
	Disable(clientID, reason, operator string) (ClientBanRecord, bool)
	Enable(clientID, operator string) (ClientBanRecord, bool)
	Get(clientID string) ClientBanRecord
	IsDisabled(clientID string) (bool, ClientBanRecord)
}

// MemoryStore keeps disabled records in memory.
type MemoryStore struct {
	mu       sync.RWMutex
	disabled map[string]banEntry
}

type banEntry struct {
	reason    string
	operator  string
	updatedAt time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		disabled: make(map[string]banEntry),
	}
}

func (s *MemoryStore) Disable(clientID, reason, operator string) (ClientBanRecord, bool) {
	if clientID == "" {
		return ClientBanRecord{Status: StatusEnabled}, false
	}

	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	old, exists := s.disabled[clientID]
	changed := !exists || old.reason != reason || old.operator != operator
	s.disabled[clientID] = banEntry{
		reason:    reason,
		operator:  operator,
		updatedAt: now,
	}

	return ClientBanRecord{
		ClientID:  clientID,
		Status:    StatusDisabled,
		Reason:    reason,
		Operator:  operator,
		UpdatedAt: now,
	}, changed
}

func (s *MemoryStore) Enable(clientID, operator string) (ClientBanRecord, bool) {
	if clientID == "" {
		return ClientBanRecord{Status: StatusEnabled}, false
	}

	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.disabled[clientID]
	if exists {
		delete(s.disabled, clientID)
	}

	record := ClientBanRecord{
		ClientID: clientID,
		Status:   StatusEnabled,
		Operator: operator,
	}
	if exists {
		record.UpdatedAt = now
	}
	return record, exists
}

func (s *MemoryStore) Get(clientID string) ClientBanRecord {
	if clientID == "" {
		return ClientBanRecord{Status: StatusEnabled}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.disabled[clientID]
	if !ok {
		return ClientBanRecord{
			ClientID: clientID,
			Status:   StatusEnabled,
		}
	}

	return ClientBanRecord{
		ClientID:  clientID,
		Status:    StatusDisabled,
		Reason:    entry.reason,
		Operator:  entry.operator,
		UpdatedAt: entry.updatedAt,
	}
}

func (s *MemoryStore) IsDisabled(clientID string) (bool, ClientBanRecord) {
	record := s.Get(clientID)
	return record.Status == StatusDisabled, record
}

var _ Store = (*MemoryStore)(nil)
