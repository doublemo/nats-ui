<script setup>
import { onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import * as echarts from 'echarts'
import { getClusterOverview, onConnectionChanged } from '../api/nats'

const DASHBOARD_VIEW_STATE_KEY = 'nats-ui-dashboard-view-state'
const REFRESH_OPTIONS = [
  { label: '关闭', value: 0 },
  { label: '5 秒', value: 5000 },
  { label: '10 秒', value: 10000 },
  { label: '30 秒', value: 30000 },
  { label: '60 秒', value: 60000 },
]
const WINDOW_OPTIONS = [
  { label: '12 点', value: 12 },
  { label: '24 点', value: 24 },
  { label: '60 点', value: 60 },
]
const METRIC_OPTIONS = [
  { key: 'healthyNodes', label: 'Healthy Nodes', soft: false },
  { key: 'totalConn', label: 'Total Conn', soft: false },
  { key: 'totalSubs', label: 'Total Subs', soft: false },
  { key: 'clusterName', label: 'Cluster', soft: false },
  { key: 'inBytes', label: 'In Bytes', soft: true },
  { key: 'outBytes', label: 'Out Bytes', soft: true },
  { key: 'memoryUsage', label: 'Memory Usage', soft: true },
  { key: 'slowConsumers', label: 'Slow Consumers', soft: true },
]

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
const settings = reactive({
  autoRefresh: true,
  refreshInterval: 5000,
  maxPoints: 12,
  visibleMetrics: METRIC_OPTIONS.map((item) => item.key),
})
let chart
let timer
let unsubscribe
const trafficTimeline = []

restoreViewState()

function formatBytes(value) {
  const size = Number(value || 0)
  if (size < 1024) return `${size} B`
  if (size < 1024 ** 2) return `${(size / 1024).toFixed(1)} KB`
  if (size < 1024 ** 3) return `${(size / 1024 ** 2).toFixed(1)} MB`
  return `${(size / 1024 ** 3).toFixed(2)} GB`
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
      : METRIC_OPTIONS.map((item) => item.key)
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
    time: new Date().toLocaleTimeString(),
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
    legend: { data: ['In Bytes', 'Out Bytes', 'In Msgs', 'Out Msgs'] },
    grid: { left: 30, right: 20, top: 40, bottom: 20, containLabel: true },
    xAxis: { type: 'category', data: trafficTimeline.map((item) => item.time) },
    yAxis: [
      { type: 'value', name: 'Bytes' },
      { type: 'value', name: 'Msgs' },
    ],
    series: [
      { name: 'In Bytes', type: 'line', smooth: true, data: trafficTimeline.map((item) => item.inBytes) },
      { name: 'Out Bytes', type: 'line', smooth: true, data: trafficTimeline.map((item) => item.outBytes) },
      { name: 'In Msgs', type: 'bar', yAxisIndex: 1, data: trafficTimeline.map((item) => item.inMsgs) },
      { name: 'Out Msgs', type: 'bar', yAxisIndex: 1, data: trafficTimeline.map((item) => item.outMsgs) },
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
</script>

<template>
  <div class="page-grid">
    <el-alert v-if="error" :title="error" type="error" show-icon class="mb-16" />
    <el-alert
      v-else-if="overview.warnings?.length"
      :title="overview.warnings.join('；')"
      type="warning"
      show-icon
      :closable="false"
      class="mb-16"
    />

    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>指标卡片</span>
          <el-select
            v-model="settings.visibleMetrics"
            multiple
            collapse-tags
            collapse-tags-tooltip
            style="width: 360px"
            placeholder="选择显示指标"
          >
            <el-option v-for="item in METRIC_OPTIONS" :key="item.key" :label="item.label" :value="item.key" />
          </el-select>
        </div>
      </template>
      <div class="stats-grid">
        <div
          v-for="item in METRIC_OPTIONS.filter((metric) => settings.visibleMetrics.includes(metric.key))"
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
        <span>当前集群摘要</span>
      </template>
      <el-descriptions :column="4" border>
        <el-descriptions-item label="Cluster">{{ overview.clusterName || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Version">{{ overview.version || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Server ID">{{ overview.serverId || '-' }}</el-descriptions-item>
        <el-descriptions-item label="Nodes">{{ overview.nodeCount || 0 }}</el-descriptions-item>
        <el-descriptions-item label="Active Conn">{{ overview.connections.active || 0 }}</el-descriptions-item>
        <el-descriptions-item label="Total Conn">{{ overview.connections.total || 0 }}</el-descriptions-item>
        <el-descriptions-item label="In Msgs">{{ overview.traffic.totalInMsgs || 0 }}</el-descriptions-item>
        <el-descriptions-item label="Out Msgs">{{ overview.traffic.totalOutMsgs || 0 }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card shadow="never" class="mb-16">
      <template #header>
        <div class="card-header">
          <span>实时流量</span>
          <div class="data-toolbar">
            <el-switch v-model="settings.autoRefresh" inline-prompt active-text="自动" inactive-text="手动" />
            <el-select v-model="settings.refreshInterval" :disabled="!settings.autoRefresh" style="width: 120px">
              <el-option v-for="item in REFRESH_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-select v-model="settings.maxPoints" style="width: 120px">
              <el-option v-for="item in WINDOW_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-button :loading="loading" text @click="loadOverview">刷新</el-button>
          </div>
        </div>
      </template>
      <div ref="trafficChartRef" class="chart-panel" />
    </el-card>

    <el-card shadow="never" class="mb-16">
      <template #header>
        <span>集群节点状态</span>
      </template>
      <el-table :data="overview.nodes || []" stripe empty-text="暂无节点监控数据">
        <el-table-column prop="name" label="节点" min-width="120" />
        <el-table-column prop="host" label="Host" min-width="120" />
        <el-table-column prop="version" label="Version" width="120" />
        <el-table-column prop="connections" label="连接数" width="100" />
        <el-table-column label="内存" width="120">
          <template #default="{ row }">{{ formatBytes(row.mem) }}</template>
        </el-table-column>
        <el-table-column prop="cpu" label="CPU" width="100" />
        <el-table-column prop="slowConsumers" label="Slow" width="90" />
        <el-table-column prop="inMsgs" label="In Msgs" width="120" />
        <el-table-column prop="outMsgs" label="Out Msgs" width="120" />
        <el-table-column label="In Bytes" width="120">
          <template #default="{ row }">{{ formatBytes(row.inBytes) }}</template>
        </el-table-column>
        <el-table-column label="Out Bytes" width="120">
          <template #default="{ row }">{{ formatBytes(row.outBytes) }}</template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'healthy' ? 'success' : 'danger'">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="lastError" label="错误信息" min-width="320" show-overflow-tooltip />
      </el-table>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <span>连接明细</span>
      </template>
      <el-table :data="overview.connections.items || []" stripe empty-text="暂无连接监控数据">
        <el-table-column prop="cid" label="CID" width="90" />
        <el-table-column prop="name" label="客户端" min-width="160" />
        <el-table-column prop="ip" label="IP" width="140" />
        <el-table-column prop="subs" label="订阅数" width="100" />
        <el-table-column prop="inMsgs" label="In Msgs" width="110" />
        <el-table-column prop="outMsgs" label="Out Msgs" width="110" />
        <el-table-column label="In Bytes" width="120">
          <template #default="{ row }">{{ formatBytes(row.inBytes) }}</template>
        </el-table-column>
        <el-table-column label="Out Bytes" width="120">
          <template #default="{ row }">{{ formatBytes(row.outBytes) }}</template>
        </el-table-column>
        <el-table-column prop="pending" label="Pending" width="120" />
      </el-table>
    </el-card>
  </div>
</template>
