<template>
  <div class="client-detail-page">
    <!-- Breadcrumb -->
    <nav class="breadcrumb">
      <a class="breadcrumb-link" @click="goBack">
        <el-icon><ArrowLeft /></el-icon>
      </a>
      <router-link to="/clients" class="breadcrumb-item">{{
        t('app.clients')
      }}</router-link>
      <span class="breadcrumb-separator">/</span>
      <span class="breadcrumb-current">{{
        client?.displayName || route.params.key
      }}</span>
    </nav>

    <div v-loading="loading" class="detail-content">
      <template v-if="client">
        <!-- Header Card -->
        <div class="header-card">
          <div class="header-main">
            <div class="header-left">
              <div class="client-avatar">
                {{ client.displayName.charAt(0).toUpperCase() }}
              </div>
              <div class="client-info">
                <div class="client-name-row">
                  <h1 class="client-name">{{ client.displayName }}</h1>
                  <el-tag v-if="client.version" size="small" type="success"
                    >v{{ client.version }}</el-tag
                  >
                </div>
                <div class="client-meta">
                  <span v-if="client.ip" class="meta-item">{{
                    client.ip
                  }}</span>
                  <span v-if="client.hostname" class="meta-item">{{
                    client.hostname
                  }}</span>
                </div>
              </div>
            </div>
            <div class="header-right">
              <span
                class="status-badge"
                :class="client.disabled ? 'banned' : client.online ? 'online' : 'offline'"
              >
                {{
                  client.disabled
                    ? t('governance.disabled')
                    : client.online
                      ? t('common.online')
                      : t('common.offline')
                }}
              </span>
            </div>
          </div>

          <!-- Info Section -->
          <div class="info-section">
            <div class="info-item">
              <span class="info-label">{{ t('clients.connections') }}</span>
              <span class="info-value">{{ totalConnections }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">{{ t('clients.runID') }}</span>
              <span class="info-value">{{ client.runID }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">{{ t('clients.firstConnected') }}</span>
              <span class="info-value">{{ client.firstConnectedAgo }}</span>
            </div>
            <div class="info-item">
              <span class="info-label">{{
                client.online ? t('clients.connected') : t('clients.disconnected')
              }}</span>
              <span class="info-value">{{
                client.online ? client.lastConnectedAgo : client.disconnectedAgo
              }}</span>
            </div>
          </div>

          <!-- Governance Actions -->
          <div v-if="client.clientID" class="governance-section">
            <el-button
              v-if="!client.disabled"
              type="danger"
              plain
              size="small"
              :icon="Lock"
              :loading="actionLoading"
              @click="showActionDialog('disable')"
            >
              {{ t('governance.disable') }}
            </el-button>
            <el-button
              v-if="client.disabled"
              type="success"
              plain
              size="small"
              :icon="Unlock"
              :loading="actionLoading"
              @click="showActionDialog('enable')"
            >
              {{ t('governance.enable') }}
            </el-button>
            <el-button
              v-if="client.online"
              type="warning"
              plain
              size="small"
              :icon="SwitchButton"
              :loading="actionLoading"
              @click="showActionDialog('disconnect')"
            >
              {{ t('governance.disconnect') }}
            </el-button>
          </div>
        </div>

        <!-- Governance Action Dialog -->
        <el-dialog
          v-model="dialogVisible"
          :title="dialogTitle"
          width="420px"
          :close-on-click-modal="false"
        >
          <p class="dialog-confirm-text">{{ dialogConfirmText }}</p>
          <el-form label-position="top">
            <el-form-item
              v-if="dialogAction === 'disable' && client?.online"
              class="disconnect-switch-item"
            >
              <div class="disconnect-switch-row">
                <el-switch v-model="alsoDisconnect" />
                <span class="disconnect-switch-label">
                  {{ t('governance.alsoDisconnect') }}
                </span>
                <el-tag size="small" type="warning">{{
                  t('governance.recommended')
                }}</el-tag>
              </div>
              <p class="disconnect-switch-hint">
                {{ t('governance.alsoDisconnectHint') }}
              </p>
            </el-form-item>
            <el-form-item
              v-if="dialogAction === 'disable'"
              class="disconnect-switch-item"
            >
              <div class="disconnect-switch-row">
                <el-switch v-model="alsoBanIP" />
                <span class="disconnect-switch-label">
                  {{ t('governance.banIP') }}
                </span>
              </div>
              <p class="disconnect-switch-hint">
                {{ t('governance.banIPHint') }}
              </p>
            </el-form-item>
            <el-form-item :label="t('governance.reason')">
              <el-input
                v-model="actionForm.reason"
                :placeholder="t('governance.reasonPlaceholder')"
              />
            </el-form-item>
            <el-form-item :label="t('governance.operator')">
              <el-input
                v-model="actionForm.operator"
                :placeholder="t('governance.operatorPlaceholder')"
              />
            </el-form-item>
          </el-form>
          <template #footer>
            <el-button @click="dialogVisible = false">{{
              t('proxies.cancel')
            }}</el-button>
            <el-button
              :type="dialogAction === 'enable' ? 'success' : 'danger'"
              :loading="actionLoading"
              @click="executeAction"
            >
              {{ dialogTitle }}
            </el-button>
          </template>
        </el-dialog>

        <!-- Proxies Card -->
        <div class="proxies-card">
          <div class="proxies-header">
            <div class="proxies-title">
              <h2>{{ t('clients.proxies') }}</h2>
              <span class="proxies-count">{{ filteredProxies.length }}</span>
            </div>
            <el-input
              v-model="proxySearch"
              :placeholder="t('clients.searchProxiesPlaceholder')"
              :prefix-icon="Search"
              clearable
              class="proxy-search"
            />
          </div>
          <div class="proxies-body">
            <div v-if="proxiesLoading" class="loading-state">
              <el-icon class="is-loading"><Loading /></el-icon>
              <span>{{ t('common.loading') }}</span>
            </div>
            <div v-else-if="filteredProxies.length > 0" class="proxies-list">
              <ProxyCard
                v-for="proxy in filteredProxies"
                :key="proxy.name"
                :proxy="proxy"
                show-type
                @refresh="fetchProxies"
              />
            </div>
            <div v-else-if="clientProxies.length > 0" class="empty-state">
              <p>{{ t('clients.noProxiesMatch', { keyword: proxySearch }) }}</p>
            </div>
            <div v-else class="empty-state">
              <p>{{ t('clients.noProxiesFound') }}</p>
            </div>
          </div>
        </div>
      </template>

      <div v-else-if="!loading" class="not-found">
        <h2>{{ t('clients.clientNotFound') }}</h2>
        <p>{{ t('clients.clientNotFoundDesc') }}</p>
        <router-link to="/clients">
          <el-button type="primary">{{ t('clients.backToClients') }}</el-button>
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  ArrowLeft,
  Loading,
  Search,
  Lock,
  Unlock,
  SwitchButton,
} from '@element-plus/icons-vue'
import { Client } from '../utils/client'
import { getClient } from '../api/client'
import {
  disableClient,
  enableClient,
  disconnectSession,
  disableAndDisconnect,
} from '../api/admin'
import { getProxiesByType } from '../api/proxy'
import {
  BaseProxy,
  TCPProxy,
  UDPProxy,
  HTTPProxy,
  HTTPSProxy,
  TCPMuxProxy,
  STCPProxy,
  SUDPProxy,
} from '../utils/proxy'
import { getServerInfo } from '../api/server'
import ProxyCard from '../components/ProxyCard.vue'
import { useI18n } from '../i18n'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const client = ref<Client | null>(null)
const loading = ref(true)

type GovernanceAction = 'disable' | 'enable' | 'disconnect'
const dialogVisible = ref(false)
const dialogAction = ref<GovernanceAction>('disable')
const actionLoading = ref(false)
const actionForm = ref({ reason: '', operator: '' })
const alsoDisconnect = ref(true)
const alsoBanIP = ref(false)

const dialogTitle = computed(() => {
  if (dialogAction.value === 'disable' && alsoDisconnect.value && client.value?.online) {
    return t('governance.disableAndDisconnect')
  }
  const map: Record<GovernanceAction, string> = {
    disable: t('governance.disable'),
    enable: t('governance.enable'),
    disconnect: t('governance.disconnect'),
  }
  return map[dialogAction.value]
})

const dialogConfirmText = computed(() => {
  if (dialogAction.value === 'disable' && client.value?.online) {
    return t('governance.confirmDisableOnline')
  }
  const map: Record<GovernanceAction, string> = {
    disable: t('governance.confirmDisable'),
    enable: t('governance.confirmEnable'),
    disconnect: t('governance.confirmDisconnect'),
  }
  return map[dialogAction.value]
})

const showActionDialog = (action: GovernanceAction) => {
  dialogAction.value = action
  actionForm.value = { reason: '', operator: '' }
  alsoDisconnect.value = true
  alsoBanIP.value = false
  dialogVisible.value = true
}

const executeAction = async () => {
  if (!client.value) return
  actionLoading.value = true
  const body = {
    reason: actionForm.value.reason || undefined,
    operator: actionForm.value.operator || undefined,
    banIP: dialogAction.value === 'disable' ? alsoBanIP.value || undefined : undefined,
  }

  try {
    const useDisableAndDisconnect =
      dialogAction.value === 'disable' && alsoDisconnect.value && client.value.online

    let successMsg: string
    if (useDisableAndDisconnect) {
      await disableAndDisconnect(client.value.clientID, body)
      successMsg = t('governance.disableAndDisconnectSuccess')
    } else {
      switch (dialogAction.value) {
        case 'disable':
          await disableClient(client.value.clientID, body)
          successMsg = t('governance.disableSuccess')
          break
        case 'enable':
          await enableClient(client.value.clientID, body)
          successMsg = t('governance.enableSuccess')
          break
        case 'disconnect':
          await disconnectSession(client.value.runID, body)
          successMsg = t('governance.disconnectSuccess')
          break
      }
    }

    ElMessage.success(successMsg!)
    dialogVisible.value = false

    const involvedDisconnect =
      useDisableAndDisconnect || dialogAction.value === 'disconnect'
    if (involvedDisconnect) {
      await new Promise((r) => setTimeout(r, 1500))
    }
    await refreshAfterAction()
  } catch (error: any) {
    ElMessage.error(`${t('governance.operationFailed')}: ${error.message}`)
  } finally {
    actionLoading.value = false
  }
}

const refreshAfterAction = async () => {
  const key = route.params.key as string
  try {
    const data = await getClient(key)
    client.value = new Client(data)
  } catch {
    router.replace('/clients')
  }
}

const goBack = () => {
  if (window.history.length > 1) {
    router.back()
  } else {
    router.push('/clients')
  }
}
const proxiesLoading = ref(false)
const allProxies = ref<BaseProxy[]>([])
const proxySearch = ref('')

let serverInfo: {
  vhostHTTPPort: number
  vhostHTTPSPort: number
  tcpmuxHTTPConnectPort: number
  subdomainHost: string
} | null = null

const clientProxies = computed(() => {
  if (!client.value) return []
  return allProxies.value.filter(
    (p) =>
      p.clientID === client.value!.clientID && p.user === client.value!.user,
  )
})

const filteredProxies = computed(() => {
  if (!proxySearch.value) return clientProxies.value
  const search = proxySearch.value.toLowerCase()
  return clientProxies.value.filter(
    (p) =>
      p.name.toLowerCase().includes(search) ||
      p.type.toLowerCase().includes(search),
  )
})

const totalConnections = computed(() => {
  return clientProxies.value.reduce((sum, p) => sum + p.conns, 0)
})

const fetchServerInfo = async () => {
  if (serverInfo) return serverInfo
  const res = await getServerInfo()
  serverInfo = res
  return serverInfo
}

const fetchClient = async () => {
  const key = route.params.key as string
  if (!key) {
    loading.value = false
    return
  }
  try {
    const data = await getClient(key)
    client.value = new Client(data)
  } catch (error: any) {
    if (error.status === 404) {
      client.value = null
    } else {
      ElMessage.error(`${t('clients.fetchClientFailed')}: ${error.message}`)
    }
  } finally {
    loading.value = false
  }
}

const fetchProxies = async () => {
  proxiesLoading.value = true
  const proxyTypes = ['tcp', 'udp', 'http', 'https', 'tcpmux', 'stcp', 'sudp']
  const proxies: BaseProxy[] = []
  try {
    const info = await fetchServerInfo()
    for (const type of proxyTypes) {
      try {
        const json = await getProxiesByType(type)
        if (!json.proxies) continue
        if (type === 'tcp') {
          proxies.push(...json.proxies.map((p: any) => new TCPProxy(p)))
        } else if (type === 'udp') {
          proxies.push(...json.proxies.map((p: any) => new UDPProxy(p)))
        } else if (type === 'http' && info?.vhostHTTPPort) {
          proxies.push(
            ...json.proxies.map(
              (p: any) =>
                new HTTPProxy(p, info.vhostHTTPPort, info.subdomainHost),
            ),
          )
        } else if (type === 'https' && info?.vhostHTTPSPort) {
          proxies.push(
            ...json.proxies.map(
              (p: any) =>
                new HTTPSProxy(p, info.vhostHTTPSPort, info.subdomainHost),
            ),
          )
        } else if (type === 'tcpmux' && info?.tcpmuxHTTPConnectPort) {
          proxies.push(
            ...json.proxies.map(
              (p: any) =>
                new TCPMuxProxy(
                  p,
                  info.tcpmuxHTTPConnectPort,
                  info.subdomainHost,
                ),
            ),
          )
        } else if (type === 'stcp') {
          proxies.push(...json.proxies.map((p: any) => new STCPProxy(p)))
        } else if (type === 'sudp') {
          proxies.push(...json.proxies.map((p: any) => new SUDPProxy(p)))
        }
      } catch {
        // Ignore
      }
    }
    allProxies.value = proxies
  } catch {
    // Ignore
  } finally {
    proxiesLoading.value = false
  }
}

onMounted(() => {
  fetchClient()
  fetchProxies()
})
</script>

<style scoped>
.client-detail-page {
}

/* Breadcrumb */
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  margin-bottom: 24px;
}

