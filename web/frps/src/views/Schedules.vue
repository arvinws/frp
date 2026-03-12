<template>
  <div class="schedules-page">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('schedule.title') }}</h1>
        <p class="page-subtitle">{{ t('schedule.subtitle') }}</p>
      </div>
      <div class="page-actions">
        <el-button :icon="Refresh" @click="fetchTasks">
          {{ t('schedule.refresh') }}
        </el-button>
        <el-button type="primary" :icon="Plus" @click="openCreateDialog">
          {{ t('schedule.newTask') }}
        </el-button>
      </div>
    </div>

    <div v-loading="loading" class="table-card">
      <el-table v-if="tasks.length > 0" :data="tasks" stripe>
        <el-table-column prop="name" :label="t('schedule.taskName')" min-width="180">
          <template #default="{ row }">
            <div class="task-name-cell">
              <span class="task-name">{{ row.name }}</span>
              <span class="task-timezone">{{ row.timezone }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column :label="t('schedule.status')" width="110">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" effect="plain">
              {{ row.enabled ? t('common.enabled') : t('common.disabled') }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column :label="t('schedule.targets')" min-width="220">
          <template #default="{ row }">
            <div class="target-summary">
              <span>{{ row.targetCount }} {{ t('schedule.targetCount') }}</span>
              <span class="target-list">
                {{ formatTargetNames(row.targets) }}
              </span>
            </div>
          </template>
        </el-table-column>

        <el-table-column :label="t('schedule.startRule')" min-width="150">
          <template #default="{ row }">
            {{ formatRule(row.startRule) }}
          </template>
        </el-table-column>

        <el-table-column :label="t('schedule.stopRule')" min-width="150">
          <template #default="{ row }">
            {{ formatRule(row.stopRule) }}
          </template>
        </el-table-column>

        <el-table-column :label="t('schedule.nextRun')" min-width="180">
          <template #default="{ row }">
            <div class="next-run-cell">
              <div>
                <span class="next-label">{{ t('schedule.nextStart') }}:</span>
                <span>{{ formatTimestamp(row.nextStartAt, row.timezone) }}</span>
              </div>
              <div>
                <span class="next-label">{{ t('schedule.nextStop') }}:</span>
                <span>{{ formatTimestamp(row.nextStopAt, row.timezone) }}</span>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column :label="t('schedule.lastResult')" min-width="150">
          <template #default="{ row }">
            <div class="last-result-cell">
              <span>{{ formatLastResult(row) }}</span>
              <span class="last-time">{{ formatTimestamp(row.lastExecutionAt, row.timezone) }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column :label="t('schedule.actions')" width="320" fixed="right">
          <template #default="{ row }">
            <div class="action-row">
              <el-button size="small" :icon="Edit" @click="openEditDialog(row)">
                {{ t('schedule.edit') }}
              </el-button>
              <el-button
                size="small"
                :type="row.enabled ? 'warning' : 'success'"
                @click="toggleTask(row)"
              >
                {{ row.enabled ? t('schedule.disableTask') : t('schedule.enableTask') }}
              </el-button>
              <el-button size="small" :icon="VideoPlay" @click="runTask(row, 'start')">
                {{ t('schedule.runStart') }}
              </el-button>
              <el-button size="small" :icon="VideoPause" @click="runTask(row, 'stop')">
                {{ t('schedule.runStop') }}
              </el-button>
              <el-button size="small" :icon="Tickets" @click="openLogsDialog(row)">
                {{ t('schedule.logs') }}
              </el-button>
              <el-button size="small" type="danger" :icon="Delete" @click="deleteTask(row)">
                {{ t('schedule.delete') }}
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-else-if="!loading" :description="t('schedule.empty')" />
    </div>

    <el-dialog
      v-model="dialogVisible"
      :title="editingTaskId ? t('schedule.editTask') : t('schedule.newTask')"
      width="960px"
      destroy-on-close
    >
      <div class="dialog-layout">
        <div class="dialog-main">
          <el-form label-position="top">
            <div class="form-grid">
              <el-form-item :label="t('schedule.taskName')">
                <el-input v-model="form.name" :placeholder="t('schedule.taskNamePlaceholder')" />
              </el-form-item>

              <el-form-item :label="t('schedule.timezone')">
                <el-input v-model="form.timezone" placeholder="Asia/Shanghai" />
              </el-form-item>
            </div>

            <el-form-item :label="t('schedule.remark')">
              <el-input v-model="form.remark" type="textarea" :rows="2" />
            </el-form-item>

            <el-switch v-model="form.enabled" :active-text="t('common.enabled')" :inactive-text="t('common.disabled')" />

            <div class="rule-grid">
              <div class="rule-card">
                <h3>{{ t('schedule.startRule') }}</h3>
                <el-form-item :label="t('schedule.mode')">
                  <el-select v-model="form.startRule.mode">
                    <el-option :label="t('schedule.modeOnce')" value="once" />
                    <el-option :label="t('schedule.modeDaily')" value="daily" />
                    <el-option :label="t('schedule.modeWeekly')" value="weekly" />
                  </el-select>
                </el-form-item>
                <el-form-item v-if="form.startRule.mode === 'once'" :label="t('schedule.date')">
                  <el-date-picker
                    v-model="form.startRule.date"
                    type="date"
                    value-format="YYYY-MM-DD"
                    style="width: 100%"
                  />
                </el-form-item>
                <el-form-item :label="t('schedule.time')">
                  <el-time-picker
                    v-model="form.startRule.time"
                    format="HH:mm"
                    value-format="HH:mm"
                    style="width: 100%"
                  />
                </el-form-item>
                <el-form-item v-if="form.startRule.mode === 'weekly'" :label="t('schedule.daysOfWeek')">
                  <el-checkbox-group v-model="form.startRule.daysOfWeek" class="day-checkbox-group">
                    <el-checkbox v-for="option in dayOptions" :key="`start-${option.value}`" :label="option.value">
                      {{ option.label }}
                    </el-checkbox>
                  </el-checkbox-group>
                </el-form-item>
              </div>
              <div class="rule-card">
                <h3>{{ t('schedule.stopRule') }}</h3>
                <el-form-item :label="t('schedule.mode')">
                  <el-select v-model="form.stopRule.mode">
                    <el-option :label="t('schedule.modeOnce')" value="once" />
                    <el-option :label="t('schedule.modeDaily')" value="daily" />
                    <el-option :label="t('schedule.modeWeekly')" value="weekly" />
                  </el-select>
                </el-form-item>
                <el-form-item v-if="form.stopRule.mode === 'once'" :label="t('schedule.date')">
                  <el-date-picker
                    v-model="form.stopRule.date"
                    type="date"
                    value-format="YYYY-MM-DD"
                    style="width: 100%"
                  />
                </el-form-item>
                <el-form-item :label="t('schedule.time')">
                  <el-time-picker
                    v-model="form.stopRule.time"
                    format="HH:mm"
                    value-format="HH:mm"
                    style="width: 100%"
                  />
                </el-form-item>
                <el-form-item v-if="form.stopRule.mode === 'weekly'" :label="t('schedule.daysOfWeek')">
                  <el-checkbox-group v-model="form.stopRule.daysOfWeek" class="day-checkbox-group">
                    <el-checkbox v-for="option in dayOptions" :key="`stop-${option.value}`" :label="option.value">
                      {{ option.label }}
                    </el-checkbox>
                  </el-checkbox-group>
                </el-form-item>
              </div>
            </div>
          </el-form>
        </div>

        <div class="dialog-side">
          <div class="selector-header">
            <h3>{{ t('schedule.selectTargets') }}</h3>
            <span class="selector-count">{{ selectedProxyNames.length }}</span>
          </div>

          <div class="selector-filters">
            <el-input v-model="proxyFilters.keyword" :placeholder="t('schedule.searchProxy')" clearable />
            <div class="selector-filter-row">
              <el-select v-model="proxyFilters.type" clearable :placeholder="t('schedule.filterType')">
                <el-option label="TCP" value="tcp" />
                <el-option label="UDP" value="udp" />
                <el-option label="HTTP" value="http" />
                <el-option label="HTTPS" value="https" />
                <el-option label="TCPMUX" value="tcpmux" />
                <el-option label="STCP" value="stcp" />
                <el-option label="XTCP" value="xtcp" />
                <el-option label="SUDP" value="sudp" />
              </el-select>
              <el-select v-model="proxyFilters.status" clearable :placeholder="t('schedule.filterStatus')">
                <el-option :label="t('common.online')" value="online" />
                <el-option :label="t('common.offline')" value="offline" />
              </el-select>
            </div>
          </div>

          <div v-loading="proxyOptionsLoading" class="proxy-selector">
            <el-checkbox-group v-model="selectedProxyNames" class="proxy-checkbox-group">
              <label v-for="option in filteredProxyOptions" :key="option.proxyName" class="proxy-option-card">
                <el-checkbox :label="option.proxyName">
                  <div class="proxy-option-content">
                    <div class="proxy-option-top">
                      <span class="proxy-option-name">{{ option.proxyName }}</span>
                      <el-tag size="small" effect="plain">{{ option.type?.toUpperCase() }}</el-tag>
                    </div>
                    <div class="proxy-option-meta">
                      <span>{{ option.user || '-' }}</span>
                      <span>{{ option.clientID || '-' }}</span>
                      <span>{{ option.status }}</span>
                      <span v-if="option.disabled">{{ t('common.disabled') }}</span>
                    </div>
                  </div>
                </el-checkbox>
              </label>
            </el-checkbox-group>
            <el-empty v-if="!proxyOptionsLoading && filteredProxyOptions.length === 0" :description="t('schedule.noProxyOptions')" />
          </div>
        </div>
      </div>

      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('proxies.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="saveTask">
          {{ t('schedule.save') }}
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="logsVisible" :title="t('schedule.logs')" width="900px">
      <div v-loading="logsLoading">
        <el-table v-if="logs.length > 0" :data="logs" stripe>
          <el-table-column prop="executedAt" :label="t('schedule.executedAt')" min-width="170">
            <template #default="{ row }">
              {{ formatTimestamp(row.executedAt, activeLogsTask?.timezone) }}
            </template>
          </el-table-column>
          <el-table-column prop="action" :label="t('schedule.action')" width="90" />
          <el-table-column prop="targetProxyName" :label="t('schedule.proxy')" min-width="160" />
          <el-table-column prop="result" :label="t('schedule.resultLabel')" min-width="150" />
          <el-table-column prop="error" :label="t('schedule.error')" min-width="220" />
        </el-table>
        <el-empty v-else-if="!logsLoading" :description="t('schedule.noLogs')" />
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus,
  Refresh,
  Delete,
  Edit,
  Tickets,
  VideoPlay,
  VideoPause,
} from '@element-plus/icons-vue'
import {
  createSchedule,
  deleteSchedule,
  disableSchedule,
  enableSchedule,
  getProxyOptions,
  getScheduleLogs,
  getSchedules,
  runSchedule,
  updateSchedule,
} from '../api/schedule'
import type {
  ProxyOption,
  ScheduleExecutionLog,
  ScheduleRule,
  ScheduleTask,
  ScheduleTaskPayload,
  ScheduleTarget,
} from '../types/schedule'
import { useI18n } from '../i18n'

const { t } = useI18n()

const loading = ref(false)
const saving = ref(false)
const tasks = ref<ScheduleTask[]>([])
const dialogVisible = ref(false)
const logsVisible = ref(false)
const logsLoading = ref(false)
const logs = ref<ScheduleExecutionLog[]>([])
const activeLogsTask = ref<ScheduleTask | null>(null)
const editingTaskId = ref<string | null>(null)

const proxyOptions = ref<ProxyOption[]>([])
const proxyOptionsLoading = ref(false)
const selectedProxyNames = ref<string[]>([])
const proxyFilters = reactive({
  keyword: '',
  type: '',
  status: '',
})

const form = reactive<ScheduleTaskPayload>({
  name: '',
  enabled: true,
  timezone: 'Asia/Shanghai',
  targets: [],
  startRule: {
    mode: 'daily',
    time: '09:00',
    daysOfWeek: [1, 2, 3, 4, 5],
  },
  stopRule: {
    mode: 'daily',
    time: '18:00',
    daysOfWeek: [1, 2, 3, 4, 5],
  },
  remark: '',
})

const dayOptions = computed(() => [
  { label: t('schedule.dayMon'), value: 1 },
  { label: t('schedule.dayTue'), value: 2 },
  { label: t('schedule.dayWed'), value: 3 },
  { label: t('schedule.dayThu'), value: 4 },
  { label: t('schedule.dayFri'), value: 5 },
  { label: t('schedule.daySat'), value: 6 },
  { label: t('schedule.daySun'), value: 7 },
])

const filteredProxyOptions = computed(() => {
  const keyword = proxyFilters.keyword.trim().toLowerCase()
  return proxyOptions.value.filter((option) => {
    if (proxyFilters.type && option.type !== proxyFilters.type) {
      return false
    }
    if (proxyFilters.status && option.status !== proxyFilters.status) {
      return false
    }
    if (!keyword) {
      return true
    }
    const text = [option.proxyName, option.user, option.clientID, option.type]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()
    return text.includes(keyword)
  })
})

const fetchTasks = async () => {
  loading.value = true
  try {
    const response = await getSchedules()
    tasks.value = response.tasks || []
  } catch (error: any) {
    ElMessage.error(`${t('schedule.fetchFailed')}: ${error.message}`)
  } finally {
    loading.value = false
  }
}

const fetchProxyOptions = async () => {
  proxyOptionsLoading.value = true
  try {
    proxyOptions.value = await getProxyOptions()
  } catch (error: any) {
    ElMessage.error(`${t('schedule.fetchProxyOptionsFailed')}: ${error.message}`)
  } finally {
    proxyOptionsLoading.value = false
  }
}

const fetchLogs = async (task: ScheduleTask) => {
  logsLoading.value = true
  activeLogsTask.value = task
  try {
    const response = await getScheduleLogs(task.id)
    logs.value = response.logs || []
  } catch (error: any) {
    ElMessage.error(`${t('schedule.fetchLogsFailed')}: ${error.message}`)
  } finally {
    logsLoading.value = false
  }
}

const resetForm = () => {
  editingTaskId.value = null
  form.name = ''
  form.enabled = true
  form.timezone = 'Asia/Shanghai'
  form.targets = []
  form.startRule = { mode: 'daily', time: '09:00', daysOfWeek: [1, 2, 3, 4, 5] }
  form.stopRule = { mode: 'daily', time: '18:00', daysOfWeek: [1, 2, 3, 4, 5] }
  form.remark = ''
  selectedProxyNames.value = []
  proxyFilters.keyword = ''
  proxyFilters.type = ''
  proxyFilters.status = ''
}

const mergeTaskTargetsIntoOptions = (targets: ScheduleTarget[]) => {
  const existing = new Map(proxyOptions.value.map((option) => [option.proxyName, option]))
  for (const target of targets) {
    if (!existing.has(target.proxyName)) {
      proxyOptions.value.push({
        proxyName: target.proxyName,
        displayName: target.proxyName,
        type: target.type,
        user: target.user,
        clientID: target.clientID,
        status: 'offline',
        disabled: false,
      })
    }
  }
}

const openCreateDialog = async () => {
  resetForm()
  dialogVisible.value = true
  await fetchProxyOptions()
}

const openEditDialog = async (task: ScheduleTask) => {
  resetForm()
  editingTaskId.value = task.id
  form.name = task.name
  form.enabled = task.enabled
  form.timezone = task.timezone
  form.startRule = cloneRule(task.startRule)
  form.stopRule = cloneRule(task.stopRule)
  form.remark = task.remark || ''
  selectedProxyNames.value = task.targets.map((target) => target.proxyName)
  dialogVisible.value = true
  await fetchProxyOptions()
  mergeTaskTargetsIntoOptions(task.targets)
}

const openLogsDialog = async (task: ScheduleTask) => {
  logsVisible.value = true
  logs.value = []
  await fetchLogs(task)
}

const buildPayload = (): ScheduleTaskPayload | null => {
  if (!form.name.trim()) {
    ElMessage.warning(t('schedule.validationTaskName'))
    return null
  }
  if (!form.timezone.trim()) {
    ElMessage.warning(t('schedule.validationTimezone'))
    return null
  }
  if (selectedProxyNames.value.length === 0) {
    ElMessage.warning(t('schedule.validationTargets'))
    return null
  }

  const payload: ScheduleTaskPayload = {
    name: form.name.trim(),
    enabled: form.enabled,
    timezone: form.timezone.trim(),
    targets: selectedProxyNames.value.map((proxyName) => {
      const option = proxyOptions.value.find((item) => item.proxyName === proxyName)
      return {
        proxyName,
        user: option?.user,
        clientID: option?.clientID,
        type: option?.type,
      }
    }),
    startRule: sanitizeRule(form.startRule),
    stopRule: sanitizeRule(form.stopRule),
    remark: form.remark?.trim(),
  }

  for (const rule of [payload.startRule, payload.stopRule]) {
    if (!rule.time) {
      ElMessage.warning(t('schedule.validationRuleTime'))
      return null
    }
    if (rule.mode === 'once' && !rule.date) {
      ElMessage.warning(t('schedule.validationRuleDate'))
      return null
    }
    if (rule.mode === 'weekly' && (!rule.daysOfWeek || rule.daysOfWeek.length === 0)) {
      ElMessage.warning(t('schedule.validationRuleDays'))
      return null
    }
  }

  return payload
}

const saveTask = async () => {
  const payload = buildPayload()
  if (!payload) {
    return
  }

  saving.value = true
  try {
    if (editingTaskId.value) {
      await updateSchedule(editingTaskId.value, payload)
      ElMessage.success(t('schedule.updateSuccess'))
    } else {
      await createSchedule(payload)
      ElMessage.success(t('schedule.createSuccess'))
    }
    dialogVisible.value = false
    fetchTasks()
  } catch (error: any) {
    ElMessage.error(`${t('schedule.saveFailed')}: ${error.message}`)
  } finally {
    saving.value = false
  }
}

const toggleTask = async (task: ScheduleTask) => {
  try {
    if (task.enabled) {
      await disableSchedule(task.id)
      ElMessage.success(t('schedule.disableSuccess'))
    } else {
      await enableSchedule(task.id)
      ElMessage.success(t('schedule.enableSuccess'))
    }
    fetchTasks()
  } catch (error: any) {
    ElMessage.error(`${t('schedule.saveFailed')}: ${error.message}`)
  }
}

const runTask = async (task: ScheduleTask, action: 'start' | 'stop') => {
  try {
    await runSchedule(task.id, action)
    ElMessage.success(
      action === 'start' ? t('schedule.runStartSuccess') : t('schedule.runStopSuccess'),
    )
    fetchTasks()
  } catch (error: any) {
    ElMessage.error(`${t('schedule.runFailed')}: ${error.message}`)
  }
}

const deleteTask = async (task: ScheduleTask) => {
  try {
    await ElMessageBox.confirm(
      t('schedule.confirmDelete', { name: task.name }),
      t('schedule.delete'),
      { type: 'warning' },
    )
    await deleteSchedule(task.id)
    ElMessage.success(t('schedule.deleteSuccess'))
    fetchTasks()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(`${t('schedule.saveFailed')}: ${error.message}`)
    }
  }
}

const formatTimestamp = (timestamp?: number, timezone?: string) => {
  if (!timestamp) {
    return '-'
  }
  const date = new Date(timestamp * 1000)
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    timeZone: timezone || undefined,
  }).format(date)
}

