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

package audit

import (
	"sync"
	"time"

	"github.com/fatedier/frp/pkg/util/log"
)

// Event represents one governance management action.
type Event struct {
	Action         string    `json:"action"`
	TargetClientID string    `json:"targetClientID,omitempty"`
	TargetRunID    string    `json:"targetRunID,omitempty"`
	Operator       string    `json:"operator,omitempty"`
	Result         string    `json:"result"`
	Detail         string    `json:"detail,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// Recorder is the minimum contract for writing audit events.
type Recorder interface {
	Record(event Event)
}

// MemoryRecorder stores audit events in-memory and logs every event.
type MemoryRecorder struct {
	mu     sync.RWMutex
	events []Event
}

func NewMemoryRecorder() *MemoryRecorder {
	return &MemoryRecorder{
		events: make([]Event, 0, 128),
	}
}

func (r *MemoryRecorder) Record(event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	r.mu.Lock()
	r.events = append(r.events, event)
	r.mu.Unlock()

	log.Infof(
		"admin audit action=%s target_client_id=%s target_run_id=%s operator=%s result=%s detail=%s",
		event.Action,
		event.TargetClientID,
		event.TargetRunID,
		event.Operator,
		event.Result,
		event.Detail,
	)
}

func (r *MemoryRecorder) List() []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Event, len(r.events))
	copy(out, r.events)
	return out
}

var _ Recorder = (*MemoryRecorder)(nil)
