<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import * as echarts from 'echarts'
import { getClusterOverview, onConnectionChanged } from '../api/nats'

const overview = ref({
  nodes: [],
  summary: {},
  traffic: {},
  connections: { items: [] },
})
const loading = ref(false)
const error = ref('')
const trafficChartRef = ref()
let chart
let timer
let unsubscribe
const trafficTimeline = []

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

function pushTrafficPoint(traffic) {
  trafficTimeline.push({
    time: new Date().toLocaleTimeString(),
    inBytes: traffic.totalInBytes,
    outBytes: traffic.totalOutBytes,
    inMsgs: traffic.totalInMsgs,
    outMsgs: traffic.totalOutMsgs,
  })
  if (trafficTimeline.length > 12) {
    trafficTimeline.shift()
  }
}

function renderChart() {
  if (!trafficChartRef.value) {
    return
  }
  if (!chart) {
    chart = echarts.init(trafficChartRef.value)
  }

  chart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['In Bytes', 'Out Bytes', 'In Msgs', 'Out Msgs'] },
    grid: { left: 30, right: 20, top: 40, bottom: 20, containLabel: true },
    xAxis: {
      type: 'category',
      data: trafficTimeline.map((item) => item.time),
    },
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
  timer = window.setInterval(loadOverview, 5000)
  window.addEventListener('resize', renderChart)
  unsubscribe = onConnectionChanged(async () => {
    trafficTimeline.length = 0
    await loadOverview()
  })
})

onBeforeUnmount(() => {
  if (timer) {
    window.clearInterval(timer)
  }
  window.removeEventListener('resize', renderChart)
  unsubscribe?.()
  chart?.dispose()
})
</script>

<template>
  <div class="page-grid">
    <el-alert v-if="error" :title="error" type="error" show-icon class="mb-16" />

    <el-row :gutter="16" class="mb-16">
      <el-col :span="6">
        <div class="metric-card">
          <span>Healthy Nodes</span>
          <strong>{{ overview.summary.healthyNodes || 0 }}</strong>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="metric-card">
          <span>Total Conn</span>
          <strong>{{ overview.summary.totalConn || 0 }}</strong>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="metric-card">
          <span>Total Subs</span>
          <strong>{{ overview.summary.totalSubs || 0 }}</strong>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="metric-card">
          <span>Cluster</span>
          <strong>{{ overview.clusterName || '-' }}</strong>
        </div>
      </el-col>
    </el-row>

    <el-card shadow="never" class="mb-16">
      <template #header>
        <div class="card-header">
          <span>实时流量</span>
          <el-button :loading="loading" text @click="loadOverview">刷新</el-button>
        </div>
      </template>
      <div ref="trafficChartRef" class="chart-panel" />
    </el-card>

    <el-card shadow="never" class="mb-16">
      <template #header>
        <span>集群节点状态</span>
      </template>
      <el-table :data="overview.nodes" stripe>
        <el-table-column prop="name" label="节点" min-width="120" />
        <el-table-column prop="host" label="Host" min-width="120" />
        <el-table-column prop="version" label="Version" width="120" />
        <el-table-column prop="connections" label="连接数" width="100" />
        <el-table-column prop="cpu" label="CPU" width="100" />
        <el-table-column prop="mem" label="内存" width="120" />
        <el-table-column prop="inMsgs" label="In Msgs" width="120" />
        <el-table-column prop="outMsgs" label="Out Msgs" width="120" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'healthy' ? 'success' : 'danger'">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <span>连接明细</span>
      </template>
      <el-table :data="overview.connections.items" stripe>
        <el-table-column prop="cid" label="CID" width="90" />
        <el-table-column prop="name" label="客户端" min-width="160" />
        <el-table-column prop="ip" label="IP" width="140" />
        <el-table-column prop="subs" label="订阅数" width="100" />
        <el-table-column prop="inMsgs" label="In Msgs" width="110" />
        <el-table-column prop="outMsgs" label="Out Msgs" width="110" />
        <el-table-column prop="pending" label="Pending" width="120" />
      </el-table>
    </el-card>
  </div>
</template>
