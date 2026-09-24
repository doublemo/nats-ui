<script setup>
import { onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import MessageContentPreview from './MessageContentPreview.vue'
import { createConsumer, deleteConsumer, getStreamMessage, onConnectionChanged } from '../api/nats'
const props = defineProps({ stream: { type: String, required: true }, consumers: { type: Array, default: () => [] } })
const emit = defineEmits(['refresh'])
const { locale } = useI18n()
const tr = (zh, en) => locale.value === 'zh-CN' ? zh : en
const dialog = ref(false)
const busy = ref(false)
const reading = ref(false)
const sequence = ref('1')
const message = ref(null)
const error = ref('')
let generation = 0
const form = reactive({ name: '', filter: '', deliverPolicy: 'all', ackWaitSec: 30, maxDeliver: 5 })
function reset() { generation++; message.value = null; error.value = ''; dialog.value = false; busy.value = false; reading.value = false }
watch(() => props.stream, reset)
const unsubscribe = onConnectionChanged(reset)
onBeforeUnmount(() => { generation++; unsubscribe() })
async function create() {
  const current = generation
  busy.value = true
  try { await createConsumer(props.stream, form); if (current === generation) { dialog.value = false; emit('refresh'); ElMessage.success(tr('Consumer 已创建', 'Consumer created')) } }
  catch (err) { if (current === generation) ElMessage.error(err.message) }
  finally { if (current === generation) busy.value = false }
}
async function remove(name) {
  const current = generation
  try {
    await ElMessageBox.confirm(tr(`删除 Consumer ${name}？确认位置和消费进度将丢失。`, `Delete consumer ${name}? Its acknowledgment state will be lost.`), tr('删除 Consumer', 'Delete consumer'), { type: 'warning' })
    if (current !== generation) return
    await deleteConsumer(props.stream, name)
    if (current === generation) emit('refresh')
  } catch (err) { if (err instanceof Error) ElMessage.error(err.message) }
}
async function read() {
  if (!/^[1-9]\d*$/.test(sequence.value)) { error.value = tr('请输入正整数序号', 'Enter a positive sequence number'); return }
  const current = generation
  reading.value = true; error.value = ''; message.value = null
  try { const result = await getStreamMessage(props.stream, sequence.value); if (current === generation) message.value = result }
  catch (err) { if (current === generation) error.value = err.message }
  finally { if (current === generation) reading.value = false }
}
</script>
<template>
  <div class="console-stack stream-tools">
    <div class="card-header"><strong>{{ tr('Consumer 管理', 'Consumer management') }}</strong><el-button type="primary" size="small" @click="dialog = true">{{ tr('创建 Pull Consumer', 'Create pull consumer') }}</el-button></div>
    <el-table :data="consumers" size="small">
      <el-table-column prop="name" label="Consumer" min-width="150" /><el-table-column prop="pending" :label="tr('待投递', 'Pending')" width="90" /><el-table-column prop="numAckPending" :label="tr('待确认', 'Ack pending')" width="100" /><el-table-column prop="numRedelivered" :label="tr('重投递', 'Redelivered')" width="100" /><el-table-column width="86"><template #default="{ row }"><el-button link type="danger" @click="remove(row.name)">{{ tr('删除', 'Delete') }}</el-button></template></el-table-column>
    </el-table>
    <div class="card-header"><strong>{{ tr('持久化消息查看', 'Stored message inspector') }}</strong><span class="table-tip">{{ tr('只读，不推进消费进度', 'Read-only; does not advance consumers') }}</span></div>
    <div class="data-toolbar"><el-input v-model="sequence" :placeholder="tr('消息序号', 'Message sequence')" @keyup.enter="read" /><el-button :loading="reading" @click="read">{{ tr('读取消息', 'Read message') }}</el-button></div>
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <div v-if="message"><p class="mono">#{{ message.sequence }} · {{ message.message.subject }} · {{ message.message.time }}</p><el-tag size="small">{{ message.preview.type }} · {{ message.preview.bytes }} bytes{{ message.preview.truncated ? ' / truncated' : '' }}</el-tag><MessageContentPreview :message="message.preview" /><details><summary>Headers</summary><pre class="payload-view">{{ JSON.stringify(message.preview.headers, null, 2) }}</pre></details></div>
    <el-dialog v-model="dialog" :title="tr('创建持久化 Pull Consumer', 'Create durable pull consumer')" width="min(520px, 94vw)">
      <el-form label-position="top"><el-form-item label="Name"><el-input v-model="form.name" /></el-form-item><el-form-item label="Filter subject"><el-input v-model="form.filter" placeholder="orders.>" /></el-form-item><el-form-item label="Deliver policy"><el-select v-model="form.deliverPolicy"><el-option label="All" value="all" /><el-option label="New" value="new" /><el-option label="Last" value="last" /></el-select></el-form-item><div class="form-pair"><el-form-item label="Ack wait (seconds)"><el-input-number v-model="form.ackWaitSec" :min="1" :max="86400" /></el-form-item><el-form-item label="Max deliveries"><el-input-number v-model="form.maxDeliver" :min="1" :max="1000" /></el-form-item></div><p class="table-tip">Ack policy: Explicit · Max ack pending: 1000</p></el-form>
      <template #footer><el-button @click="dialog = false">{{ tr('取消', 'Cancel') }}</el-button><el-button type="primary" :loading="busy" :disabled="!form.name.trim()" @click="create">{{ tr('创建', 'Create') }}</el-button></template>
    </el-dialog>
  </div>
</template>
