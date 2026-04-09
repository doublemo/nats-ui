<script setup>
import { onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { batchDeleteStreams, createStream, deleteStream, getStreamDetail, getStreams, onConnectionChanged } from '../api/nats'

const JETSTREAM_VIEW_STATE_KEY = 'nats-ui-jetstream-view-state'

const streams = ref([])
const total = ref(0)
const loading = ref(false)
const selected = ref(null)
const selectedRows = ref([])
const detail = ref(null)
const dialogVisible = ref(false)
const page = ref(1)
const pageSize = ref(8)
let unsubscribe
const form = reactive({
  name: '',
  subjects: '',
  storage: 'file',
  replicas: 1,
  maxAgeSec: 0,
})

restoreViewState()

async function loadStreams() {
  loading.value = true
  try {
    const data = await getStreams({ page: page.value, pageSize: pageSize.value })
    streams.value = data.items
    total.value = data.total
    if (selected.value) {
      await selectStream(selected.value)
    }
  } finally {
    loading.value = false
  }
}

function restoreViewState() {
  try {
    const raw = window.localStorage.getItem(JETSTREAM_VIEW_STATE_KEY)
    if (!raw) return
    const saved = JSON.parse(raw)
    page.value = saved.page || 1
    pageSize.value = saved.pageSize || 8
    selected.value = saved.selected || null
  } catch {
    window.localStorage.removeItem(JETSTREAM_VIEW_STATE_KEY)
  }
}

function persistViewState() {
  window.localStorage.setItem(
    JETSTREAM_VIEW_STATE_KEY,
    JSON.stringify({
      page: page.value,
      pageSize: pageSize.value,
      selected: selected.value,
    }),
  )
}

async function selectStream(name) {
  selected.value = name
  detail.value = await getStreamDetail(name)
}

async function submitStream() {
  await createStream({
    name: form.name,
    subjects: form.subjects.split(',').map((item) => item.trim()).filter(Boolean),
    storage: form.storage,
    replicas: form.replicas,
    maxAgeSec: form.maxAgeSec,
  })
  ElMessage.success('Stream 创建成功')
  dialogVisible.value = false
  await loadStreams()
}

async function removeStream(name) {
  await ElMessageBox.confirm(`确认删除 Stream ${name} ?`, '提示', { type: 'warning' })
  await deleteStream(name)
  ElMessage.success('Stream 已删除')
  if (selected.value === name) {
    selected.value = null
    detail.value = null
  }
  await loadStreams()
}

async function removeSelectedStreams() {
  if (!selectedRows.value.length) {
    ElMessage.warning('请先选择 Stream')
    return
  }
  await ElMessageBox.confirm(`确认批量删除 ${selectedRows.value.length} 个 Stream ?`, '提示', { type: 'warning' })
  const names = selectedRows.value.map((row) => row.name)
  const result = await batchDeleteStreams(names)
  if (names.includes(selected.value)) {
    selected.value = null
    detail.value = null
  }
  selectedRows.value = []
  await loadStreams()
  ElMessage.success(`批量删除完成，成功 ${result.deleted}，失败 ${result.failed}`)
}

onMounted(async () => {
  await loadStreams()
  unsubscribe = onConnectionChanged(async () => {
    selected.value = null
    detail.value = null
    selectedRows.value = []
    page.value = 1
    await loadStreams()
  })
})

onBeforeUnmount(() => {
  unsubscribe?.()
})

watch(pageSize, async () => {
  selectedRows.value = []
  page.value = 1
  await loadStreams()
})

watch(page, async () => {
  selectedRows.value = []
  await loadStreams()
})

watch(
  () => [page.value, pageSize.value, selected.value],
  persistViewState,
  { deep: true },
)
</script>

<template>
  <div class="split-layout">
    <el-card shadow="never" class="split-side">
      <template #header>
        <div class="card-header">
          <span>Streams</span>
          <div class="data-toolbar">
            <el-button type="danger" plain @click="removeSelectedStreams">批量删除</el-button>
            <el-button type="primary" @click="dialogVisible = true">新建</el-button>
          </div>
        </div>
      </template>
      <el-table :data="streams" stripe v-loading="loading" @selection-change="selectedRows = $event">
        <el-table-column type="selection" width="46" />
        <el-table-column prop="name" label="名称" min-width="120">
          <template #default="{ row }">
            <el-link type="primary" @click="selectStream(row.name)">{{ row.name }}</el-link>
          </template>
        </el-table-column>
        <el-table-column prop="messages" label="消息数" width="110" />
        <el-table-column prop="storage" label="存储" width="100" />
        <el-table-column prop="consumers" label="Consumers" width="110" />
        <el-table-column label="操作" width="90">
          <template #default="{ row }">
            <el-button text type="danger" @click="removeStream(row.name)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="table-pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          layout="total, sizes, prev, pager, next"
          :total="total"
          :page-sizes="[8, 12, 20]"
        />
      </div>
    </el-card>

    <el-card shadow="never" class="split-main">
      <template #header>
        <span>Stream 详情</span>
      </template>
      <template v-if="detail">
        <el-descriptions :column="2" border class="mb-16">
          <el-descriptions-item label="Name">{{ detail.stream.name }}</el-descriptions-item>
          <el-descriptions-item label="Storage">{{ detail.stream.storage }}</el-descriptions-item>
          <el-descriptions-item label="Messages">{{ detail.stream.messages }}</el-descriptions-item>
          <el-descriptions-item label="Bytes">{{ detail.stream.bytes }}</el-descriptions-item>
          <el-descriptions-item label="Subjects">{{ detail.stream.subjects.join(', ') }}</el-descriptions-item>
          <el-descriptions-item label="Retention">{{ detail.stream.retention }}</el-descriptions-item>
        </el-descriptions>

        <el-card shadow="never" class="mb-16">
          <template #header>
            <span>Subject 状态</span>
          </template>
          <el-table :data="detail.stream.subjectsState" stripe>
            <el-table-column prop="subject" label="Subject" min-width="180" />
            <el-table-column prop="count" label="消息数" width="120" />
          </el-table>
        </el-card>

        <el-card shadow="never">
          <template #header>
            <span>Consumers</span>
          </template>
          <el-table :data="detail.consumers" stripe>
            <el-table-column prop="name" label="Name" min-width="140" />
            <el-table-column prop="durable" label="Durable" min-width="120" />
            <el-table-column prop="ackPolicy" label="AckPolicy" width="120" />
            <el-table-column prop="pending" label="Pending" width="120" />
            <el-table-column prop="waiting" label="Waiting" width="100" />
            <el-table-column prop="numRedelivered" label="Redelivered" width="120" />
          </el-table>
        </el-card>
      </template>
      <el-empty v-else description="请选择一个 Stream" />
    </el-card>

    <el-dialog v-model="dialogVisible" title="创建 Stream" width="520px">
      <el-form label-width="110px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="Subjects">
          <el-input v-model="form.subjects" placeholder="例如: orders.*, billing.created" />
        </el-form-item>
        <el-form-item label="存储类型">
          <el-select v-model="form.storage" style="width: 100%">
            <el-option label="File" value="file" />
            <el-option label="Memory" value="memory" />
          </el-select>
        </el-form-item>
        <el-form-item label="副本数">
          <el-input-number v-model="form.replicas" :min="1" :max="5" />
        </el-form-item>
        <el-form-item label="最大保留秒">
          <el-input-number v-model="form.maxAgeSec" :min="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitStream">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>