.breadcrumb-link {
  display: flex;
  align-items: center;
  color: var(--text-secondary);
  cursor: pointer;
  transition: color 0.2s;
  margin-right: 4px;
}

.breadcrumb-link:hover {
  color: var(--text-primary);
}

.breadcrumb-item {
  color: var(--text-secondary);
  text-decoration: none;
  transition: color 0.2s;
}

.breadcrumb-item:hover {
  color: var(--el-color-primary);
}

.breadcrumb-separator {
  color: var(--el-border-color);
}

.breadcrumb-current {
  color: var(--text-primary);
  font-weight: 500;
}

/* Card Base */
.header-card,
.proxies-card {
  background: var(--el-bg-color);
  border: 1px solid var(--header-border);
  border-radius: 12px;
  margin-bottom: 16px;
}

/* Header Card */
.header-main {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 24px;
}

.header-left {
  display: flex;
  gap: 16px;
  align-items: center;
}

.client-avatar {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 500;
  flex-shrink: 0;
}

.client-info {
  min-width: 0;
}

.client-name-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 4px;
}

.client-name {
  font-size: 20px;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0;
  line-height: 1.3;
}

.client-meta {
  display: flex;
  gap: 12px;
  font-size: 14px;
  color: var(--text-secondary);
}

.status-badge {
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
}

