import { http } from './http'
import type {
  ActionRequest,
  RuleActionResponse,
  DisconnectResponse,
  DisableAndDisconnectResponse,
} from '../types/client'

export const disableClient = (clientID: string, body?: ActionRequest) => {
  return http.post<RuleActionResponse>(
    `../api/admin/clients/${encodeURIComponent(clientID)}/disable`,
    body,
  )
}

export const enableClient = (clientID: string, body?: ActionRequest) => {
  return http.post<RuleActionResponse>(
    `../api/admin/clients/${encodeURIComponent(clientID)}/enable`,
    body,
  )
}

export const disconnectSession = (runID: string, body?: ActionRequest) => {
  return http.post<DisconnectResponse>(
    `../api/admin/sessions/${encodeURIComponent(runID)}/disconnect`,
    body,
  )
}

export const disableAndDisconnect = (
  clientID: string,
  body?: ActionRequest,
) => {
  return http.post<DisableAndDisconnectResponse>(
    `../api/admin/clients/${encodeURIComponent(clientID)}/disable-and-disconnect`,
    body,
  )
}
