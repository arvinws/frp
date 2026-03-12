package scheduler

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

const defaultMaxLogs = 2000

type Store interface {
	ListTasks() []Task
	GetTask(id string) (Task, bool)
	SaveTask(task Task) error
	DeleteTask(id string) (bool, error)
	AppendLogs(logs []ExecutionLog) error
	ListLogs(taskID string, limit int) []ExecutionLog
}

type FileStore struct {
	mu       sync.RWMutex
	taskPath string
	logPath  string
	maxLogs  int
	tasks    map[string]Task
	logs     []ExecutionLog
}

func NewFileStore(taskPath, logPath string) (*FileStore, error) {
	if taskPath == "" {
		taskPath = filepath.Join("data", "frps-schedules.json")
	}
	if logPath == "" {
		logPath = filepath.Join("data", "frps-schedule-logs.json")
	}

	s := &FileStore{
		taskPath: taskPath,
		logPath:  logPath,
		maxLogs:  defaultMaxLogs,
		tasks:    make(map[string]Task),
		logs:     make([]ExecutionLog, 0),
	}
	if err := s.loadTasks(); err != nil {
		return nil, err
	}
	if err := s.loadLogs(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *FileStore) ListTasks() []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		out = append(out, cloneTask(task))
	}
	slices.SortFunc(out, func(a, b Task) int {
		if v := cmp.Compare(a.Name, b.Name); v != 0 {
			return v
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return out
}

func (s *FileStore) GetTask(id string) (Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	if !ok {
		return Task{}, false
	}
	return cloneTask(task), true
}

func (s *FileStore) SaveTask(task Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = cloneTask(task)
	return s.saveTasksLocked()
}

func (s *FileStore) DeleteTask(id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[id]; !ok {
		return false, nil
	}
	delete(s.tasks, id)
	return true, s.saveTasksLocked()
}

func (s *FileStore) AppendLogs(logs []ExecutionLog) error {
	if len(logs) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, log := range logs {
		s.logs = append(s.logs, cloneLog(log))
	}
	if len(s.logs) > s.maxLogs {
		s.logs = slices.Clone(s.logs[len(s.logs)-s.maxLogs:])
	}
	return s.saveLogsLocked()
}

func (s *FileStore) ListLogs(taskID string, limit int) []ExecutionLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ExecutionLog, 0)
	for i := len(s.logs) - 1; i >= 0; i-- {
		log := s.logs[i]
		if taskID != "" && log.TaskID != taskID {
			continue
		}
		result = append(result, cloneLog(log))
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func (s *FileStore) loadTasks() error {
	data, err := os.ReadFile(s.taskPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("load schedule tasks: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return fmt.Errorf("parse schedule tasks: %w", err)
	}
	for _, task := range tasks {
		s.tasks[task.ID] = cloneTask(task)
	}
	return nil
}

func (s *FileStore) loadLogs() error {
	data, err := os.ReadFile(s.logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("load schedule logs: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	var logs []ExecutionLog
	if err := json.Unmarshal(data, &logs); err != nil {
		return fmt.Errorf("parse schedule logs: %w", err)
	}
	s.logs = make([]ExecutionLog, 0, len(logs))
	for _, log := range logs {
		s.logs = append(s.logs, cloneLog(log))
	}
	return nil
}

func (s *FileStore) saveTasksLocked() error {
	tasks := make([]Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, cloneTask(task))
	}
	slices.SortFunc(tasks, func(a, b Task) int {
		if v := cmp.Compare(a.Name, b.Name); v != 0 {
			return v
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return writeJSONFile(s.taskPath, tasks)
}

func (s *FileStore) saveLogsLocked() error {
	logs := make([]ExecutionLog, 0, len(s.logs))
	for _, log := range s.logs {
		logs = append(logs, cloneLog(log))
	}
	return writeJSONFile(s.logPath, logs)
}

func writeJSONFile(path string, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func cloneTask(task Task) Task {
	out := task
	out.Targets = slices.Clone(task.Targets)
	if task.StartRule != nil {
		startRule := *task.StartRule
		startRule.DaysOfWeek = slices.Clone(task.StartRule.DaysOfWeek)
		out.StartRule = &startRule
	}
	if task.StopRule != nil {
		stopRule := *task.StopRule
		stopRule.DaysOfWeek = slices.Clone(task.StopRule.DaysOfWeek)
		out.StopRule = &stopRule
	}
	return out
}

func cloneLog(log ExecutionLog) ExecutionLog {
	return log
}

var _ Store = (*FileStore)(nil)
