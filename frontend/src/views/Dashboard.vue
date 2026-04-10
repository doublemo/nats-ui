<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import * as echarts from 'echarts'
import { useI18n } from 'vue-i18n'
import { getClusterNodeDetail, getClusterOverview, onConnectionChanged } from '../api/nats'

const DASHBOARD_VIEW_STATE_KEY = 'nats-ui-dashboard-view-state'
const METRIC_KEYS = [
  { key: 'healthyNodes', soft: false },
  { key: 'totalConn', soft: false },
  { key: 'totalSubs', soft: false },
  { key: 'clusterName', soft: false },
  { key: 'inBytes', soft: true },
  { key: 'outBytes', soft: true },
  { key: 'memoryUsage', soft: true },
  { key: 'slowConsumers', soft: true },
]

const { t, locale } = useI18n()
const overview = ref({
  nodes: [],
  summary: {},
  traffic: {},
  connections: { items: [] },
  warnings: [],
})
const loading = ref(false)
const error = ref('')
const trafficChartRef = ref()
const nodeDialogVisible = ref(false)
const nodeDetailLoading = ref(false)
const nodeDetailError = ref('')
const selectedNode = ref(null)
const connectionDialogVisible = ref(false)
const selectedConnection = ref(null)
const settings = reactive({
  autoRefresh: true,
  refreshInterval: 5000,
  maxPoints: 12,
  visibleMetrics: METRIC_KEYS.map((item) => item.key),
})
let chart
let timer
let unsubscribe
const trafficTimeline = []

const refreshOptions = computed(() => [
  { label: t('dashboard.refreshOptions.off'), value: 0 },
  { label: t('dashboard.refreshOptions.five'), value: 5000 },
  { label: t('dashboard.refreshOptions.ten'), value: 10000 },
  { label: t('dashboard.refreshOptions.thirty'), value: 30000 },
  { label: t('dashboard.refreshOptions.sixty'), value: 60000 },
])
const windowOptions = computed(() => [
  { label: t('dashboard.windowOptions.twelve'), value: 12 },
  { label: t('dashboard.windowOptions.twentyFour'), value: 24 },
  { label: t('dashboard.windowOptions.sixty'), value: 60 },
])
const metricOptions = computed(() =>
  METRIC_KEYS.map((item) => ({
    ...item,
    label: t(`dashboard.metrics.${item.key}`),
  })),
)
const warningTitle = computed(() => (overview.value.warnings || []).join(locale.value === 'zh-CN' ? '；' : '; '))
const nodeOverviewItems = computed(() => {
  if (!selectedNode.value) return []
  return [
    { label: t('dashboard.table.node'), value: selectedNode.value.name || '-' },
    { label: t('dashboard.table.host'), value: selectedNode.value.host || '-' },
    { label: t('dashboard.table.serverId'), value: selectedNode.value.serverId || '-' },
    { label: t('dashboard.table.cluster'), value: selectedNode.value.cluster || '-' },
    { label: t('dashboard.table.version'), value: selectedNode.value.version || '-' },
    { label: t('dashboard.table.monitorEndpoint'), value: selectedNode.value.monitorEndpoint || '-' },
    { label: t('dashboard.table.status'), value: formatNodeStatus(selectedNode.value.status) },
    { label: t('dashboard.table.lastError'), value: selectedNode.value.lastError || '-' },
  ]
})
const nodeMetricItems = computed(() => {
  if (!selectedNode.value) return []
  return [
    { label: t('dashboard.table.connections'), value: selectedNode.value.connections ?? 0 },
    { label: t('dashboard.table.activeConnections'), value: selectedNode.value.activeConnections ?? 0 },
    { label: t('dashboard.table.totalConnections'), value: selectedNode.value.totalConnections ?? 0 },
    { label: t('dashboard.table.subscriptions'), value: selectedNode.value.subscriptions ?? 0 },
    { label: t('dashboard.table.cpu'), value: selectedNode.value.cpu ?? 0 },
    { label: t('dashboard.table.memory'), value: formatBytes(selectedNode.value.mem) },
    { label: t('dashboard.table.slow'), value: selectedNode.value.slowConsumers ?? 0 },
    { label: t('dashboard.table.inMsgs'), value: selectedNode.value.inMsgs ?? 0 },
    { label: t('dashboard.table.outMsgs'), value: selectedNode.value.outMsgs ?? 0 },
    { label: t('dashboard.table.inBytes'), value: formatBytes(selectedNode.value.inBytes) },
    { label: t('dashboard.table.outBytes'), value: formatBytes(selectedNode.value.outBytes) },
  ]
})
const selectedNodeRaw = computed(() => (selectedNode.value?.rawVarz ? JSON.stringify(selectedNode.value.rawVarz, null, 2) : ''))
const selectedConnectionAddress = computed(() => {
  if (!selectedConnection.value) return '-'
  const { ip, port } = selectedConnection.value
  if (!ip) return '-'
  return port ? `${ip}:${port}` : ip
})
const connectionOverviewItems = computed(() => {
  if (!selectedConnection.value) return []
  return [
    { label: t('dashboard.table.cid'), value: selectedConnection.value.cid },
    { label: t('dashboard.table.client'), value: selectedConnection.value.name || '-' },
    { label: t('dashboard.table.address'), value: selectedConnectionAddress.value },
    { label: t('dashboard.table.port'), value: selectedConnection.value.port ?? '-' },
  ]
})
const connectionMetricItems = computed(() => {
  if (!selectedConnection.value) return []
  return [
    { label: t('dashboard.table.subscriptions'), value: selectedConnection.value.subs ?? 0 },
    { label: t('dashboard.table.inMsgs'), value: selectedConnection.value.inMsgs ?? 0 },
    { label: t('dashboard.table.outMsgs'), value: selectedConnection.value.outMsgs ?? 0 },
    { label: t('dashboard.table.inBytes'), value: formatBytes(selectedConnection.value.inBytes) },
    { label: t('dashboard.table.outBytes'), value: formatBytes(selectedConnection.value.outBytes) },
    { label: t('dashboard.table.pending'), value: formatBytes(selectedConnection.value.pending) },
  ]
})
const selectedConnectionRaw = computed(() =>
  selectedConnection.value ? JSON.stringify(selectedConnection.value, null, 2) : '',
)

