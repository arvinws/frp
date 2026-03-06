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

package adminapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fatedier/frp/pkg/ext/audit"
	"github.com/fatedier/frp/pkg/ext/banlist"
	"github.com/fatedier/frp/pkg/ext/clientmgr"
	httppkg "github.com/fatedier/frp/pkg/util/http"
)

type Handler struct {
	banStore       banlist.Store
	sessionManager *clientmgr.Manager
	auditRecorder  audit.Recorder
}

func NewHandler(banStore banlist.Store, sessionManager *clientmgr.Manager, auditRecorder audit.Recorder) *Handler {
	if auditRecorder == nil {
		auditRecorder = noopRecorder{}
	}
	return &Handler{
		banStore:       banStore,
		sessionManager: sessionManager,
		auditRecorder:  auditRecorder,
	}
}

type ActionRequest struct {
	Reason   string `json:"reason"`
	Operator string `json:"operator"`
}

type RuleActionResponse struct {
	ClientID  string `json:"clientID"`
	Status    string `json:"status"`
	Changed   bool   `json:"changed"`
	Reason    string `json:"reason,omitempty"`
	Operator  string `json:"operator,omitempty"`
	UpdatedAt int64  `json:"updatedAt,omitempty"`
	Result    string `json:"result"`
}

type DisconnectResponse struct {
	RunID    string `json:"runID"`
	ClientID string `json:"clientID,omitempty"`
	Result   string `json:"result"`
}

type DisableAndDisconnectResponse struct {
	ClientID         string `json:"clientID"`
	Status           string `json:"status"`
	RuleChanged      bool   `json:"ruleChanged"`
	Reason           string `json:"reason,omitempty"`
	Operator         string `json:"operator,omitempty"`
	UpdatedAt        int64  `json:"updatedAt,omitempty"`
	RunID            string `json:"runID,omitempty"`
	DisconnectResult string `json:"disconnectResult"`
}

// DisableClient handles POST /api/admin/clients/{clientID}/disable.
func (h *Handler) DisableClient(ctx *httppkg.Context) (any, error) {
	if h.banStore == nil {
		return nil, fmt.Errorf("banlist store unavailable")
	}

	clientID := strings.TrimSpace(ctx.Param("clientID"))
	if clientID == "" {
		return nil, httppkg.NewError(http.StatusBadRequest, "missing clientID")
	}

	req, err := parseActionRequest(ctx)
	if err != nil {
		return nil, err
	}

	record, changed := h.banStore.Disable(clientID, req.Reason, req.Operator)
	result := "disabled"
	if !changed {
		result = "already_disabled"
	}
	h.recordAudit("disable", clientID, "", req.Operator, result, req.Reason)

	return RuleActionResponse{
		ClientID:  clientID,
		Status:    record.Status,
		Changed:   changed,
		Reason:    record.Reason,
		Operator:  req.Operator,
		UpdatedAt: toUnix(record.UpdatedAt),
		Result:    result,
	}, nil
}

// EnableClient handles POST /api/admin/clients/{clientID}/enable.
func (h *Handler) EnableClient(ctx *httppkg.Context) (any, error) {
	if h.banStore == nil {
		return nil, fmt.Errorf("banlist store unavailable")
	}

	clientID := strings.TrimSpace(ctx.Param("clientID"))
	if clientID == "" {
		return nil, httppkg.NewError(http.StatusBadRequest, "missing clientID")
	}

	req, err := parseActionRequest(ctx)
	if err != nil {
		return nil, err
	}

	record, changed := h.banStore.Enable(clientID, req.Operator)
	result := "enabled"
	if !changed {
		result = "already_enabled"
	}
	h.recordAudit("enable", clientID, "", req.Operator, result, "")

	return RuleActionResponse{
		ClientID:  clientID,
		Status:    record.Status,
		Changed:   changed,
		Operator:  req.Operator,
		UpdatedAt: toUnix(record.UpdatedAt),
		Result:    result,
	}, nil
}

// DisconnectSession handles POST /api/admin/sessions/{runID}/disconnect.
func (h *Handler) DisconnectSession(ctx *httppkg.Context) (any, error) {
	if h.sessionManager == nil {
		return nil, fmt.Errorf("session manager unavailable")
	}

	runID := strings.TrimSpace(ctx.Param("runID"))
	if runID == "" {
		return nil, httppkg.NewError(http.StatusBadRequest, "missing runID")
	}

	req, err := parseActionRequest(ctx)
	if err != nil {
		return nil, err
	}

	session, found := h.sessionManager.GetByRunID(runID)
	if !found {
		result := "already_offline"
		h.recordAudit("disconnect", "", runID, req.Operator, result, req.Reason)
		return DisconnectResponse{
			RunID:  runID,
			Result: result,
		}, nil
	}

	result := "disconnected"
	if ok := h.sessionManager.DisconnectByRunID(runID); !ok {
		result = "already_offline"
	}
	h.recordAudit("disconnect", session.ClientID, runID, req.Operator, result, req.Reason)

	return DisconnectResponse{
		RunID:    runID,
		ClientID: session.ClientID,
		Result:   result,
	}, nil
}

// DisableAndDisconnect handles POST /api/admin/clients/{clientID}/disable-and-disconnect.
func (h *Handler) DisableAndDisconnect(ctx *httppkg.Context) (any, error) {
	if h.banStore == nil {
		return nil, fmt.Errorf("banlist store unavailable")
	}
	if h.sessionManager == nil {
		return nil, fmt.Errorf("session manager unavailable")
	}

	clientID := strings.TrimSpace(ctx.Param("clientID"))
	if clientID == "" {
		return nil, httppkg.NewError(http.StatusBadRequest, "missing clientID")
	}

	req, err := parseActionRequest(ctx)
	if err != nil {
		return nil, err
	}

	record, changed := h.banStore.Disable(clientID, req.Reason, req.Operator)

	runID := ""
	disconnectResult := "already_offline"
	if session, ok := h.sessionManager.GetByClientID(clientID); ok {
		runID = session.RunID
		disconnectResult = "disconnected"
		if disconnectOK := h.sessionManager.DisconnectByRunID(runID); !disconnectOK {
			disconnectResult = "already_offline"
		}
	}

	h.recordAudit("disable_and_disconnect", clientID, runID, req.Operator, disconnectResult, req.Reason)

	return DisableAndDisconnectResponse{
		ClientID:         clientID,
		Status:           record.Status,
		RuleChanged:      changed,
		Reason:           record.Reason,
		Operator:         req.Operator,
		UpdatedAt:        toUnix(record.UpdatedAt),
		RunID:            runID,
		DisconnectResult: disconnectResult,
	}, nil
}

func parseActionRequest(ctx *httppkg.Context) (ActionRequest, error) {
	req := ActionRequest{}
	body, err := ctx.Body()
	if err != nil {
		return req, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return req, nil
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return req, httppkg.NewError(http.StatusBadRequest, "invalid request body")
	}
	req.Reason = strings.TrimSpace(req.Reason)
	req.Operator = strings.TrimSpace(req.Operator)
	return req, nil
}

func (h *Handler) recordAudit(action, targetClientID, targetRunID, operator, result, detail string) {
	h.auditRecorder.Record(audit.Event{
		Action:         action,
		TargetClientID: targetClientID,
		TargetRunID:    targetRunID,
		Operator:       operator,
		Result:         result,
		Detail:         detail,
	})
}

func toUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

type noopRecorder struct{}

func (noopRecorder) Record(audit.Event) {}
