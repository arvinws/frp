package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/fatedier/frp/pkg/ext/governance"
)

type fakeStore struct {
	tasks map[string]Task
	logs  []ExecutionLog
}

func newFakeStore() *fakeStore {
	return &fakeStore{tasks: make(map[string]Task), logs: make([]ExecutionLog, 0)}
}

func (s *fakeStore) ListTasks() []Task {
	out := make([]Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		out = append(out, task)
	}
	return out
}

func (s *fakeStore) GetTask(id string) (Task, bool) {
	task, ok := s.tasks[id]
	return task, ok
}

func (s *fakeStore) SaveTask(task Task) error {
	s.tasks[task.ID] = task
	return nil
}

func (s *fakeStore) DeleteTask(id string) (bool, error) {
	if _, ok := s.tasks[id]; !ok {
		return false, nil
	}
	delete(s.tasks, id)
	return true, nil
}

func (s *fakeStore) AppendLogs(logs []ExecutionLog) error {
	s.logs = append(s.logs, logs...)
	return nil
}

func (s *fakeStore) ListLogs(taskID string, limit int) []ExecutionLog {
	return s.logs
}

type fakeProxyGovernance struct {
	disabled []string
	enabled  []string
}

func (f *fakeProxyGovernance) DisableProxy(proxyName, source, reason, operator string) (governance.ProxyActionResult, error) {
	f.disabled = append(f.disabled, proxyName+"@"+source)
	return governance.ProxyActionResult{ProxyName: proxyName, Result: "disabled"}, nil
}

func (f *fakeProxyGovernance) EnableProxy(proxyName, source string) (governance.ProxyActionResult, error) {
	f.enabled = append(f.enabled, proxyName+"@"+source)
	return governance.ProxyActionResult{ProxyName: proxyName, Result: "enabled"}, nil
}

func TestValidateTaskWeekly(t *testing.T) {
	err := validateTask(Task{
		ID:       "task-1",
		Name:     "Workday",
		Enabled:  true,
		Timezone: "Asia/Shanghai",
		Targets:  []Target{{ProxyName: "alice.tcp"}},
		StartRule: Rule{Mode: RuleModeWeekly, Time: "09:00", DaysOfWeek: []int{1, 2, 3, 4, 5}},
		StopRule:  Rule{Mode: RuleModeWeekly, Time: "18:00", DaysOfWeek: []int{1, 2, 3, 4, 5}},
	})
	if err != nil {
		t.Fatalf("validate task: %v", err)
	}
}

func TestRunPendingExecutesDueTasks(t *testing.T) {
	store := newFakeStore()
	proxyGovernance := &fakeProxyGovernance{}
	svc := NewService(store, proxyGovernance)
	now := time.Date(2026, 3, 13, 10, 0, 15, 0, time.FixedZone("CST", 8*3600))
	svc.nowFn = func() time.Time { return now }
	svc.scanInterval = time.Second
	svc.dueWindow = time.Minute

	store.tasks["task-1"] = Task{
		ID:       "task-1",
		Name:     "Office",
		Enabled:  true,
		Timezone: "Asia/Shanghai",
		Targets:  []Target{{ProxyName: "alice.tcp"}},
		StartRule: Rule{Mode: RuleModeDaily, Time: "10:00"},
		StopRule:  Rule{Mode: RuleModeDaily, Time: "18:00"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()

	if len(proxyGovernance.enabled) == 0 {
		t.Fatalf("expected due start action to enable proxy")
	}
	updated := store.tasks["task-1"]
	if updated.LastExecutionAction != TaskActionStart {
		t.Fatalf("expected last execution action start, got %q", updated.LastExecutionAction)
	}
	if len(store.logs) == 0 {
		t.Fatalf("expected execution log to be recorded")
	}
}

func TestManualRunSuppressesImmediateDueSchedule(t *testing.T) {
	store := newFakeStore()
	proxyGovernance := &fakeProxyGovernance{}
	svc := NewService(store, proxyGovernance)
	now := time.Date(2026, 3, 13, 10, 0, 10, 0, time.FixedZone("CST", 8*3600))
	svc.nowFn = func() time.Time { return now }
	svc.dueWindow = time.Minute

	store.tasks["task-1"] = Task{
		ID:       "task-1",
		Name:     "Office",
		Enabled:  true,
		Timezone: "Asia/Shanghai",
		Targets:  []Target{{ProxyName: "alice.tcp"}},
		StartRule: Rule{Mode: RuleModeDaily, Time: "10:00"},
		StopRule:  Rule{Mode: RuleModeDaily, Time: "18:00"},
	}

	if _, err := svc.RunTask("task-1", TaskActionStop); err != nil {
		t.Fatalf("manual stop run: %v", err)
	}
	if len(proxyGovernance.disabled) != 1 {
		t.Fatalf("expected one manual stop call, got %v", proxyGovernance.disabled)
	}

	svc.runPending()
	if len(proxyGovernance.enabled) != 0 {
		t.Fatalf("expected no immediate scheduled start after manual stop, got %v", proxyGovernance.enabled)
	}
	updated := store.tasks["task-1"]
	if updated.LastStartScheduledAt.IsZero() {
		t.Fatalf("expected manual stop to mark current start window as handled")
	}
	if updated.LastExecutionAction != TaskActionStop {
		t.Fatalf("expected last execution action stop, got %q", updated.LastExecutionAction)
	}
}