restoreViewState()

function formatBytes(value) {
  const size = Number(value || 0)
  if (size < 1024) return `${size} B`
  if (size < 1024 ** 2) return `${(size / 1024).toFixed(1)} KB`
  if (size < 1024 ** 3) return `${(size / 1024 ** 2).toFixed(1)} MB`
  return `${(size / 1024 ** 3).toFixed(2)} GB`
}

function formatTrafficBytes(value) {
  const size = Number(value || 0)
  if (size < 1024) return `${size} B`
  if (size < 1024 ** 2) return `${formatTrafficUnit(size / 1024)} kb`
  if (size < 1024 ** 3) return `${formatTrafficUnit(size / 1024 ** 2)} M`
  return `${formatTrafficUnit(size / 1024 ** 3)} G`
}

function formatTrafficUnit(value) {
  if (value >= 100) return value.toFixed(0)
  if (value >= 10) return value.toFixed(1).replace(/\.0$/, '')
  return value.toFixed(2).replace(/0+$/, '').replace(/\.$/, '')
}

function formatNodeStatus(status) {
  if (status === 'healthy') {
    return t('dashboard.status.healthy')
  }
  if (!status) {
    return '-'
  }
  if (status === 'unhealthy') {
    return t('dashboard.status.unhealthy')
  }
  return status
}

function openConnectionDetail(row) {
  selectedConnection.value = { ...row }
  connectionDialogVisible.value = true
}

async function openNodeDetail(row) {
  nodeDialogVisible.value = true
  nodeDetailLoading.value = true
  nodeDetailError.value = ''
  selectedNode.value = {
    ...row,
    activeConnections: row.connections ?? 0,
    totalConnections: row.connections ?? 0,
  }

  try {
    if (!row.monitorEndpoint) {
      throw new Error(t('dashboard.table.monitorEndpoint'))
    }
    selectedNode.value = await getClusterNodeDetail(row.monitorEndpoint)
  } catch (err) {
    nodeDetailError.value = err.message
  } finally {
    nodeDetailLoading.value = false
  }
}

const metricValues = {
  healthyNodes: () => overview.value.summary.healthyNodes || 0,
  totalConn: () => overview.value.summary.totalConn || 0,
  totalSubs: () => overview.value.summary.totalSubs || 0,
  clusterName: () => overview.value.clusterName || '-',
  inBytes: () => formatBytes(overview.value.traffic.totalInBytes),
  outBytes: () => formatBytes(overview.value.traffic.totalOutBytes),
  memoryUsage: () => formatBytes(overview.value.summary.totalMem),
  slowConsumers: () => overview.value.connections.slowCount || 0,
}

