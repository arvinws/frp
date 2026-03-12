package scheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	httppkg "github.com/fatedier/frp/pkg/util/http"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type taskListResponse struct {
	Tasks []taskResponse `json:"tasks"`
}

type logListResponse struct {
	Logs []executionLogResponse `json:"logs"`
}

type taskResponse struct {
	ID                  string         `json:"id"`
	Name                string         `json:"name"`
	Enabled             bool           `json:"enabled"`
	Timezone            string         `json:"timezone"`
	Targets             []Target       `json:"targets"`
	StartRule           *Rule          `json:"startRule,omitempty"`
	StopRule            *Rule          `json:"stopRule,omitempty"`
	Remark              string         `json:"remark,omitempty"`
	CreatedAt           int64          `json:"createdAt"`
	UpdatedAt           int64          `json:"updatedAt"`
	LastExecutionAt     int64          `json:"lastExecutionAt,omitempty"`
	LastExecutionAction TaskAction     `json:"lastExecutionAction,omitempty"`
	LastExecutionResult string         `json:"lastExecutionResult,omitempty"`
	NextStartAt         int64          `json:"nextStartAt,omitempty"`
	NextStopAt          int64          `json:"nextStopAt,omitempty"`
	TargetCount         int            `json:"targetCount"`
	Logs                []executionLogResponse `json:"logs,omitempty"`
}

type executionLogResponse struct {
	ID              string           `json:"id"`
	TaskID          string           `json:"taskID"`
	TaskName        string           `json:"taskName"`
	Action          TaskAction       `json:"action"`
	Trigger         ExecutionTrigger `json:"trigger"`
	ScheduledAt     int64            `json:"scheduledAt"`
	ExecutedAt      int64            `json:"executedAt"`
	TargetProxyName string           `json:"targetProxyName"`
	Result          string           `json:"result"`
	Error           string           `json:"error,omitempty"`
}

func (h *Handler) ListTasks(ctx *httppkg.Context) (any, error) {
	if h.service == nil {
		return nil, fmt.Errorf("schedule service unavailable")
	}
	views := h.service.ListTasks()
	resp := taskListResponse{Tasks: make([]taskResponse, 0, len(views))}
	for _, view := range views {
		resp.Tasks = append(resp.Tasks, toTaskResponse(view))
	}
	return resp, nil
}

func (h *Handler) CreateTask(ctx *httppkg.Context) (any, error) {
	task, err := parseTaskBody(ctx)
	if err != nil {
		return nil, err
	}
	view, err := h.service.CreateTask(task)
	if err != nil {
		return nil, toHTTPError(err)
	}
	return toTaskResponse(view), nil
}

func (h *Handler) GetTask(ctx *httppkg.Context) (any, error) {
	view, ok := h.service.GetTask(strings.TrimSpace(ctx.Param("id")))
	if !ok {
		return nil, httppkg.NewError(http.StatusNotFound, "schedule task not found")
	}
	return toTaskResponse(view), nil
}

func (h *Handler) UpdateTask(ctx *httppkg.Context) (any, error) {
	task, err := parseTaskBody(ctx)
	if err != nil {
		return nil, err
	}
	view, err := h.service.UpdateTask(ctx.Param("id"), task)
	if err != nil {
		return nil, toHTTPError(err)
	}
	return toTaskResponse(view), nil
}

func (h *Handler) DeleteTask(ctx *httppkg.Context) (any, error) {
	if err := h.service.DeleteTask(ctx.Param("id")); err != nil {
		return nil, toHTTPError(err)
	}
	return httppkg.GeneralResponse{Code: http.StatusOK, Msg: "deleted"}, nil
}

func (h *Handler) EnableTask(ctx *httppkg.Context) (any, error) {
	view, err := h.service.SetTaskEnabled(ctx.Param("id"), true)
	if err != nil {
		return nil, toHTTPError(err)
	}
	return toTaskResponse(view), nil
}

func (h *Handler) DisableTask(ctx *httppkg.Context) (any, error) {
	view, err := h.service.SetTaskEnabled(ctx.Param("id"), false)
	if err != nil {
		return nil, toHTTPError(err)
	}
	return toTaskResponse(view), nil
}

func (h *Handler) RunTask(ctx *httppkg.Context) (any, error) {
	action := TaskAction(strings.TrimSpace(ctx.Query("action")))
	result, err := h.service.RunTask(ctx.Param("id"), action)
	if err != nil {
		return nil, toHTTPError(err)
	}
	view, _ := h.service.GetTask(result.TaskID)
	resp := toTaskResponse(view)
	resp.Logs = make([]executionLogResponse, 0, len(result.Logs))
	for _, log := range result.Logs {
		resp.Logs = append(resp.Logs, toLogResponse(log))
	}
	return resp, nil
}

func (h *Handler) ListLogs(ctx *httppkg.Context) (any, error) {
	limit := 50
	if raw := strings.TrimSpace(ctx.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return nil, httppkg.NewError(http.StatusBadRequest, "limit must be a positive integer")
		}
		limit = parsed
	}
	logs := h.service.ListLogs(ctx.Param("id"), limit)
	resp := logListResponse{Logs: make([]executionLogResponse, 0, len(logs))}
	for _, log := range logs {
		resp.Logs = append(resp.Logs, toLogResponse(log))
	}
	return resp, nil
}

func parseTaskBody(ctx *httppkg.Context) (Task, error) {
	body, err := ctx.Body()
	if err != nil {
		return Task{}, httppkg.NewError(http.StatusBadRequest, fmt.Sprintf("read request body error: %v", err))
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return Task{}, httppkg.NewError(http.StatusBadRequest, "request body is required")
	}
	var task Task
	if err := json.Unmarshal(body, &task); err != nil {
		return Task{}, httppkg.NewError(http.StatusBadRequest, "invalid request body")
	}
	return task, nil
}

func toHTTPError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	status := http.StatusInternalServerError
	switch {
	case strings.Contains(message, "not found"):
		status = http.StatusNotFound
	case strings.Contains(message, "required"), strings.Contains(message, "invalid"), strings.Contains(message, "unsupported"), strings.Contains(message, "duplicate"), strings.Contains(message, "configured"):
		status = http.StatusBadRequest
	}
	return httppkg.NewError(status, message)
}

func toTaskResponse(view TaskView) taskResponse {
	return taskResponse{
		ID:                  view.ID,
		Name:                view.Name,
		Enabled:             view.Enabled,
		Timezone:            view.Timezone,
		Targets:             view.Targets,
		StartRule:           view.StartRule,
		StopRule:            view.StopRule,
		Remark:              view.Remark,
		CreatedAt:           toUnix(view.CreatedAt),
		UpdatedAt:           toUnix(view.UpdatedAt),
		LastExecutionAt:     toUnix(view.LastExecutionAt),
		LastExecutionAction: view.LastExecutionAction,
		LastExecutionResult: view.LastExecutionResult,
		NextStartAt:         toUnix(view.NextStartAt),
		NextStopAt:          toUnix(view.NextStopAt),
		TargetCount:         len(view.Targets),
	}
}

func toLogResponse(log ExecutionLog) executionLogResponse {
	return executionLogResponse{
		ID:              log.ID,
		TaskID:          log.TaskID,
		TaskName:        log.TaskName,
		Action:          log.Action,
		Trigger:         log.Trigger,
		ScheduledAt:     toUnix(log.ScheduledAt),
		ExecutedAt:      toUnix(log.ExecutedAt),
		TargetProxyName: log.TargetProxyName,
		Result:          log.Result,
		Error:           log.Error,
	}
}

func toUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}
