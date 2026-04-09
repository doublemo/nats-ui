<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
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

restoreViewState()

const groupOptions = computed(() => {
  return [...new Set(connections.value.map((item) => item.group || '未分组'))]
})

const tagOptions = computed(() => {
  return [...new Set(connections.value.flatMap((item) => item.tags || []))]
})

const statusOptions = ['当前连接', 'CONNECTED', 'DISCONNECTED', 'RECONNECTING', 'ERROR', '未检测']

const groupedConnections = computed(() => {
  const keyword = filters.keyword.trim().toLowerCase()
  const groups = new Map()

  for (const item of connections.value) {
    const targetGroup = item.group || '未分组'
    if (filters.group && targetGroup !== filters.group) {
      continue
    }
    if (filters.tag && !(item.tags || []).includes(filters.tag)) {
      continue
    }
    const displayStatus = item.id === activeId.value ? '当前连接' : (item.status || '未检测')
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

  return Array.from(groups.entries())
    .sort(([a], [b]) => a.localeCompare(b, 'zh-CN'))
    .map(([name, items]) => ({
      name,
      count: items.length,
      items: items.sort((a, b) => a.name.localeCompare(b.name, 'zh-CN')),
    }))
})

async function loadConnections() {
  loading.value = true
  try {
    const data = await getConnections({
      page: page.value,
      pageSize: pageSize.value,
      keyword: filters.keyword || undefined,
      group: filters.group || undefined,
      tag: filters.tag || undefined,
      status: filters.status || undefined,
    })
    connections.value = data.items
    activeId.value = data.activeId
    total.value = data.total
    if (!activeGroups.value.length) {
      activeGroups.value = [...new Set(data.items.map((item) => item.group || '未分组'))]
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
    filters.group = saved.group || ''
    filters.tag = saved.tag || ''
    filters.status = saved.status || ''
    page.value = saved.page || 1
    pageSize.value = saved.pageSize || 12
    activeGroups.value = Array.isArray(saved.activeGroups) ? saved.activeGroups : []
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
  ElMessage.success('已根据 NATS 地址推导监控地址')
}

async function submit() {
  const payload = buildPayload()
  if (editingId.value) {
    await updateConnection(editingId.value, payload)
    ElMessage.success('连接已更新')
  } else {
    await createConnection(payload)
    ElMessage.success('连接已创建')
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
    ElMessage.success('连接测试通过')
    await loadConnections()
  } finally {
    testing.value = false
  }
}

async function runSavedTest(id) {
  await testConnection(id)
  ElMessage.success('连接测试通过')
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
  ElMessage.success('连接配置已导出')
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
          `检测到 ${preview.conflicts} 个重名连接，${preview.newCount} 个新连接。选择“覆盖导入”将更新重名连接，选择“跳过冲突”只导入新连接。`,
          '导入预览',
          {
            confirmButtonText: '覆盖导入',
            cancelButtonText: '跳过冲突',
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
    ElMessage.success(`导入完成，新建 ${result.created}，覆盖 ${result.updated}，跳过 ${result.skipped}，失败 ${result.failed}`)
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
    ElMessage.success(`批量测试完成，成功 ${passed}，失败 ${failed}`)
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
  ElMessage.success('已切换当前连接')
  await loadConnections()
}

async function removeConnection(id, name) {
  await ElMessageBox.confirm(`确认删除连接 ${name} ?`, '提示', { type: 'warning' })
  const data = await deleteConnection(id)
  setActiveConnectionId(data.activeId || '')
  ElMessage.success('连接已删除')
  await loadConnections()
}

async function removeSelectedConnections() {
  if (!selectedIds.value.length) {
    ElMessage.warning('请先选择要删除的连接')
    return
  }
  await ElMessageBox.confirm(`确认批量删除 ${selectedIds.value.length} 个连接？`, '提示', { type: 'warning' })
  const result = await batchDeleteConnections(selectedIds.value)
  selectedIds.value = []
  setActiveConnectionId(result.activeId || '')
  await loadConnections()
  ElMessage.success(`批量删除完成，成功 ${result.deleted}，失败 ${result.failed}`)
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
          <span>NATS 服务器连接管理</span>
          <div class="connection-toolbar">
            <el-button :loading="batchTesting" @click="runBatchTest">批量测试</el-button>
            <el-button @click="exportConnections">导出配置</el-button>
            <label class="upload-button">
              <input type="file" accept="application/json" :disabled="importing" @change="handleImport" />
              <span>{{ importing ? '导入中...' : '导入配置' }}</span>
            </label>
            <el-button type="danger" plain @click="removeSelectedConnections">批量删除</el-button>
            <el-button @click="expandAllGroups">展开全部</el-button>
            <el-button @click="collapseAllGroups">收起全部</el-button>
            <el-button type="primary" @click="openCreate">新增连接</el-button>
          </div>
        </div>
      </template>

      <div class="connection-filters">
        <el-input v-model="filters.keyword" placeholder="搜索名称、地址、标签" clearable />
        <el-select v-model="filters.group" placeholder="全部分组" clearable>
          <el-option v-for="item in groupOptions" :key="item" :label="item" :value="item" />
        </el-select>
        <el-select v-model="filters.tag" placeholder="全部标签" clearable>
          <el-option v-for="item in tagOptions" :key="item" :label="item" :value="item" />
        </el-select>
        <el-select v-model="filters.status" placeholder="全部状态" clearable>
          <el-option v-for="item in statusOptions" :key="item" :label="item" :value="item" />
        </el-select>
        <el-button @click="resetFilters">重置筛选</el-button>
      </div>
    </el-card>

    <el-collapse v-model="activeGroups" class="connection-groups">
      <el-collapse-item v-for="group in groupedConnections" :key="group.name" :name="group.name">
        <template #title>
          <div class="connection-group-title">
            <strong>{{ group.name }}</strong>
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
                {{ row.id === activeId ? '当前连接' : row.status || '未检测' }}
              </el-tag>
            </div>

            <div class="connection-meta">
              <span>监控地址: {{ row.monitorEndpoints.join(', ') || '-' }}</span>
              <span>当前节点: {{ row.connectedUrl || '-' }}</span>
              <span>鉴权: {{ row.hasPassword || row.hasToken ? '已配置' : '无' }}</span>
            </div>

            <el-space wrap class="connection-tags" v-if="row.tags?.length">
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
              <el-button type="primary" plain @click="switchActive(row.id)">切换</el-button>
              <el-button plain @click="runSavedTest(row.id)">测试</el-button>
              <el-button plain @click="openEdit(row)">编辑</el-button>
              <el-button type="danger" plain @click="removeConnection(row.id, row.name)">删除</el-button>
            </div>
          </el-card>
        </div>
      </el-collapse-item>
    </el-collapse>

    <div class="table-pagination" v-if="total > 0">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        layout="total, sizes, prev, pager, next"
        :total="total"
        :page-sizes="[12, 24, 48]"
      />
    </div>

    <el-empty v-if="!groupedConnections.length" description="没有匹配的连接" />
  </div>

  <el-dialog v-model="dialogVisible" :title="editingId ? '编辑连接' : '新增连接'" width="640px">
    <el-form label-width="110px">
      <el-form-item label="连接名称">
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="分组">
        <el-input v-model="form.group" placeholder="例如: 生产 / 测试 / 区域A" />
      </el-form-item>
      <el-form-item label="标签">
        <el-input v-model="form.tags" placeholder="例如: jetstream, core, cn-east" />
      </el-form-item>
      <el-form-item label="NATS URLs">
        <el-input v-model="form.natsUrls" placeholder="nats://127.0.0.1:4222, nats://127.0.0.1:4223" />
      </el-form-item>
      <el-form-item label="Monitor URLs">
        <el-input v-model="form.monitorEndpoints" placeholder="http://127.0.0.1:8222, http://127.0.0.1:8223">
          <template #append>
            <el-button @click="fillMonitorEndpoints">自动发现</el-button>
          </template>
        </el-input>
      </el-form-item>
      <el-form-item label="用户名">
        <el-input v-model="form.username" />
      </el-form-item>
      <el-form-item label="密码">
        <el-input v-model="form.password" type="password" show-password :placeholder="editingId ? '留空表示保持不变' : ''" />
      </el-form-item>
      <el-form-item label="Token">
        <el-input v-model="form.token" :placeholder="editingId ? '留空表示保持不变' : ''" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button :loading="testing" @click="runProbe">测试连接</el-button>
      <el-button type="primary" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>
