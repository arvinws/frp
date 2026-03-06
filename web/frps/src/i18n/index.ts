import { ref } from 'vue'

export type Locale = 'en-US' | 'zh-CN'

const LOCALE_STORAGE_KEY = 'frp.dashboard.locale'

const messages = {
  'en-US': {
    app: {
      server: 'Server',
      overview: 'Overview',
      clients: 'Clients',
      proxies: 'Proxies',
      traffic: 'Traffic',
      language: 'Language',
      github: 'GitHub',
    },
    common: {
      ip: 'IP',
      online: 'Online',
      offline: 'Offline',
      enabled: 'Enabled',
      disabled: 'Disabled',
      noLimit: 'No limit',
      loading: 'Loading...',
    },
    overview: {
      connectedClients: 'Connected clients',
      activeProxies: 'Active proxies',
      currentConnections: 'Current connections',
      totalToday: 'Total today',
      networkTraffic: 'Network Traffic',
      today: 'Today',
      inbound: 'Inbound',
      outbound: 'Outbound',
      proxyTypes: 'Proxy Types',
      now: 'Now',
      noActiveProxies: 'No active proxies',
      serverConfiguration: 'Server Configuration',
      bindPort: 'Bind Port',
      kcpPort: 'KCP Port',
      quicPort: 'QUIC Port',
      httpPort: 'HTTP Port',
      httpsPort: 'HTTPS Port',
      tcpmuxPort: 'TCPMux Port',
      subdomainHost: 'Subdomain Host',
      maxPoolCount: 'Max Pool Count',
      maxPortsPerClient: 'Max Ports/Client',
      allowPorts: 'Allow Ports',
      tlsForce: 'TLS Force',
      heartbeatTimeout: 'Heartbeat Timeout',
      fetchFailed: 'Get server info from frps failed!',
    },
    clients: {
      title: 'Clients',
      subtitle: 'Manage connected clients and their status',
      all: 'All',
      online: 'Online',
      offline: 'Offline',
      searchPlaceholder: 'Search clients...',
      noClientsFound: 'No clients found',
      fetchFailed: 'Failed to fetch clients',
      fetchClientFailed: 'Failed to fetch client',
      backToClients: 'Back to Clients',
      clientNotFound: 'Client not found',
      clientNotFoundDesc: "The client doesn't exist or has been removed.",
      connections: 'Connections',
      runID: 'Run ID',
      firstConnected: 'First Connected',
      connected: 'Connected',
      disconnected: 'Disconnected',
      proxies: 'Proxies',
      searchProxiesPlaceholder: 'Search proxies...',
      noProxiesMatch: 'No proxies match "{keyword}"',
      noProxiesFound: 'No proxies found',
    },
    governance: {
      disabled: 'Disabled',
      disable: 'Disable',
      enable: 'Enable',
      disconnect: 'Disconnect',
      disableAndDisconnect: 'Disable & Disconnect',
      reason: 'Reason',
      reasonPlaceholder: 'Enter reason (optional)',
      operator: 'Operator',
      operatorPlaceholder: 'Enter operator name (optional)',
      confirmDisable: 'Are you sure to disable this client? It will be denied from logging in and creating new proxies.',
      confirmEnable: 'Are you sure to enable this client?',
      confirmDisconnect: 'Are you sure to disconnect this session? The client may reconnect immediately.',
      confirmDisableAndDisconnect: 'Are you sure to disable and disconnect this client? It will be blocked from reconnecting.',
      disableSuccess: 'Client disabled successfully',
      enableSuccess: 'Client enabled successfully',
      disconnectSuccess: 'Session disconnected successfully',
      disableAndDisconnectSuccess: 'Client disabled and session disconnected',
      operationFailed: 'Operation failed',
    },
    proxies: {
      title: 'Proxies',
      subtitle: 'View and manage all proxy configurations',
      refresh: 'Refresh',
      clearOffline: 'Clear Offline',
      clearOfflineConfirm: 'Clear all offline proxies?',
      clear: 'Clear',
      cancel: 'Cancel',
      searchPlaceholder: 'Search proxies...',
      allClients: 'All Clients',
      clientNotFound: '{label} (not found)',
      noProxiesFound: 'No proxies found',
      fetchFailed: 'Failed to fetch proxies',
      clearSuccess: 'Successfully cleared offline proxies',
      clearFailed: 'Failed to clear offline proxies',
    },
    proxy: {
      client: 'Client',
      port: 'Port',
      connections: 'Connections',
      trafficIn: 'Traffic In',
      trafficOut: 'Traffic Out',
      statusTimeline: 'Status Timeline',
      lastStartTime: 'Last Start Time',
      lastCloseTime: 'Last Close Time',
      configuration: 'Configuration',
      encryption: 'Encryption',
      compression: 'Compression',
      customDomains: 'Custom Domains',
      subdomain: 'Subdomain',
      locations: 'Locations',
      hostRewrite: 'Host Rewrite',
      multiplexer: 'Multiplexer',
      routeByHTTPUser: 'Route By HTTP User',
      trafficStatistics: 'Traffic Statistics',
      notFound: 'Proxy not found',
      notFoundDesc: "The proxy doesn't exist or has been removed.",
      backToProxies: 'Back to Proxies',
      fetchFailed: 'Failed to fetch proxy',
    },
    traffic: {
      tooltipIn: 'In: {value}',
      tooltipOut: 'Out: {value}',
      legendIn: 'Traffic In',
      legendOut: 'Traffic Out',
      noData: 'No traffic data',
      fetchFailed: 'Get traffic info failed! {error}',
    },
  },
  'zh-CN': {
    app: {
      server: '服务端',
      overview: '概览',
      clients: '客户端',
      proxies: '代理',
      traffic: '流量',
      language: '语言',
      github: 'GitHub',
    },
    common: {
      ip: 'IP',
      online: '在线',
      offline: '离线',
      enabled: '已启用',
      disabled: '已禁用',
      noLimit: '不限制',
      loading: '加载中...',
    },
    overview: {
      connectedClients: '在线客户端',
      activeProxies: '活跃代理',
      currentConnections: '当前连接数',
      totalToday: '今日总流量',
      networkTraffic: '网络流量',
      today: '今日',
      inbound: '入站',
      outbound: '出站',
      proxyTypes: '代理类型',
      now: '当前',
      noActiveProxies: '暂无活跃代理',
      serverConfiguration: '服务端配置',
      bindPort: '绑定端口',
      kcpPort: 'KCP 端口',
      quicPort: 'QUIC 端口',
      httpPort: 'HTTP 端口',
      httpsPort: 'HTTPS 端口',
      tcpmuxPort: 'TCPMux 端口',
      subdomainHost: '子域名主机',
      maxPoolCount: '最大连接池',
      maxPortsPerClient: '每客户端最大端口数',
      allowPorts: '允许端口',
      tlsForce: '强制 TLS',
      heartbeatTimeout: '心跳超时',
      fetchFailed: '从 frps 获取服务信息失败！',
    },
    clients: {
      title: '客户端',
      subtitle: '管理已连接客户端及其状态',
      all: '全部',
      online: '在线',
      offline: '离线',
      searchPlaceholder: '搜索客户端...',
      noClientsFound: '未找到客户端',
      fetchFailed: '获取客户端列表失败',
      fetchClientFailed: '获取客户端详情失败',
      backToClients: '返回客户端列表',
      clientNotFound: '客户端不存在',
      clientNotFoundDesc: '该客户端不存在或已被移除。',
      connections: '连接数',
      runID: '运行 ID',
      firstConnected: '首次连接',
      connected: '最近连接',
      disconnected: '断开时间',
      proxies: '代理',
      searchProxiesPlaceholder: '搜索代理...',
      noProxiesMatch: '没有匹配“{keyword}”的代理',
      noProxiesFound: '未找到代理',
    },
    governance: {
      disabled: '已禁用',
      disable: '禁用',
      enable: '启用',
      disconnect: '断开连接',
      disableAndDisconnect: '禁用并断开',
      reason: '原因',
      reasonPlaceholder: '请输入原因（可选）',
      operator: '操作人',
      operatorPlaceholder: '请输入操作人（可选）',
      confirmDisable: '确定要禁用该客户端吗？禁用后将无法登录和创建新代理。',
      confirmEnable: '确定要启用该客户端吗？',
      confirmDisconnect: '确定要断开该会话吗？客户端可能会立即重连。',
      confirmDisableAndDisconnect: '确定要禁用并断开该客户端吗？客户端将无法重新连接。',
      disableSuccess: '客户端已禁用',
      enableSuccess: '客户端已启用',
      disconnectSuccess: '会话已断开',
      disableAndDisconnectSuccess: '客户端已禁用并断开连接',
      operationFailed: '操作失败',
    },
    proxies: {
      title: '代理',
      subtitle: '查看和管理所有代理配置',
      refresh: '刷新',
      clearOffline: '清理离线',
      clearOfflineConfirm: '确认清理所有离线代理？',
      clear: '清理',
      cancel: '取消',
      searchPlaceholder: '搜索代理...',
      allClients: '全部客户端',
      clientNotFound: '{label}（不存在）',
      noProxiesFound: '未找到代理',
      fetchFailed: '获取代理列表失败',
      clearSuccess: '已成功清理离线代理',
      clearFailed: '清理离线代理失败',
    },
    proxy: {
      client: '客户端',
      port: '端口',
      connections: '连接数',
      trafficIn: '入站流量',
      trafficOut: '出站流量',
      statusTimeline: '状态时间线',
      lastStartTime: '最近启动时间',
      lastCloseTime: '最近关闭时间',
      configuration: '配置项',
      encryption: '加密',
      compression: '压缩',
      customDomains: '自定义域名',
      subdomain: '子域名',
      locations: '路径匹配',
      hostRewrite: 'Host 重写',
      multiplexer: '复用器',
      routeByHTTPUser: '按 HTTP 用户路由',
      trafficStatistics: '流量统计',
      notFound: '代理不存在',
      notFoundDesc: '该代理不存在或已被移除。',
      backToProxies: '返回代理列表',
      fetchFailed: '获取代理详情失败',
    },
    traffic: {
      tooltipIn: '入站: {value}',
      tooltipOut: '出站: {value}',
      legendIn: '入站流量',
      legendOut: '出站流量',
      noData: '暂无流量数据',
      fetchFailed: '获取流量信息失败！{error}',
    },
  },
} as const