.status-badge.online {
  background: rgba(34, 197, 94, 0.1);
  color: #16a34a;
}

.status-badge.offline {
  background: var(--hover-bg);
  color: var(--text-secondary);
}

.status-badge.banned {
  background: rgba(239, 68, 68, 0.1);
  color: #dc2626;
}

html.dark .status-badge.online {
  background: rgba(34, 197, 94, 0.15);
  color: #4ade80;
}

html.dark .status-badge.banned {
  background: rgba(239, 68, 68, 0.15);
  color: #f87171;
}

/* Info Section */
.info-section {
  display: flex;
  flex-wrap: wrap;
  gap: 16px 32px;
  padding: 16px 24px;
}

.info-item {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.info-label {
  font-size: 13px;
  color: var(--text-secondary);
}

.info-label::after {
  content: ':';
}

.info-value {
  font-size: 13px;
  color: var(--text-primary);
  font-weight: 500;
  word-break: break-all;
}

/* Governance Section */
.governance-section {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 24px 16px;
  border-top: 1px solid var(--header-border);
  flex-wrap: wrap;
}

.dialog-confirm-text {
  margin: 0 0 16px;
  font-size: 14px;
  color: var(--el-text-color-regular);
  line-height: 1.6;
}

.disconnect-switch-item {
  margin-bottom: 12px;
}

.disconnect-switch-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.disconnect-switch-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.disconnect-switch-hint {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

/* Proxies Card */
.proxies-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  gap: 16px;
}

.proxies-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.proxies-title h2 {
  font-size: 15px;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0;
}

.proxies-count {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  background: var(--hover-bg);
  padding: 4px 10px;
  border-radius: 6px;
}

.proxy-search {
  width: 200px;
}

.proxy-search :deep(.el-input__wrapper) {
  border-radius: 6px;
}

.proxies-body {
  padding: 16px;
}

.proxies-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 40px;
  color: var(--text-secondary);
}

.empty-state {
  text-align: center;
  padding: 40px;
  color: var(--text-secondary);
}

.empty-state p {
  margin: 0;
}

/* Not Found */
.not-found {
  text-align: center;
  padding: 60px 20px;
}

.not-found h2 {
  font-size: 18px;
  font-weight: 500;
  color: var(--text-primary);
  margin: 0 0 8px;
}

.not-found p {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0 0 20px;
}

/* Responsive */
@media (max-width: 640px) {
  .header-main {
    flex-direction: column;
    gap: 16px;
  }

  .header-right {
    align-self: flex-start;
  }
}
</style>