async function loadOverview() {
  loading.value = true
  try {
    const data = await getClusterOverview()
    overview.value = data
    error.value = ''
    pushTrafficPoint(data.traffic)
    renderChart()
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

function restoreViewState() {
  try {
    const raw = window.localStorage.getItem(DASHBOARD_VIEW_STATE_KEY)
    if (!raw) return
    const saved = JSON.parse(raw)
    settings.autoRefresh = Boolean(saved.autoRefresh)
    settings.refreshInterval = Number(saved.refreshInterval || 5000)
    settings.maxPoints = Number(saved.maxPoints || 12)
    settings.visibleMetrics = Array.isArray(saved.visibleMetrics) && saved.visibleMetrics.length
      ? saved.visibleMetrics
      : METRIC_KEYS.map((item) => item.key)
  } catch {
    window.localStorage.removeItem(DASHBOARD_VIEW_STATE_KEY)
  }
}

function persistViewState() {
  window.localStorage.setItem(
    DASHBOARD_VIEW_STATE_KEY,
    JSON.stringify({
      autoRefresh: settings.autoRefresh,
      refreshInterval: settings.refreshInterval,
      maxPoints: settings.maxPoints,
      visibleMetrics: settings.visibleMetrics,
    }),
  )
}

function resetTimer() {
  if (timer) {
    window.clearInterval(timer)
    timer = null
  }
  if (settings.autoRefresh && settings.refreshInterval > 0) {
    timer = window.setInterval(loadOverview, settings.refreshInterval)
  }
}

function pushTrafficPoint(traffic) {
  trafficTimeline.push({
    time: new Date().toLocaleTimeString(locale.value === 'zh-CN' ? 'zh-CN' : 'en-US'),
    inBytes: traffic.totalInBytes,
    outBytes: traffic.totalOutBytes,
    inMsgs: traffic.totalInMsgs,
    outMsgs: traffic.totalOutMsgs,
  })
  if (trafficTimeline.length > settings.maxPoints) {
    trafficTimeline.shift()
  }
}

function renderChart() {
  if (!trafficChartRef.value) return
  if (!chart) chart = echarts.init(trafficChartRef.value)

  chart.setOption({
    tooltip: { trigger: 'axis' },
    legend: {
      data: [
        t('dashboard.chart.inBytes'),
        t('dashboard.chart.outBytes'),
        t('dashboard.chart.inMsgs'),
        t('dashboard.chart.outMsgs'),
      ],
    },
    grid: { left: 30, right: 20, top: 40, bottom: 20, containLabel: true },
    xAxis: { type: 'category', data: trafficTimeline.map((item) => item.time) },
    yAxis: [
      {
        type: 'value',
        name: t('dashboard.chart.bytesAxis'),
        axisLabel: { formatter: (value) => formatTrafficBytes(value) },
      },
      { type: 'value', name: t('dashboard.chart.msgsAxis') },
    ],
    series: [
      {
        name: t('dashboard.chart.inBytes'),
        type: 'line',
        smooth: true,
        tooltip: { valueFormatter: (value) => formatTrafficBytes(value) },
        data: trafficTimeline.map((item) => item.inBytes),
      },
      {
        name: t('dashboard.chart.outBytes'),
        type: 'line',
        smooth: true,
        tooltip: { valueFormatter: (value) => formatTrafficBytes(value) },
        data: trafficTimeline.map((item) => item.outBytes),
      },
      {
        name: t('dashboard.chart.inMsgs'),
        type: 'bar',
        yAxisIndex: 1,
        data: trafficTimeline.map((item) => item.inMsgs),
      },
      {
        name: t('dashboard.chart.outMsgs'),
        type: 'bar',
        yAxisIndex: 1,
        data: trafficTimeline.map((item) => item.outMsgs),
      },
    ],
  })
}

onMounted(async () => {
  await loadOverview()
  resetTimer()
  window.addEventListener('resize', renderChart)
  unsubscribe = onConnectionChanged(async () => {
    trafficTimeline.length = 0
    await loadOverview()
  })
})

onBeforeUnmount(() => {
  if (timer) window.clearInterval(timer)
  window.removeEventListener('resize', renderChart)
  unsubscribe?.()
  chart?.dispose()
})

watch(
  () => [settings.autoRefresh, settings.refreshInterval],
  () => {
    persistViewState()
    resetTimer()
  },
  { deep: true },
)

watch(
  () => settings.maxPoints,
  () => {
    while (trafficTimeline.length > settings.maxPoints) {
      trafficTimeline.shift()
    }
    persistViewState()
    renderChart()
  },
)

watch(
  () => settings.visibleMetrics,
  persistViewState,
  { deep: true },
)

watch(
  () => locale.value,
  () => {
    renderChart()
  },
)
</script>

<template>
  <div class="page-grid">
    <el-alert v-if="error" :title="error" type="error" show-icon class="mb-16" />
    <el-alert
      v-else-if="overview.warnings?.length"
      :title="warningTitle"
      type="warning"
      show-icon
      :closable="false"
      class="mb-16"
    />

    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>{{ t('dashboard.cardsTitle') }}</span>
          <el-select
            v-model="settings.visibleMetrics"
            multiple
            collapse-tags
            collapse-tags-tooltip
            style="width: 360px"
            :placeholder="t('dashboard.selectMetrics')"
          >
            <el-option v-for="item in metricOptions" :key="item.key" :label="item.label" :value="item.key" />
          </el-select>
        </div>
      </template>
      <div class="stats-grid">
        <div
          v-for="item in metricOptions.filter((metric) => settings.visibleMetrics.includes(metric.key))"
          :key="item.key"
          class="metric-card"
          :class="{ soft: item.soft }"
        >
          <span>{{ item.label }}</span>
          <strong>{{ metricValues[item.key]() }}</strong>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="mb-16">
      <template #header>
        <span>{{ t('dashboard.summaryTitle') }}</span>
      </template>
      <el-descriptions :column="4" border>
        <el-descriptions-item :label="t('dashboard.summary.cluster')">{{ overview.clusterName || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.summary.version')">{{ overview.version || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.summary.serverId')">{{ overview.serverId || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.summary.nodes')">{{ overview.nodeCount || 0 }}</el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.summary.activeConn')">{{ overview.connections.active || 0 }}</el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.summary.totalConn')">{{ overview.connections.total || 0 }}</el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.summary.inMsgs')">{{ overview.traffic.totalInMsgs || 0 }}</el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.summary.outMsgs')">{{ overview.traffic.totalOutMsgs || 0 }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card shadow="never" class="mb-16">
      <template #header>
        <div class="card-header">
          <span>{{ t('dashboard.trafficTitle') }}</span>
          <div class="data-toolbar">
            <el-switch
              v-model="settings.autoRefresh"
              inline-prompt
              :active-text="t('dashboard.auto')"
              :inactive-text="t('dashboard.manual')"
            />
            <el-select v-model="settings.refreshInterval" :disabled="!settings.autoRefresh" style="width: 120px">
              <el-option v-for="item in refreshOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-select v-model="settings.maxPoints" style="width: 120px">
              <el-option v-for="item in windowOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-button :loading="loading" text @click="loadOverview">{{ t('common.refresh') }}</el-button>
          </div>
        </div>
      </template>
      <div ref="trafficChartRef" class="chart-panel" />
    </el-card>

    <el-card shadow="never" class="mb-16">
      <template #header>
        <div class="card-header">
          <span>{{ t('dashboard.nodeStatusTitle') }}</span>
          <span class="table-tip">{{ t('dashboard.nodeHint') }}</span>
        </div>
      </template>
      <el-table :data="overview.nodes || []" stripe :empty-text="t('dashboard.nodeEmpty')" @row-dblclick="openNodeDetail">
        <el-table-column prop="name" :label="t('dashboard.table.node')" min-width="120" />
        <el-table-column prop="host" :label="t('dashboard.table.host')" min-width="120" />
        <el-table-column prop="version" :label="t('dashboard.table.version')" width="120" />
        <el-table-column prop="connections" :label="t('dashboard.table.connections')" width="100" />
        <el-table-column :label="t('dashboard.table.memory')" width="120">
          <template #default="{ row }">{{ formatBytes(row.mem) }}</template>
        </el-table-column>
        <el-table-column prop="cpu" :label="t('dashboard.table.cpu')" width="100" />
        <el-table-column prop="slowConsumers" :label="t('dashboard.table.slow')" width="110" />
        <el-table-column prop="inMsgs" :label="t('dashboard.table.inMsgs')" width="120" />
        <el-table-column prop="outMsgs" :label="t('dashboard.table.outMsgs')" width="120" />
        <el-table-column :label="t('dashboard.table.inBytes')" width="120">
          <template #default="{ row }">{{ formatBytes(row.inBytes) }}</template>
        </el-table-column>
        <el-table-column :label="t('dashboard.table.outBytes')" width="120">
          <template #default="{ row }">{{ formatBytes(row.outBytes) }}</template>
        </el-table-column>
        <el-table-column prop="status" :label="t('dashboard.table.status')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'healthy' ? 'success' : 'danger'">{{ formatNodeStatus(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="lastError" :label="t('dashboard.table.lastError')" min-width="320" show-overflow-tooltip />
      </el-table>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>{{ t('dashboard.connectionDetailsTitle') }}</span>
          <span class="table-tip">{{ t('dashboard.connectionHint') }}</span>
        </div>
      </template>
      <el-table
        :data="overview.connections.items || []"
        stripe
        :empty-text="t('dashboard.connectionEmpty')"
        @row-dblclick="openConnectionDetail"
      >
        <el-table-column prop="cid" :label="t('dashboard.table.cid')" width="90" />
        <el-table-column prop="name" :label="t('dashboard.table.client')" min-width="160" />
        <el-table-column prop="ip" :label="t('dashboard.table.ip')" width="140" />
        <el-table-column prop="subs" :label="t('dashboard.table.subscriptions')" width="100" />
        <el-table-column prop="inMsgs" :label="t('dashboard.table.inMsgs')" width="110" />
        <el-table-column prop="outMsgs" :label="t('dashboard.table.outMsgs')" width="110" />
        <el-table-column :label="t('dashboard.table.inBytes')" width="120">
          <template #default="{ row }">{{ formatBytes(row.inBytes) }}</template>
        </el-table-column>
        <el-table-column :label="t('dashboard.table.outBytes')" width="120">
          <template #default="{ row }">{{ formatBytes(row.outBytes) }}</template>
        </el-table-column>
        <el-table-column prop="pending" :label="t('dashboard.table.pending')" width="120" />
      </el-table>
    </el-card>
  </div>

  <el-dialog
    v-model="nodeDialogVisible"
    :title="t('dashboard.nodeDialog.title')"
    width="820px"
    destroy-on-close
  >
    <div v-loading="nodeDetailLoading">
      <el-alert v-if="nodeDetailError" :title="nodeDetailError" type="error" show-icon class="mb-16" />

      <template v-if="selectedNode">
        <el-card shadow="never" class="mb-16">
          <template #header>
            <span>{{ t('dashboard.nodeDialog.overview') }}</span>
          </template>
          <el-descriptions :column="2" border>
            <el-descriptions-item v-for="item in nodeOverviewItems" :key="item.label" :label="item.label">
              {{ item.value }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>

        <el-card shadow="never" class="mb-16">
          <template #header>
            <span>{{ t('dashboard.nodeDialog.metrics') }}</span>
          </template>
          <el-descriptions :column="2" border>
            <el-descriptions-item v-for="item in nodeMetricItems" :key="item.label" :label="item.label">
              {{ item.value }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>

        <el-card v-if="selectedNodeRaw" shadow="never">
          <template #header>
            <span>{{ t('dashboard.nodeDialog.raw') }}</span>
          </template>
          <el-input :model-value="selectedNodeRaw" type="textarea" :rows="12" readonly />
        </el-card>
      </template>
    </div>
  </el-dialog>

  <el-dialog
    v-model="connectionDialogVisible"
    :title="t('dashboard.connectionDialog.title')"
    width="760px"
    destroy-on-close
  >
    <template v-if="selectedConnection">
      <el-card shadow="never" class="mb-16">
        <template #header>
          <span>{{ t('dashboard.connectionDialog.overview') }}</span>
        </template>
        <el-descriptions :column="2" border>
          <el-descriptions-item v-for="item in connectionOverviewItems" :key="item.label" :label="item.label">
            {{ item.value }}
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <el-card shadow="never" class="mb-16">
        <template #header>
          <span>{{ t('dashboard.connectionDialog.metrics') }}</span>
        </template>
        <el-descriptions :column="2" border>
          <el-descriptions-item v-for="item in connectionMetricItems" :key="item.label" :label="item.label">
            {{ item.value }}
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <el-card shadow="never">
        <template #header>
          <span>{{ t('dashboard.connectionDialog.raw') }}</span>
        </template>
        <el-input :model-value="selectedConnectionRaw" type="textarea" :rows="10" readonly />
      </el-card>
    </template>
  </el-dialog>
</template>
