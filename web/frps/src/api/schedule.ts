import { http } from './http'
import type {
  ProxyOption,
  ScheduleLogsResponse,
  ScheduleRunResponse,
  ScheduleTask,
  ScheduleTaskListResponse,
  ScheduleTaskPayload,
} from '../types/schedule'

export const getSchedules = () => {
  return http.get<ScheduleTaskListResponse>('../api/admin/schedules')
}

export const getSchedule = (id: string) => {
  return http.get<ScheduleTask>(`../api/admin/schedules/${encodeURIComponent(id)}`)
}

export const createSchedule = (body: ScheduleTaskPayload) => {
  return http.post<ScheduleTask>('../api/admin/schedules', body)
}

export const updateSchedule = (id: string, body: ScheduleTaskPayload) => {
  return http.put<ScheduleTask>(`../api/admin/schedules/${encodeURIComponent(id)}`, body)
}

export const deleteSchedule = (id: string) => {
  return http.delete(`../api/admin/schedules/${encodeURIComponent(id)}`)
}

export const enableSchedule = (id: string) => {
  return http.post<ScheduleTask>(
    `../api/admin/schedules/${encodeURIComponent(id)}/enable`,
  )
}

export const disableSchedule = (id: string) => {
  return http.post<ScheduleTask>(
    `../api/admin/schedules/${encodeURIComponent(id)}/disable`,
  )
}

export const runSchedule = (id: string, action: 'start' | 'stop') => {
  return http.post<ScheduleRunResponse>(
    `../api/admin/schedules/${encodeURIComponent(id)}/run?action=${action}`,
  )
}

export const getScheduleLogs = (id: string, limit = 50) => {
  return http.get<ScheduleLogsResponse>(
    `../api/admin/schedules/${encodeURIComponent(id)}/logs?limit=${limit}`,
  )
}

export const getProxyOptions = (params?: Record<string, string>) => {
  const search = new URLSearchParams(params).toString()
  const suffix = search ? `?${search}` : ''
  return http.get<ProxyOption[]>(`../api/admin/proxy-options${suffix}`)
}
