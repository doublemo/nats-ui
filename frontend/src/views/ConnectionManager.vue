<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import {
  activateConnection,
  batchDeleteConnections,
  createConnection,
  deleteConnection,
  discoverMonitorEndpoints,
  getActiveConnectionId,
  getConnections,
  importConnections,
  previewImportConnections,
  probeConnection,
  setActiveConnectionId,
  testConnection,
  updateConnection,
} from '../api/nats'

const CONNECTION_MANAGER_STATE_KEY = 'nats-ui-connection-manager-state'
const UI_UNASSIGNED_GROUP = '__UNASSIGNED__'
const BACKEND_UNASSIGNED_GROUP = '未分组'
const STATUS_CURRENT = '当前连接'
const STATUS_UNCHECKED = '未检测'
const STATUS_CONNECTED = 'CONNECTED'
const STATUS_DISCONNECTED = 'DISCONNECTED'
const STATUS_RECONNECTING = 'RECONNECTING'
const STATUS_ERROR = 'ERROR'

const { t, locale } = useI18n()
const loading = ref(false)
const testing = ref(false)
const batchTesting = ref(false)
const importing = ref(false)
const dialogVisible = ref(false)
const editingId = ref('')
const activeId = ref(getActiveConnectionId())
const connections = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(12)
const filters = reactive({
  keyword: '',
  group: '',
  tag: '',
  status: '',
})
const activeGroups = ref([])
const selectedIds = ref([])
const form = reactive({
  name: '',
  group: '',
  tags: '',
  natsUrls: '',
  monitorEndpoints: '',
  username: '',
  password: '',
  token: '',
})

const groupOptions = computed(() => {
  const groups = [...new Set(connections.value.map((item) => item.group || UI_UNASSIGNED_GROUP))]
  return groups.map((value) => ({
    value,
    label: formatGroupName(value),
  }))
})

const tagOptions = computed(() => [...new Set(connections.value.flatMap((item) => item.tags || []))])

const statusOptions = computed(() => [
  { value: STATUS_CURRENT, label: t('connections.status.current') },
  { value: STATUS_CONNECTED, label: t('connections.status.connected') },
  { value: STATUS_DISCONNECTED, label: t('connections.status.disconnected') },
  { value: STATUS_RECONNECTING, label: t('connections.status.reconnecting') },
  { value: STATUS_ERROR, label: t('connections.status.error') },
  { value: STATUS_UNCHECKED, label: t('connections.status.unchecked') },
])

