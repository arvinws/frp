package scheduler

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fatedier/frp/pkg/ext/governance"
	"github.com/fatedier/frp/pkg/util/util"
)

const (
	defaultScanInterval = 30 * time.Second
	defaultDueWindow    = 2 * time.Minute
)

type ProxyGovernance interface {
	DisableProxy(proxyName, source, reason, operator string) (governance.ProxyActionResult, error)
	EnableProxy(proxyName, source string) (governance.ProxyActionResult, error)
}

type Service struct {
	store          Store
	proxyGovernance ProxyGovernance
	scanInterval   time.Duration
	dueWindow      time.Duration
	nowFn          func() time.Time
	runMu          sync.Mutex
}

func NewService(store Store, proxyGovernance ProxyGovernance) *Service {
	return &Service{
		store:           store,
		proxyGovernance: proxyGovernance,
		scanInterval:    defaultScanInterval,
		dueWindow:       defaultDueWindow,
		nowFn:           time.Now,
	}
}

func (s *Service) Start(ctx context.Context) {
	if s == nil || s.store == nil {
		return
	}
	go s.loop(ctx)
}

func (s *Service) loop(ctx context.Context) {
	ticker := time.NewTicker(s.scanInterval)
	defer ticker.Stop()
	s.runPending()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runPending()
		}
	}
}