const formatRule = (rule: ScheduleRule) => {
  if (rule.mode === 'once') {
    return `${rule.date || '-'} ${rule.time}`
  }
  if (rule.mode === 'daily') {
    return `${t('schedule.everyDay')} ${rule.time}`
  }
  const labels = (rule.daysOfWeek || [])
    .map((day) => dayOptions.value.find((option) => option.value === day)?.label || day)
    .join(', ')
  return `${labels} ${rule.time}`
}

const formatLastResult = (task: ScheduleTask) => {
  if (!task.lastExecutionResult) {
    return '-'
  }
  return t(`schedule.result.${task.lastExecutionResult}`)
}

const formatTargetNames = (targets: ScheduleTarget[]) => {
  return targets.map((target) => target.proxyName).join(', ')
}

const cloneRule = (rule: ScheduleRule): ScheduleRule => ({
  mode: rule.mode,
  date: rule.date,
  time: rule.time,
  daysOfWeek: rule.daysOfWeek ? [...rule.daysOfWeek] : [],
})

const sanitizeRule = (rule: ScheduleRule): ScheduleRule => {
  const next: ScheduleRule = {
    mode: rule.mode,
    time: rule.time,
  }
  if (rule.mode === 'once') {
    next.date = rule.date
  }
  if (rule.mode === 'weekly') {
    next.daysOfWeek = [...(rule.daysOfWeek || [])].sort((a, b) => a - b)
  }
  return next
}

