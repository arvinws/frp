export interface ClientInfoData {
  key: string
  user: string
  clientID: string
  runID: string
  version?: string
  hostname: string
  clientIP?: string
  metas?: Record<string, string>
  firstConnectedAt: number
  lastConnectedAt: number
  disconnectedAt?: number
  online: boolean
  disabled: boolean
}

export interface ActionRequest {
  reason?: string
  operator?: string
  banIP?: boolean
  clientID?: string
}

export interface RuleActionResponse {
  clientID: string
  status: string
  changed: boolean
  reason?: string
  operator?: string
  updatedAt?: number
  result: string
}

export interface DisconnectResponse {
  runID: string
  clientID?: string
  result: string
}

export interface DisableAndDisconnectResponse {
  clientID: string
  status: string
  ruleChanged: boolean
  reason?: string
  operator?: string
  updatedAt?: number
  runID?: string
  disconnectResult: string
}
