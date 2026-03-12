package scheduler

import "time"

type TaskAction string

const (
	TaskActionStart TaskAction = "start"
	TaskActionStop  TaskAction = "stop"
)

type ExecutionTrigger string

const (
	ExecutionTriggerSchedule ExecutionTrigger = "schedule"
	ExecutionTriggerManual   ExecutionTrigger = "manual"
)

type RuleMode string

const (
	RuleModeOnce   RuleMode = "once"
	RuleModeDaily  RuleMode = "daily"
	RuleModeWeekly RuleMode = "weekly"
)

type Rule struct {
	Mode       RuleMode `json:"mode"`
	Date       string   `json:"date,omitempty"`
	Time       string   `json:"time"`
	DaysOfWeek []int    `json:"daysOfWeek,omitempty"`
}

type Target struct {
	ProxyName string `json:"proxyName"`
	User      string `json:"user,omitempty"`
	ClientID  string `json:"clientID,omitempty"`
	Type      string `json:"type,omitempty"`
}

type Task struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Enabled              bool       `json:"enabled"`
	Timezone             string     `json:"timezone"`
	Targets              []Target   `json:"targets"`
	StartRule            *Rule      `json:"startRule,omitempty"`
	StopRule             *Rule      `json:"stopRule,omitempty"`
	Remark               string     `json:"remark,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
	LastExecutionAt      time.Time  `json:"lastExecutionAt,omitempty"`
	LastExecutionAction  TaskAction `json:"lastExecutionAction,omitempty"`
	LastExecutionResult  string     `json:"lastExecutionResult,omitempty"`
	LastStartScheduledAt time.Time  `json:"lastStartScheduledAt,omitempty"`
	LastStopScheduledAt  time.Time  `json:"lastStopScheduledAt,omitempty"`
}

type ExecutionLog struct {
	ID              string           `json:"id"`
	TaskID          string           `json:"taskID"`
	TaskName        string           `json:"taskName"`
	Action          TaskAction       `json:"action"`
	Trigger         ExecutionTrigger `json:"trigger"`
	ScheduledAt     time.Time        `json:"scheduledAt"`
	ExecutedAt      time.Time        `json:"executedAt"`
	TargetProxyName string           `json:"targetProxyName"`
	Result          string           `json:"result"`
	Error           string           `json:"error,omitempty"`
}

type ProxyOption struct {
	ProxyName   string `json:"proxyName"`
	DisplayName string `json:"displayName"`
	Type        string `json:"type,omitempty"`
	User        string `json:"user,omitempty"`
	ClientID    string `json:"clientID,omitempty"`
	Status      string `json:"status"`
	Disabled    bool   `json:"disabled"`
}

type ProxyOptionFilter struct {
	Keyword  string
	ClientID string
	User     string
	Type     string
	Status   string
}

type TaskView struct {
	Task
	NextStartAt time.Time `json:"nextStartAt,omitempty"`
	NextStopAt  time.Time `json:"nextStopAt,omitempty"`
}

type RunResult struct {
	TaskID      string       `json:"taskID"`
	TaskName    string       `json:"taskName"`
	Action      TaskAction   `json:"action"`
	ScheduledAt time.Time    `json:"scheduledAt"`
	ExecutedAt  time.Time    `json:"executedAt"`
	Result      string       `json:"result"`
	Logs        []ExecutionLog `json:"logs"`
}