type MessageTree = Record<string, unknown>

function normalizeLocale(input: string | null | undefined): Locale | null {
  if (!input) {
    return null
  }

  const value = input.toLowerCase()
  if (value.startsWith('zh')) {
    return 'zh-CN'
  }
  if (value.startsWith('en')) {
    return 'en-US'
  }
  return null
}

function detectLocale(): Locale {
  if (typeof localStorage !== 'undefined') {
    try {
      const stored = normalizeLocale(localStorage.getItem(LOCALE_STORAGE_KEY))
      if (stored) {
        return stored
      }
    } catch {
      // Ignore localStorage errors and continue with browser detection.
    }
  }

  if (typeof navigator !== 'undefined') {
    const candidates = Array.isArray(navigator.languages)
      ? navigator.languages
      : [navigator.language]
    for (const candidate of candidates) {
      const matched = normalizeLocale(candidate)
      if (matched) {
        return matched
      }
    }
  }

  return 'en-US'
}

function applyDocumentLocale(next: Locale) {
  if (typeof document !== 'undefined') {
    document.documentElement.lang = next
  }
}

export const locale = ref<Locale>(detectLocale())
applyDocumentLocale(locale.value)

function getMessageValue(source: MessageTree, key: string): unknown {
  return key.split('.').reduce<unknown>((acc, segment) => {
    if (acc && typeof acc === 'object' && segment in acc) {
      return (acc as Record<string, unknown>)[segment]
    }
    return undefined
  }, source)
}

function interpolate(
  template: string,
  params?: Record<string, string | number>,
): string {
  if (!params) {
    return template
  }
  return template.replace(/\{(\w+)\}/g, (_, key: string) => {
    const value = params[key]
    return value === undefined ? '' : String(value)
  })
}

export function t(
  key: string,
  params?: Record<string, string | number>,
): string {
  const activeMessages = messages[locale.value]
  const fallbackMessages = messages['en-US']
  const raw =
    getMessageValue(activeMessages, key) ?? getMessageValue(fallbackMessages, key)
  if (typeof raw !== 'string') {
    return key
  }
  return interpolate(raw, params)
}

export function setLocale(next: Locale) {
  locale.value = next
  applyDocumentLocale(next)
  if (typeof localStorage !== 'undefined') {
    try {
      localStorage.setItem(LOCALE_STORAGE_KEY, next)
    } catch {
      // Ignore localStorage errors.
    }
  }
}

export function useI18n() {
  return {
    locale,
    setLocale,
    t,
  }
}