func (s *Service) ListTasks() []TaskView {
	tasks := s.store.ListTasks()
	views := make([]TaskView, 0, len(tasks))
	now := s.nowFn()
	for _, task := range tasks {
		view := TaskView{Task: task}
		view.NextStartAt = s.nextScheduledAt(task, TaskActionStart, now)
		view.NextStopAt = s.nextScheduledAt(task, TaskActionStop, now)
		views = append(views, view)
	}
	slices.SortFunc(views, func(a, b TaskView) int {
		if v := cmp.Compare(a.Name, b.Name); v != 0 {
			return v
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return views
}

func (s *Service) GetTask(id string) (TaskView, bool) {
	task, ok := s.store.GetTask(strings.TrimSpace(id))
	if !ok {
		return TaskView{}, false
	}
	now := s.nowFn()
	return TaskView{
		Task:        task,
		NextStartAt: s.nextScheduledAt(task, TaskActionStart, now),
		NextStopAt:  s.nextScheduledAt(task, TaskActionStop, now),
	}, true
}

func (s *Service) CreateTask(input Task) (TaskView, error) {
	if s.store == nil {
		return TaskView{}, fmt.Errorf("schedule store unavailable")
	}
	s.runMu.Lock()
	defer s.runMu.Unlock()
	now := s.nowFn()
	task := Task{
		ID:        strings.TrimSpace(input.ID),
		Name:      strings.TrimSpace(input.Name),
		Enabled:   input.Enabled,
		Timezone:  strings.TrimSpace(input.Timezone),
		Targets:   normalizeTargets(input.Targets),
		StartRule: normalizeRule(input.StartRule),
		StopRule:  normalizeRule(input.StopRule),
		Remark:    strings.TrimSpace(input.Remark),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if task.ID == "" {
		id, err := util.RandID()
		if err != nil {
			return TaskView{}, err
		}
		task.ID = id
	}
	if err := validateTask(task); err != nil {
		return TaskView{}, err
	}
	if err := s.store.SaveTask(task); err != nil {
		return TaskView{}, err
	}
	view, _ := s.GetTask(task.ID)
	return view, nil
}

func (s *Service) UpdateTask(id string, input Task) (TaskView, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	id = strings.TrimSpace(id)
	if id == "" {
		return TaskView{}, fmt.Errorf("task id is required")
	}
	current, ok := s.store.GetTask(id)
	if !ok {
		return TaskView{}, fmt.Errorf("schedule task not found")
	}
	if current.LastExecutionAction == TaskActionStop {
		_ = s.releaseTaskSources(current)
		current.LastExecutionAction = ""
		current.LastExecutionResult = ""
		current.LastStopScheduledAt = time.Time{}
	}

	updated := current
	updated.Name = strings.TrimSpace(input.Name)
	updated.Enabled = input.Enabled
	updated.Timezone = strings.TrimSpace(input.Timezone)
	updated.Targets = normalizeTargets(input.Targets)
	updated.StartRule = normalizeRule(input.StartRule)
	updated.StopRule = normalizeRule(input.StopRule)
	updated.Remark = strings.TrimSpace(input.Remark)
	updated.UpdatedAt = s.nowFn()
	if err := validateTask(updated); err != nil {
		return TaskView{}, err
	}
	if err := s.store.SaveTask(updated); err != nil {
		return TaskView{}, err
	}
	view, _ := s.GetTask(id)
	return view, nil
}

func (s *Service) DeleteTask(id string) error {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("task id is required")
	}
	task, ok := s.store.GetTask(id)
	if !ok {
		return fmt.Errorf("schedule task not found")
	}
	if task.LastExecutionAction == TaskActionStop {
		if err := s.releaseTaskSources(task); err != nil {
			return err
		}
	}
	deleted, err := s.store.DeleteTask(id)
	if err != nil {
		return err
	}
	if !deleted {
		return fmt.Errorf("schedule task not found")
	}
	return nil
}

func (s *Service) SetTaskEnabled(id string, enabled bool) (TaskView, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	id = strings.TrimSpace(id)
	if id == "" {
		return TaskView{}, fmt.Errorf("task id is required")
	}
	task, ok := s.store.GetTask(id)
	if !ok {
		return TaskView{}, fmt.Errorf("schedule task not found")
	}
	if task.Enabled == enabled {
		view, _ := s.GetTask(id)
		return view, nil
	}
	if !enabled && task.LastExecutionAction == TaskActionStop {
		if err := s.releaseTaskSources(task); err != nil {
			return TaskView{}, err
		}
		task.LastExecutionAction = ""
		task.LastExecutionResult = ""
		task.LastStopScheduledAt = time.Time{}
	}
	task.Enabled = enabled
	task.UpdatedAt = s.nowFn()
	if err := s.store.SaveTask(task); err != nil {
		return TaskView{}, err
	}
	view, _ := s.GetTask(id)
	return view, nil
}

func (s *Service) RunTask(id string, action TaskAction) (RunResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return RunResult{}, fmt.Errorf("task id is required")
	}
	task, ok := s.store.GetTask(id)
	if !ok {
		return RunResult{}, fmt.Errorf("schedule task not found")
	}
	s.runMu.Lock()
	defer s.runMu.Unlock()
	return s.executeTask(task, action, s.nowFn(), ExecutionTriggerManual)
}

func (s *Service) ListLogs(taskID string, limit int) []ExecutionLog {
	return s.store.ListLogs(strings.TrimSpace(taskID), limit)
}

func (s *Service) runPending() {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	now := s.nowFn()
	for _, task := range s.store.ListTasks() {
		if !task.Enabled {
			continue
		}
		if scheduledAt, due := s.scheduledDueAt(task, TaskActionStart, now); due && !sameScheduledAt(task.LastStartScheduledAt, scheduledAt) {
			_, _ = s.executeTask(task, TaskActionStart, scheduledAt, ExecutionTriggerSchedule)
		}
		if scheduledAt, due := s.scheduledDueAt(task, TaskActionStop, now); due && !sameScheduledAt(task.LastStopScheduledAt, scheduledAt) {
			_, _ = s.executeTask(task, TaskActionStop, scheduledAt, ExecutionTriggerSchedule)
		}
	}
}

func (s *Service) executeTask(task Task, action TaskAction, scheduledAt time.Time, trigger ExecutionTrigger) (RunResult, error) {
	if s.proxyGovernance == nil {
		return RunResult{}, fmt.Errorf("proxy governance unavailable")
	}
	if action != TaskActionStart && action != TaskActionStop {
		return RunResult{}, fmt.Errorf("unsupported task action %q", action)
	}
	executedAt := s.nowFn()
	source := governance.ScheduleProxySource(task.ID)
	logs := make([]ExecutionLog, 0, len(task.Targets))
	result := "success"
	for _, target := range task.Targets {
		logEntry := ExecutionLog{
			ID:              mustRandID(),
			TaskID:          task.ID,
			TaskName:        task.Name,
			Action:          action,
			Trigger:         trigger,
			ScheduledAt:     scheduledAt,
			ExecutedAt:      executedAt,
			TargetProxyName: target.ProxyName,
		}
		var (
			proxyResult governance.ProxyActionResult
			err         error
		)
		switch action {
		case TaskActionStart:
			proxyResult, err = s.proxyGovernance.EnableProxy(target.ProxyName, source)
		case TaskActionStop:
			proxyResult, err = s.proxyGovernance.DisableProxy(target.ProxyName, source, fmt.Sprintf("scheduled by task %s", task.Name), "scheduler")
		}
		if err != nil {
			logEntry.Result = "failed"
			logEntry.Error = err.Error()
			if result == "success" {
				result = "partial_success"
			}
		} else {
			logEntry.Result = proxyResult.Result
		}
		logs = append(logs, logEntry)
	}
	if len(logs) == 0 {
		result = "skipped"
	}
	if result == "partial_success" {
		failed := 0
		for _, logEntry := range logs {
			if logEntry.Error != "" {
				failed++
			}
		}
		if failed == len(logs) {
			result = "failed"
		}
	}

	task.LastExecutionAt = executedAt
	task.LastExecutionAction = action
	task.LastExecutionResult = result
	task.UpdatedAt = executedAt
	if action == TaskActionStart {
		task.LastStartScheduledAt = scheduledAt
	} else {
		task.LastStopScheduledAt = scheduledAt
	}
	if trigger == ExecutionTriggerManual {
		s.applyManualExecutionWindow(&task, executedAt)
	}
	if err := s.store.SaveTask(task); err != nil {
		return RunResult{}, err
	}
	if err := s.store.AppendLogs(logs); err != nil {
		return RunResult{}, err
	}
	return RunResult{
		TaskID:      task.ID,
		TaskName:    task.Name,
		Action:      action,
		ScheduledAt: scheduledAt,
		ExecutedAt:  executedAt,
		Result:      result,
		Logs:        logs,
	}, nil
}

func (s *Service) applyManualExecutionWindow(task *Task, now time.Time) {
	if task == nil {
		return
	}
	if scheduledAt, due := s.scheduledDueAt(*task, TaskActionStart, now); due {
		task.LastStartScheduledAt = scheduledAt
	}
	if scheduledAt, due := s.scheduledDueAt(*task, TaskActionStop, now); due {
		task.LastStopScheduledAt = scheduledAt
	}
}

func (s *Service) releaseTaskSources(task Task) error {
	if s.proxyGovernance == nil {
		return nil
	}
	source := governance.ScheduleProxySource(task.ID)
	for _, target := range task.Targets {
		if _, err := s.proxyGovernance.EnableProxy(target.ProxyName, source); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) nextScheduledAt(task Task, action TaskAction, now time.Time) time.Time {
	if !task.Enabled {
		return time.Time{}
	}
	rule := task.StartRule
	if action == TaskActionStop {
		rule = task.StopRule
	}
	loc, err := time.LoadLocation(task.Timezone)
	if err != nil {
		return time.Time{}
	}
	return nextRuleTime(rule, now, loc)
}

func (s *Service) scheduledDueAt(task Task, action TaskAction, now time.Time) (time.Time, bool) {
	rule := task.StartRule
	if action == TaskActionStop {
		rule = task.StopRule
	}
	loc, err := time.LoadLocation(task.Timezone)
	if err != nil {
		return time.Time{}, false
	}
	candidate := latestCandidate(rule, now, loc)
	if candidate.IsZero() {
		return time.Time{}, false
	}
	if candidate.After(now) {
		return time.Time{}, false
	}
	if now.Sub(candidate) > s.dueWindow {
		return time.Time{}, false
	}
	return candidate, true
}

func validateTask(task Task) error {
	if strings.TrimSpace(task.ID) == "" {
		return fmt.Errorf("task id is required")
	}
	if strings.TrimSpace(task.Name) == "" {
		return fmt.Errorf("task name is required")
	}
	if strings.TrimSpace(task.Timezone) == "" {
		return fmt.Errorf("timezone is required")
	}
	if _, err := time.LoadLocation(task.Timezone); err != nil {
		return fmt.Errorf("invalid timezone: %w", err)
	}
	if len(task.Targets) == 0 {
		return fmt.Errorf("at least one target proxy is required")
	}
	seen := make(map[string]struct{}, len(task.Targets))
	for _, target := range task.Targets {
		if target.ProxyName == "" {
			return fmt.Errorf("target proxy name is required")
		}
		if _, ok := seen[target.ProxyName]; ok {
			return fmt.Errorf("duplicate target proxy %q", target.ProxyName)
		}
		seen[target.ProxyName] = struct{}{}
	}
	if err := validateRule(task.StartRule); err != nil {
		return fmt.Errorf("startRule: %w", err)
	}
	if err := validateRule(task.StopRule); err != nil {
		return fmt.Errorf("stopRule: %w", err)
	}
	return nil
}

func validateRule(rule Rule) error {
	if _, _, err := parseRuleClock(rule.Time); err != nil {
		return err
	}
	switch rule.Mode {
	case RuleModeOnce:
		if strings.TrimSpace(rule.Date) == "" {
			return fmt.Errorf("date is required for once rule")
		}
		if _, err := time.Parse("2006-01-02", rule.Date); err != nil {
			return fmt.Errorf("invalid date format, expected YYYY-MM-DD")
		}
	case RuleModeDaily:
		return nil
	case RuleModeWeekly:
		if len(rule.DaysOfWeek) == 0 {
			return fmt.Errorf("daysOfWeek is required for weekly rule")
		}
		seen := map[int]struct{}{}
		for _, day := range rule.DaysOfWeek {
			if day < 1 || day > 7 {
				return fmt.Errorf("daysOfWeek values must be between 1 and 7")
			}
			if _, ok := seen[day]; ok {
				return fmt.Errorf("daysOfWeek contains duplicate value %d", day)
			}
			seen[day] = struct{}{}
		}
	default:
		return fmt.Errorf("unsupported rule mode %q", rule.Mode)
	}
	return nil
}

func normalizeTargets(targets []Target) []Target {
	out := make([]Target, 0, len(targets))
	for _, target := range targets {
		name := strings.TrimSpace(target.ProxyName)
		if name == "" {
			continue
		}
		out = append(out, Target{
			ProxyName: name,
			User:      strings.TrimSpace(target.User),
			ClientID:  strings.TrimSpace(target.ClientID),
			Type:      strings.TrimSpace(target.Type),
		})
	}
	return out
}

func normalizeRule(rule Rule) Rule {
	rule.Date = strings.TrimSpace(rule.Date)
	rule.Time = strings.TrimSpace(rule.Time)
	rule.DaysOfWeek = slices.Clone(rule.DaysOfWeek)
	slices.Sort(rule.DaysOfWeek)
	return rule
}

func parseRuleClock(value string) (int, int, error) {
	t, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid time format, expected HH:MM")
	}
	return t.Hour(), t.Minute(), nil
}

func latestCandidate(rule Rule, now time.Time, loc *time.Location) time.Time {
	hour, minute, err := parseRuleClock(rule.Time)
	if err != nil {
		return time.Time{}
	}
	nowLoc := now.In(loc)
	switch rule.Mode {
	case RuleModeOnce:
		date, err := time.ParseInLocation("2006-01-02", rule.Date, loc)
		if err != nil {
			return time.Time{}
		}
		return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, loc)
	case RuleModeDaily:
		return time.Date(nowLoc.Year(), nowLoc.Month(), nowLoc.Day(), hour, minute, 0, 0, loc)
	case RuleModeWeekly:
		if !containsWeekday(rule.DaysOfWeek, weekdayNumber(nowLoc.Weekday())) {
			return time.Time{}
		}
		return time.Date(nowLoc.Year(), nowLoc.Month(), nowLoc.Day(), hour, minute, 0, 0, loc)
	default:
		return time.Time{}
	}
}

func nextRuleTime(rule Rule, after time.Time, loc *time.Location) time.Time {
	hour, minute, err := parseRuleClock(rule.Time)
	if err != nil {
		return time.Time{}
	}
	afterLoc := after.In(loc)
	switch rule.Mode {
	case RuleModeOnce:
		date, err := time.ParseInLocation("2006-01-02", rule.Date, loc)
		if err != nil {
			return time.Time{}
		}
		candidate := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, loc)
		if candidate.After(afterLoc) {
			return candidate
		}
		return time.Time{}
	case RuleModeDaily:
		candidate := time.Date(afterLoc.Year(), afterLoc.Month(), afterLoc.Day(), hour, minute, 0, 0, loc)
		if !candidate.After(afterLoc) {
			candidate = candidate.AddDate(0, 0, 1)
		}
		return candidate
	case RuleModeWeekly:
		for dayOffset := 0; dayOffset <= 14; dayOffset++ {
			day := afterLoc.AddDate(0, 0, dayOffset)
			if !containsWeekday(rule.DaysOfWeek, weekdayNumber(day.Weekday())) {
				continue
			}
			candidate := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, loc)
			if candidate.After(afterLoc) {
				return candidate
			}
		}
	}
	return time.Time{}
}

func containsWeekday(days []int, want int) bool {
	for _, day := range days {
		if day == want {
			return true
		}
	}
	return false
}

func weekdayNumber(weekday time.Weekday) int {
	if weekday == time.Sunday {
		return 7
	}
	return int(weekday)
}

func sameScheduledAt(a, b time.Time) bool {
	if a.IsZero() || b.IsZero() {
		return a.IsZero() && b.IsZero()
	}
	return a.Equal(b)
}

func mustRandID() string {
	id, err := util.RandID()
	if err != nil || id == "" {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return id
}