const groupedConnections = computed(() => {
  const keyword = filters.keyword.trim().toLowerCase()
  const groups = new Map()

  for (const item of connections.value) {
    const targetGroup = item.group || UI_UNASSIGNED_GROUP
    if (filters.group && targetGroup !== filters.group) {
      continue
    }
    if (filters.tag && !(item.tags || []).includes(filters.tag)) {
      continue
    }
    const displayStatus = item.id === activeId.value ? STATUS_CURRENT : (item.status || STATUS_UNCHECKED)
    if (filters.status && displayStatus !== filters.status) {
      continue
    }

    const searchText = [
      item.name,
      item.group,
      ...(item.tags || []),
      ...(item.natsUrls || []),
      ...(item.monitorEndpoints || []),
      item.connectedUrl,
      item.status,
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()

    if (keyword && !searchText.includes(keyword)) {
      continue
    }

    if (!groups.has(targetGroup)) {
      groups.set(targetGroup, [])
    }
    groups.get(targetGroup).push(item)
  }

  const sortLocale = locale.value === 'zh-CN' ? 'zh-CN' : 'en-US'

  return Array.from(groups.entries())
    .sort(([a], [b]) => formatGroupName(a).localeCompare(formatGroupName(b), sortLocale))
    .map(([name, items]) => ({
      name,
      count: items.length,
      items: items.sort((a, b) => a.name.localeCompare(b.name, sortLocale)),
    }))
})

restoreViewState()

function normalizeGroupValue(value) {
  if (value === BACKEND_UNASSIGNED_GROUP) {
    return UI_UNASSIGNED_GROUP
  }
  return value || ''
}

function getRequestGroupValue(value) {
  if (value === UI_UNASSIGNED_GROUP) {
    return BACKEND_UNASSIGNED_GROUP
  }
  return value || undefined
}

function formatGroupName(value) {
  return value === UI_UNASSIGNED_GROUP ? t('connections.unassignedGroup') : value
}

function formatConnectionStatus(status, isActive) {
  if (isActive) return t('connections.status.current')

  const statusMap = {
    [STATUS_CONNECTED]: 'connections.status.connected',
    [STATUS_DISCONNECTED]: 'connections.status.disconnected',
    [STATUS_RECONNECTING]: 'connections.status.reconnecting',
    [STATUS_ERROR]: 'connections.status.error',
    [STATUS_UNCHECKED]: 'connections.status.unchecked',
  }

  const key = statusMap[status || STATUS_UNCHECKED]
  return key ? t(key) : status
}

async function loadConnections() {
  loading.value = true
  try {
    const data = await getConnections({
      page: page.value,
      pageSize: pageSize.value,
      keyword: filters.keyword || undefined,
      group: getRequestGroupValue(filters.group),
      tag: filters.tag || undefined,
      status: filters.status || undefined,
    })
    connections.value = data.items
    activeId.value = data.activeId
    total.value = data.total
    if (!activeGroups.value.length) {
      activeGroups.value = [...new Set(data.items.map((item) => item.group || UI_UNASSIGNED_GROUP))]
    }
    if (data.activeId) {
      setActiveConnectionId(data.activeId)
    }
  } finally {
    loading.value = false
  }
}

function resetForm() {
  editingId.value = ''
  form.name = ''
  form.group = ''
  form.tags = ''
  form.natsUrls = ''
  form.monitorEndpoints = ''
  form.username = ''
  form.password = ''
  form.token = ''
}

function resetFilters() {
  filters.keyword = ''
  filters.group = ''
  filters.tag = ''
  filters.status = ''
  page.value = 1
}

function restoreViewState() {
  try {
    const raw = window.localStorage.getItem(CONNECTION_MANAGER_STATE_KEY)
    if (!raw) return
    const saved = JSON.parse(raw)
    filters.keyword = saved.keyword || ''
    filters.group = normalizeGroupValue(saved.group)
    filters.tag = saved.tag || ''
    filters.status = saved.status || ''
    page.value = saved.page || 1
    pageSize.value = saved.pageSize || 12
    activeGroups.value = Array.isArray(saved.activeGroups) ? saved.activeGroups.map(normalizeGroupValue) : []
  } catch {
    window.localStorage.removeItem(CONNECTION_MANAGER_STATE_KEY)
  }
}

function persistViewState() {
  window.localStorage.setItem(
    CONNECTION_MANAGER_STATE_KEY,
    JSON.stringify({
      keyword: filters.keyword,
      group: filters.group,
      tag: filters.tag,
      status: filters.status,
      page: page.value,
      pageSize: pageSize.value,
      activeGroups: activeGroups.value,
    }),
  )
}

function isSelected(id) {
  return selectedIds.value.includes(id)
}

function toggleSelected(id, checked) {
  if (checked) {
    if (!selectedIds.value.includes(id)) {
      selectedIds.value = [...selectedIds.value, id]
    }
    return
  }
  selectedIds.value = selectedIds.value.filter((item) => item !== id)
}

function openCreate() {
  resetForm()
  dialogVisible.value = true
}

function openEdit(row) {
  editingId.value = row.id
  form.name = row.name
  form.group = row.group || ''
  form.tags = (row.tags || []).join(', ')
  form.natsUrls = row.natsUrls.join(', ')
  form.monitorEndpoints = row.monitorEndpoints.join(', ')
  form.username = row.username || ''
  form.password = ''
  form.token = ''
  dialogVisible.value = true
}

function buildPayload() {
  return {
    name: form.name,
    group: form.group,
    tags: form.tags.split(',').map((item) => item.trim()).filter(Boolean),
    natsUrls: form.natsUrls.split(',').map((item) => item.trim()).filter(Boolean),
    monitorEndpoints: form.monitorEndpoints.split(',').map((item) => item.trim()).filter(Boolean),
    username: form.username,
    password: form.password,
    token: form.token,
  }
}

async function fillMonitorEndpoints() {
  const natsUrls = form.natsUrls.split(',').map((item) => item.trim()).filter(Boolean)
  const data = await discoverMonitorEndpoints(natsUrls)
  form.monitorEndpoints = data.monitorEndpoints.join(', ')
  ElMessage.success(t('connections.messages.discoverSuccess'))
}

async function submit() {
  const payload = buildPayload()
  if (editingId.value) {
    await updateConnection(editingId.value, payload)
    ElMessage.success(t('connections.messages.updatedSuccess'))
  } else {
    await createConnection(payload)
    ElMessage.success(t('connections.messages.createdSuccess'))
  }
  dialogVisible.value = false
  await loadConnections()
}

async function runProbe() {
  testing.value = true
  try {
    if (editingId.value) {
      await testConnection(editingId.value)
    } else {
      await probeConnection(buildPayload())
    }
    ElMessage.success(t('connections.messages.testSuccess'))
    await loadConnections()
  } finally {
    testing.value = false
  }
}

async function runSavedTest(id) {
  await testConnection(id)
  ElMessage.success(t('connections.messages.testSuccess'))
  await loadConnections()
}

function exportConnections() {
  const payload = {
    exportedAt: new Date().toISOString(),
    items: groupedConnections.value.flatMap((group) =>
      group.items.map((item) => ({
        name: item.name,
        group: item.group || '',
        tags: item.tags || [],
        natsUrls: item.natsUrls || [],
        monitorEndpoints: item.monitorEndpoints || [],
        username: item.username || '',
      })),
    ),
  }
  const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' })
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `nats-connections-${Date.now()}.json`
  link.click()
  window.URL.revokeObjectURL(url)
  ElMessage.success(t('connections.messages.exportSuccess'))
}

async function handleImport(event) {
  const file = event.target.files?.[0]
  if (!file) {
    return
  }
  importing.value = true
  try {
    const text = await file.text()
    const payload = JSON.parse(text)
    const items = Array.isArray(payload) ? payload : (payload.items || [])
    const preview = await previewImportConnections(items)
    let strategy = 'skip'
    if (preview.conflicts > 0) {
      try {
        await ElMessageBox.confirm(
          t('connections.messages.importPreviewMessage', {
            conflicts: preview.conflicts,
            newCount: preview.newCount,
          }),
          t('connections.messages.importPreviewTitle'),
          {
            confirmButtonText: t('connections.messages.overwriteImport'),
            cancelButtonText: t('connections.messages.skipConflict'),
            distinguishCancelAndClose: true,
            type: 'warning',
          },
        )
        strategy = 'overwrite'
      } catch (action) {
        if (action !== 'cancel') {
          return
        }
        strategy = 'skip'
      }
    }
    const result = await importConnections({ items, strategy })
    await loadConnections()
    ElMessage.success(
      t('connections.messages.importSummary', {
        created: result.created,
        updated: result.updated,
        skipped: result.skipped,
        failed: result.failed,
      }),
    )
  } finally {
    importing.value = false
    event.target.value = ''
  }
}

async function runBatchTest() {
  if (!connections.value.length) {
    return
  }
  batchTesting.value = true
  let passed = 0
  let failed = 0
  try {
    for (const item of connections.value) {
      try {
        await testConnection(item.id)
        passed += 1
      } catch {
        failed += 1
      }
    }
    await loadConnections()
    ElMessage.success(t('connections.messages.batchTestSummary', { passed, failed }))
  } finally {
    batchTesting.value = false
  }
}

function expandAllGroups() {
  activeGroups.value = groupedConnections.value.map((item) => item.name)
}

function collapseAllGroups() {
  activeGroups.value = []
}

async function switchActive(id) {
  await activateConnection(id)
  setActiveConnectionId(id)
  activeId.value = id
  ElMessage.success(t('connections.messages.switchSuccess'))
  await loadConnections()
}

async function removeConnection(id, name) {
  await ElMessageBox.confirm(t('connections.messages.deleteConfirm', { name }), t('common.prompt'), { type: 'warning' })
  const data = await deleteConnection(id)
  setActiveConnectionId(data.activeId || '')
  ElMessage.success(t('connections.messages.deletedSuccess'))
  await loadConnections()
}

async function removeSelectedConnections() {
  if (!selectedIds.value.length) {
    ElMessage.warning(t('connections.messages.selectDeleteWarning'))
    return
  }
  await ElMessageBox.confirm(
    t('connections.messages.batchDeleteConfirm', { count: selectedIds.value.length }),
    t('common.prompt'),
    { type: 'warning' },
  )
  const result = await batchDeleteConnections(selectedIds.value)
  selectedIds.value = []
  setActiveConnectionId(result.activeId || '')
  await loadConnections()
  ElMessage.success(t('connections.messages.batchDeleteSummary', { deleted: result.deleted, failed: result.failed }))
}

onMounted(loadConnections)

watch(
  () => [filters.keyword, filters.group, filters.tag, filters.status],
  async () => {
    page.value = 1
    selectedIds.value = []
    await loadConnections()
  },
)

watch(
  () => [filters.keyword, filters.group, filters.tag, filters.status, page.value, pageSize.value, activeGroups.value],
  persistViewState,
  { deep: true },
)

watch(pageSize, async () => {
  page.value = 1
  selectedIds.value = []
  await loadConnections()
})

watch(page, async () => {
  selectedIds.value = []
  await loadConnections()
})
</script>

<template>
  <div class="page-grid">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>{{ t('connections.title') }}</span>
          <div class="connection-toolbar">
            <el-button :loading="batchTesting" @click="runBatchTest">{{ t('connections.batchTest') }}</el-button>
            <el-button @click="exportConnections">{{ t('connections.exportConfig') }}</el-button>
            <label class="upload-button">
              <input type="file" accept="application/json" :disabled="importing" @change="handleImport" />
              <span>{{ importing ? t('connections.importing') : t('connections.importConfig') }}</span>
            </label>
            <el-button type="danger" plain @click="removeSelectedConnections">{{ t('connections.batchDelete') }}</el-button>
            <el-button @click="expandAllGroups">{{ t('connections.expandAll') }}</el-button>
            <el-button @click="collapseAllGroups">{{ t('connections.collapseAll') }}</el-button>
            <el-button type="primary" @click="openCreate">{{ t('connections.createConnection') }}</el-button>
          </div>
        </div>
      </template>

      <div class="connection-filters">
        <el-input v-model="filters.keyword" :placeholder="t('connections.searchPlaceholder')" clearable />
        <el-select v-model="filters.group" :placeholder="t('connections.allGroups')" clearable>
          <el-option v-for="item in groupOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-select v-model="filters.tag" :placeholder="t('connections.allTags')" clearable>
          <el-option v-for="item in tagOptions" :key="item" :label="item" :value="item" />
        </el-select>
        <el-select v-model="filters.status" :placeholder="t('connections.allStatuses')" clearable>
          <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-button @click="resetFilters">{{ t('connections.resetFilters') }}</el-button>
      </div>
    </el-card>

    <el-collapse v-model="activeGroups" class="connection-groups">
      <el-collapse-item v-for="group in groupedConnections" :key="group.name" :name="group.name">
        <template #title>
          <div class="connection-group-title">
            <strong>{{ formatGroupName(group.name) }}</strong>
            <el-tag size="small" type="info">{{ group.count }}</el-tag>
          </div>
        </template>

        <div class="connection-card-grid">
          <el-card v-for="row in group.items" :key="row.id" shadow="hover" class="connection-card">
            <div class="connection-card-head">
              <div class="connection-card-title-wrap">
                <el-checkbox :model-value="isSelected(row.id)" @change="(checked) => toggleSelected(row.id, checked)" />
                <div>
                  <h3>{{ row.name }}</h3>
                  <p>{{ row.natsUrls.join(', ') }}</p>
                </div>
              </div>
              <el-tag :type="row.id === activeId ? 'success' : 'info'">
                {{ formatConnectionStatus(row.status, row.id === activeId) }}
              </el-tag>
            </div>

            <div class="connection-meta">
              <span>{{ t('connections.monitorEndpoints') }}: {{ row.monitorEndpoints.join(', ') || '-' }}</span>
              <span>{{ t('connections.currentNode') }}: {{ row.connectedUrl || '-' }}</span>
              <span>{{ t('connections.auth') }}: {{ row.hasPassword || row.hasToken ? t('connections.configured') : t('connections.none') }}</span>
            </div>

            <el-space v-if="row.tags?.length" wrap class="connection-tags">
              <el-tag v-for="tag in row.tags" :key="tag" size="small">{{ tag }}</el-tag>
            </el-space>

            <el-alert
              v-if="row.lastError"
              :title="row.lastError"
              type="error"
              show-icon
              :closable="false"
              class="connection-error"
            />

            <div class="connection-card-actions">
              <el-button type="primary" plain @click="switchActive(row.id)">{{ t('common.switch') }}</el-button>
              <el-button plain @click="runSavedTest(row.id)">{{ t('common.test') }}</el-button>
              <el-button plain @click="openEdit(row)">{{ t('common.edit') }}</el-button>
              <el-button type="danger" plain @click="removeConnection(row.id, row.name)">{{ t('common.delete') }}</el-button>
            </div>
          </el-card>
        </div>
      </el-collapse-item>
    </el-collapse>

    <div v-if="total > 0" class="table-pagination">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        layout="total, sizes, prev, pager, next"
        :total="total"
        :page-sizes="[12, 24, 48]"
      />
    </div>

    <el-empty v-if="!groupedConnections.length" :description="t('connections.empty')" />
  </div>

  <el-dialog v-model="dialogVisible" :title="editingId ? t('connections.dialog.editTitle') : t('connections.dialog.createTitle')" width="640px">
    <el-form label-width="110px">
      <el-form-item :label="t('connections.fields.name')">
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item :label="t('connections.fields.group')">
        <el-input v-model="form.group" :placeholder="t('connections.placeholders.group')" />
      </el-form-item>
      <el-form-item :label="t('connections.fields.tags')">
        <el-input v-model="form.tags" :placeholder="t('connections.placeholders.tags')" />
      </el-form-item>
      <el-form-item :label="t('connections.fields.natsUrls')">
        <el-input v-model="form.natsUrls" :placeholder="t('connections.placeholders.natsUrls')" />
      </el-form-item>
      <el-form-item :label="t('connections.fields.monitorUrls')">
        <el-input v-model="form.monitorEndpoints" :placeholder="t('connections.placeholders.monitorUrls')">
          <template #append>
            <el-button @click="fillMonitorEndpoints">{{ t('connections.actions.autoDiscover') }}</el-button>
          </template>
        </el-input>
      </el-form-item>
      <el-form-item :label="t('connections.fields.username')">
        <el-input v-model="form.username" />
      </el-form-item>
      <el-form-item :label="t('connections.fields.password')">
        <el-input
          v-model="form.password"
          type="password"
          show-password
          :placeholder="editingId ? t('connections.placeholders.keepUnchanged') : ''"
        />
      </el-form-item>
      <el-form-item :label="t('connections.fields.token')">
        <el-input v-model="form.token" :placeholder="editingId ? t('connections.placeholders.keepUnchanged') : ''" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
      <el-button :loading="testing" @click="runProbe">{{ t('common.test') }}</el-button>
      <el-button type="primary" @click="submit">{{ t('common.save') }}</el-button>
    </template>
  </el-dialog>
</template>
