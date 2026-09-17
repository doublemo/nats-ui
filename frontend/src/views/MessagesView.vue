<script setup>
import { computed, onBeforeUnmount, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { getActiveConnectionId, onConnectionChanged, sendMessage, subscribeMessages } from '../api/nats'

const { locale } = useI18n()
const tr = (zh, en) => locale.value === 'zh-CN' ? zh : en
const filter = ref('')
const subject = ref('>')
const queue = ref('')
const state = ref('idle')
const error = ref('')
const messages = ref([])
const received = ref(0)
const selected = ref(null)
const sending = ref(false)
const result = ref(null)
const mode = ref('publish')
const form = reactive({ subject: '', payload: '', headers: '{}', timeoutMs: 3000 })
let source
let generation = 0
const visible = computed(() => messages.value.filter(item => item.subject.includes(filter.value)))
function stop() { source?.close(); source = null; state.value = 'idle' }
function start() {
  stop(); error.value = ''; state.value = 'connecting'
  const current = subscribeMessages(subject.value.trim(), queue.value.trim())
  source = current
  current.addEventListener('ready', () => { if (source === current) state.value = 'live' })
  current.onmessage = event => {
    if (source !== current) return
    const record = JSON.parse(event.data)
    received.value++
    messages.value.unshift({ ...record, id: received.value })
    if (messages.value.length > 200) messages.value.pop()
  }
  current.addEventListener('failure', event => { error.value = JSON.parse(event.data).message; stop() })
  current.onerror = () => { if (source === current) { error.value = tr('订阅已断开，请检查连接和订阅权限后重新开始。', 'Subscription disconnected. Check connection and permissions, then restart.'); stop() } }
}
function reply(item) {
  mode.value = 'publish'; form.subject = item.reply; form.payload = ''; selected.value = item
}
async function send() {
  const current = generation
  const connection = getActiveConnectionId()
  sending.value = true; result.value = null
  try {
    const headers = JSON.parse(form.headers || '{}')
    if (!headers || Array.isArray(headers) || typeof headers !== 'object' || Object.values(headers).some(v => typeof v !== 'string')) throw new Error(tr('Headers 必须是字符串键值 JSON 对象', 'Headers must be a JSON object of string values'))
    const data = await sendMessage(mode.value, { ...form, subject: form.subject.trim(), headers }, connection)
    if (current === generation) { result.value = data; ElMessage.success(tr('发送成功', 'Sent successfully')) }
  } catch (err) { if (current === generation) ElMessage.error(err.message) }
  finally { if (current === generation) sending.value = false }
}
const unsubscribe = onConnectionChanged(() => { generation++; stop(); messages.value = []; selected.value = null; result.value = null; received.value = 0; error.value = ''; sending.value = false })
onBeforeUnmount(() => { generation++; stop(); unsubscribe() })
</script>

<template>
  <div class="console-stack">
    <div class="workspace-heading"><div><span class="eyebrow">CORE NATS / MESSAGING</span><h2>{{ tr('消息工作台', 'Message workspace') }}</h2><p>{{ tr('实时订阅、发布与请求响应。支持 * 和 > 通配符。', 'Live subscriptions, publishing and request/reply. Supports * and > wildcards.') }}</p></div><el-tag :type="state === 'live' ? 'success' : 'info'">{{ state === 'live' ? tr('订阅中', 'LIVE') : state === 'connecting' ? tr('连接中', 'CONNECTING') : tr('已停止', 'STOPPED') }}</el-tag></div>
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <div class="message-layout">
      <el-card shadow="never">
        <template #header><div class="card-header"><strong>{{ tr('实时订阅', 'Live subscription') }}</strong><span class="mono">{{ received }} {{ tr('条已接收', 'received') }}</span></div></template>
        <el-form label-position="top" @submit.prevent="start">
          <div class="form-pair"><el-form-item label="Subject"><el-input v-model="subject" :disabled="state !== 'idle'" placeholder="orders.>" /></el-form-item><el-form-item :label="tr('Queue Group（可选）', 'Queue group (optional)')"><el-input v-model="queue" :disabled="state !== 'idle'" /></el-form-item></div>
          <div class="data-toolbar"><el-button v-if="state === 'idle'" type="primary" :disabled="!subject.trim()" @click="start">{{ tr('开始订阅', 'Subscribe') }}</el-button><el-button v-else type="danger" plain @click="stop">{{ tr('停止订阅', 'Stop') }}</el-button><el-button @click="messages = []; selected = null">{{ tr('清空列表', 'Clear') }}</el-button><el-input v-model="filter" :placeholder="tr('筛选 Subject', 'Filter subjects')" clearable /></div>
        </el-form>
        <p class="table-tip">{{ tr('保留最近 200 条，每条预览最多 64 KiB。队列组会与同组客户端竞争消息。', 'Keeps 200 recent messages; previews up to 64 KiB each. Queue groups compete with other members for messages.') }}</p>
        <el-table :data="visible" height="350" highlight-current-row @row-click="selected = $event">
          <el-table-column prop="id" label="#" width="58" /><el-table-column prop="subject" label="Subject" min-width="170" show-overflow-tooltip /><el-table-column label="Time" width="108"><template #default="{ row }">{{ new Date(row.time).toLocaleTimeString() }}</template></el-table-column><el-table-column prop="bytes" label="Bytes" width="80" />
        </el-table>
        <div v-if="selected" class="message-detail"><div class="card-header"><strong class="mono">{{ selected.subject }}</strong><el-button v-if="selected.reply" size="small" @click="reply(selected)">{{ tr('回复', 'Reply') }}</el-button></div><p v-if="selected.reply" class="mono">Reply: {{ selected.reply }}</p><el-tag size="small">{{ selected.encoding }}{{ selected.truncated ? ' / truncated' : '' }}</el-tag><pre class="payload-view">{{ selected.payload }}</pre><details><summary>Headers</summary><pre class="payload-view">{{ JSON.stringify(selected.headers, null, 2) }}</pre></details></div>
      </el-card>
      <el-card shadow="never">
        <template #header><strong>{{ tr('消息发送', 'Message composer') }}</strong></template>
        <el-radio-group v-model="mode" class="mb-16"><el-radio-button value="publish">Publish / Reply</el-radio-button><el-radio-button value="request">Request</el-radio-button></el-radio-group>
        <el-form label-position="top" @submit.prevent="send">
          <el-form-item label="Subject"><el-input v-model="form.subject" placeholder="orders.created" /></el-form-item>
          <el-form-item label="Payload"><el-input v-model="form.payload" type="textarea" :rows="8" class="code-input" :placeholder="tr('文本或 JSON 消息', 'Text or JSON payload')" /></el-form-item>
          <el-form-item label="Headers (JSON)"><el-input v-model="form.headers" type="textarea" :rows="2" class="code-input" /></el-form-item>
          <el-form-item v-if="mode === 'request'" :label="tr('超时（毫秒）', 'Timeout (ms)')"><el-input-number v-model="form.timeoutMs" :min="100" :max="30000" :step="100" /></el-form-item>
          <el-button type="primary" :loading="sending" :disabled="!form.subject.trim()" @click="send">{{ mode === 'request' ? tr('发送请求', 'Send request') : tr('发布消息', 'Publish message') }}</el-button>
        </el-form>
        <div v-if="result" class="message-detail"><el-tag type="success">{{ result.elapsedMs.toFixed(1) }} ms</el-tag><template v-if="result.message"><p class="mono">{{ result.message.subject }} · {{ result.message.encoding }}</p><pre class="payload-view">{{ result.message.payload }}</pre></template><p v-else>{{ tr('消息已发送至服务器（不代表已被消费者处理）', 'Sent to server (does not confirm consumer processing)') }}</p></div>
      </el-card>
    </div>
  </div>
</template>