fetchTasks()
</script>

<style scoped>
.schedules-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  gap: 24px;
  align-items: flex-start;
}

.page-title {
  margin: 0;
  font-size: 32px;
  line-height: 1.1;
}

.page-subtitle {
  margin: 8px 0 0;
  color: var(--text-secondary);
}

.page-actions {
  display: flex;
  gap: 12px;
}

.table-card {
  background: var(--el-bg-color);
  border: 1px solid var(--header-border);
  border-radius: 16px;
  padding: 16px;
}

.task-name-cell,
.last-result-cell,
.next-run-cell,
.target-summary {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.task-name {
  font-weight: 600;
}

.task-timezone,
.last-time,
.target-list,
.next-label {
  color: var(--text-secondary);
  font-size: 12px;
}

.action-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.dialog-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(320px, 0.8fr);
  gap: 20px;
}

.dialog-main,
.dialog-side {
  border: 1px solid var(--header-border);
  border-radius: 16px;
  padding: 18px;
  background: var(--el-bg-color-page);
}

.form-grid,
.selector-filter-row,
.rule-grid {
  display: grid;
  gap: 16px;
}

.form-grid,
.selector-filter-row {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.rule-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-top: 20px;
}

.rule-card h3,
.selector-header h3 {
  margin: 0 0 16px;
}

.selector-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.selector-count {
  font-size: 12px;
  padding: 4px 10px;
  border-radius: 999px;
  background: var(--hover-bg);
}

.selector-filters {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 16px;
}

.proxy-selector {
  max-height: 520px;
  overflow: auto;
}

.proxy-checkbox-group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.proxy-option-card {
  border: 1px solid var(--header-border);
  border-radius: 12px;
  padding: 12px 14px;
  background: var(--el-bg-color);
}

.proxy-option-card :deep(.el-checkbox) {
  width: 100%;
}

.proxy-option-content {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.proxy-option-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.proxy-option-name {
  font-weight: 600;
}

.proxy-option-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 12px;
  color: var(--text-secondary);
}

.day-checkbox-group {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 16px;
}

@media (max-width: 960px) {
  .dialog-layout,
  .rule-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .page-header,
  .page-actions,
  .form-grid,
  .selector-filter-row {
    grid-template-columns: 1fr;
    display: grid;
  }

  .page-actions {
    width: 100%;
  }

  .page-actions :deep(.el-button) {
    width: 100%;
  }
}
</style>
