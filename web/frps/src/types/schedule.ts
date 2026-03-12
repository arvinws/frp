export type ScheduleRuleMode = 'once' | 'daily' | 'weekly'
export type ScheduleTaskAction = 'start' | 'stop'
export type ScheduleTrigger = 'schedule' | 'manual'

export interface ScheduleRule {
  mode: ScheduleRuleMode
  date?: string
  time: string
  daysOfWeek?: number[]
}

export interface ScheduleTarget {
  proxyName: string
  user?: string
  clientID?: string
  type?: string
}

export interface ScheduleTask {
  id: string
  name: string
  enabled: boolean
  timezone: string
  targets: ScheduleTarget[]
  startRule?: ScheduleRule
  stopRule?: ScheduleRule
  remark?: string
  createdAt: number
  updatedAt: number
  lastExecutionAt?: number
  lastExecutionAction?: ScheduleTaskAction
  lastExecutionResult?: string
  nextStartAt?: number
  nextStopAt?: number
  targetCount: number
}

export interface ScheduleTaskListResponse {
  tasks: ScheduleTask[]
}

export interface ScheduleExecutionLog {
  id: string
  taskID: string
  taskName: string
  action: ScheduleTaskAction
  trigger: ScheduleTrigger
  scheduledAt: number
  executedAt: number
  targetProxyName: string
  result: string
  error?: string
}

export interface ScheduleLogsResponse {
  logs: ScheduleExecutionLog[]
}

export interface ScheduleRunResponse extends ScheduleTask {
  logs?: ScheduleExecutionLog[]
}

export interface ProxyOption {
  proxyName: string
  displayName: string
  type?: string
  user?: string
  clientID?: string
  status: string
  disabled: boolean
}

export interface ScheduleTaskPayload {
  name: string
  enabled: boolean
  timezone: string
  targets: ScheduleTarget[]
  startRule?: ScheduleRule
  stopRule?: ScheduleRule
  remark?: string
}
