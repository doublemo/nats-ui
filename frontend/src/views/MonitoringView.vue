<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import * as echarts from 'echarts'
import { getClusterOverview, getJetStreamAccount, onConnectionChanged } from '../api/nats'
const { locale } = useI18n()
const tr = (zh, en) => locale.value === 'zh-CN' ? zh : en
const overview = ref(null)
const account = ref(null)
const error = ref('')
const jsError = ref('')
const updated = ref(null)
const loading = ref(false)
const interval = ref(5000)
const rates = ref(null)
const chartEl = ref(null)
const samples = []
let timer, chart, resizeObserver, unsubscribe, previous
let generation = 0
let disposed = false
const bytes = n => { const v = Number(n || 0); return v >= 1073741824 ? `${(v / 1073741824).toFixed(2)} GiB` : v >= 1048576 ? `${(v / 1048576).toFixed(1)} MiB` : v >= 1024 ? `${(v / 1024).toFixed(1)} KiB` : `${v} B` }
const cards = computed(() => [
  [tr('入站消息 / 秒', 'Inbound messages / s'), rates.value?.inMsgs?.toFixed(1) ?? '—'],
  [tr('出站消息 / 秒', 'Outbound messages / s'), rates.value?.outMsgs?.toFixed(1) ?? '—'],
  [tr('入站流量 / 秒', 'Inbound bytes / s'), rates.value ? bytes(rates.value.inBytes) : '—'],
  [tr('出站流量 / 秒', 'Outbound bytes / s'), rates.value ? bytes(rates.value.outBytes) : '—'],
])
const pending = computed(() => (overview.value?.connections?.items || []).reduce((sum, item) => sum + item.pending, 0))
function render() {
  if (!chart) return
  chart.setOption({ animation: false, color: ['#6558d3', '#0d9488'], tooltip: { trigger: 'axis' }, legend: { data: ['IN msg/s', 'OUT msg/s'], right: 12 }, grid: { left: 60, right: 24, top: 42, bottom: 30 }, xAxis: { type: 'category', boundaryGap: false, data: samples.map(s => s.time), axisLine: { lineStyle: { color: '#cbd5e1' } }, axisLabel: { color: '#64748b' } }, yAxis: { type: 'value', splitLine: { lineStyle: { color: '#edf0f4' } } }, series: ['inMsgs', 'outMsgs'].map((key, i) => ({ name: i ? 'OUT msg/s' : 'IN msg/s', type: 'line', showSymbol: false, areaStyle: { opacity: 0.07 }, data: samples.map(s => s[key]) })) })
}
async function refresh() {
  if (loading.value || disposed) return
  clearTimeout(timer)
  const current = generation
  loading.value = true
  try {
    const [cluster, js] = await Promise.allSettled([getClusterOverview(), getJetStreamAccount()])
    if (current !== generation || disposed) return
    const now = Date.now()
    if (cluster.status === 'fulfilled') {
      overview.value = cluster.value; error.value = ''; updated.value = now
      const nodes = cluster.value.nodes.filter(node => node.status === 'healthy')
      const fingerprint = nodes.map(node => node.serverId).sort().join(',')
      const elapsed = previous ? (now - previous.time) / 1000 : 0
      const comparable = previous && elapsed > 0 && fingerprint === previous.fingerprint && nodes.length > 0 && nodes.every(node => {
        const before = previous.nodes.find(item => item.serverId === node.serverId)
        return before && ['inMsgs', 'outMsgs', 'inBytes', 'outBytes'].every(key => node[key] >= before[key])
      })
      rates.value = comparable ? Object.fromEntries(['inMsgs', 'outMsgs', 'inBytes', 'outBytes'].map(key => [key, nodes.reduce((sum, node) => sum + node[key] - previous.nodes.find(item => item.serverId === node.serverId)[key], 0) / elapsed])) : null
      previous = { time: now, fingerprint, nodes }
      samples.push({ time: new Date(now).toLocaleTimeString(), inMsgs: rates.value?.inMsgs ?? null, outMsgs: rates.value?.outMsgs ?? null })
      if (samples.length > 60) samples.shift()
      render()
    } else { error.value = cluster.reason.message; previous = null; rates.value = null }
    if (js.status === 'fulfilled') { account.value = js.value; jsError.value = '' } else { account.value = null; jsError.value = js.reason.message }
  } finally {
    if (current === generation && !disposed) { loading.value = false; if (interval.value) timer = setTimeout(refresh, interval.value) }
  }
}
function reschedule() { clearTimeout(timer); if (interval.value && !loading.value) refresh() }
function reset() { generation++; clearTimeout(timer); previous = null; samples.length = 0; rates.value = null; overview.value = null; account.value = null; updated.value = null; error.value = ''; jsError.value = ''; loading.value = false; render(); refresh() }
onMounted(async () => { await nextTick(); chart = echarts.init(chartEl.value); resizeObserver = new ResizeObserver(() => chart?.resize()); resizeObserver.observe(chartEl.value); unsubscribe = onConnectionChanged(reset); render(); refresh() })
onBeforeUnmount(() => { disposed = true; generation++; clearTimeout(timer); unsubscribe?.(); resizeObserver?.disconnect(); chart?.dispose() })
</script>
<template>
  <div class="console-stack">
    <div class="workspace-heading"><div><span class="eyebrow">OBSERVABILITY / LIVE METRICS</span><h2>{{ tr('运行监控', 'Operations monitor') }}</h2><p>{{ tr('基于服务器计数器计算速率；节点变化或计数器重置后重新采样。', 'Rates derived from server counters; sampling resets on topology or counter changes.') }}</p></div><div class="data-toolbar"><el-select v-model="interval" style="width: 120px" @change="reschedule"><el-option :label="tr('暂停', 'Paused')" :value="0" /><el-option label="5 seconds" :value="5000" /><el-option label="10 seconds" :value="10000" /><el-option label="30 seconds" :value="30000" /></el-select><el-button :loading="loading" @click="refresh">{{ tr('刷新', 'Refresh') }}</el-button></div></div>
    <el-alert v-if="error" :title="error" type="error" :closable="false" /><el-alert v-if="overview?.warnings?.length" :title="overview.warnings.join('; ')" type="warning" :closable="false" />
    <div class="stats-grid"><div v-for="[label, value] in cards" :key="label" class="metric-card"><span>{{ label }}</span><strong>{{ value }}</strong></div></div>
    <el-card shadow="never"><template #header><div class="card-header"><strong>{{ tr('消息吞吐趋势', 'Message throughput') }}</strong><span class="table-tip">{{ updated ? new Date(updated).toLocaleTimeString() : '—' }} · {{ tr('最近 60 次采样', 'Last 60 samples') }}</span></div></template><div ref="chartEl" style="height: 250px" /></el-card>
    <div class="monitor-grid"><el-card shadow="never"><template #header><strong>{{ tr('节点健康与资源', 'Node health & resources') }}</strong></template><el-table :data="overview?.nodes || []"><el-table-column prop="name" label="Server" min-width="130" /><el-table-column label="Status" width="105"><template #default="{ row }"><el-tag :type="row.status === 'healthy' ? 'success' : 'danger'" size="small">{{ row.status }}</el-tag></template></el-table-column><el-table-column label="CPU" width="80"><template #default="{ row }">{{ row.status === 'healthy' ? `${row.cpu.toFixed(1)}%` : '—' }}</template></el-table-column><el-table-column label="Memory" width="110"><template #default="{ row }">{{ row.status === 'healthy' ? bytes(row.mem) : '—' }}</template></el-table-column><el-table-column prop="slowConsumers" :label="tr('慢消费者累计', 'Slow consumers')" width="130" /></el-table><p class="table-tip">{{ tr('慢消费者为服务器累计计数；离线节点不参与速率计算。', 'Slow consumers is a cumulative server counter; offline nodes are excluded from rates.') }}</p></el-card>
    <el-card shadow="never"><template #header><strong>{{ tr('JetStream 账户资源', 'JetStream account resources') }}</strong></template><el-alert v-if="jsError" :title="jsError" type="warning" :closable="false" /><el-descriptions v-else-if="account" :column="2" border><el-descriptions-item label="Streams">{{ account.streams }}</el-descriptions-item><el-descriptions-item label="Consumers">{{ account.consumers }}</el-descriptions-item><el-descriptions-item label="Memory">{{ bytes(account.memory) }}</el-descriptions-item><el-descriptions-item label="Storage">{{ bytes(account.storage) }}</el-descriptions-item><el-descriptions-item label="API requests">{{ account.api?.total }}</el-descriptions-item><el-descriptions-item label="API errors">{{ account.api?.errors }}</el-descriptions-item></el-descriptions><el-empty v-else :description="tr('等待采样', 'Awaiting sample')" /></el-card></div>
    <el-card shadow="never"><template #header><div class="card-header"><strong>{{ tr('连接积压', 'Connection backlog') }}</strong><el-tag :type="pending > 0 ? 'warning' : 'info'">{{ bytes(pending) }} pending</el-tag></div></template><p class="table-tip">{{ tr('显示监控接口返回的连接样本；连接数很大时并非全量。', 'Shows connection samples returned by monitoring endpoints; large connection sets may be partial.') }}</p><el-table :data="[...(overview?.connections?.items || [])].sort((a, b) => b.pending - a.pending).slice(0, 50)" max-height="320"><el-table-column prop="cid" label="CID" width="100" /><el-table-column prop="name" label="Client" min-width="170" /><el-table-column prop="ip" label="IP" min-width="130" /><el-table-column prop="subs" label="Subscriptions" width="120" /><el-table-column label="Pending bytes" width="150"><template #default="{ row }"><span :class="{ 'warning-text': row.pending > 0 }">{{ bytes(row.pending) }}</span></template></el-table-column></el-table></el-card>
  </div>
</template>
