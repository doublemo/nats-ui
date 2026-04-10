<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { batchDeleteStreams, createStream, deleteStream, getStreamDetail, getStreams, onConnectionChanged } from '../api/nats'

const JETSTREAM_VIEW_STATE_KEY = 'nats-ui-jetstream-view-state'

const { t } = useI18n()
const streams = ref([])
const total = ref(0)
const loading = ref(false)
const selected = ref(null)
const selectedRows = ref([])
const detail = ref(null)
const dialogVisible = ref(false)
const page = ref(1)
const pageSize = ref(8)
const keyword = ref('')
let unsubscribe
const form = reactive({
  name: '',
  subjects: '',
  storage: 'file',
  replicas: 1,
  maxAgeSec: 0,
})

const detailItems = computed(() => [
  { label: t('jetstream.detail.name'), value: detail.value?.stream.name || '-' },
  { label: t('jetstream.detail.storage'), value: detail.value?.stream.storage || '-' },
  { label: t('jetstream.detail.messages'), value: detail.value?.stream.messages || 0 },
  { label: t('jetstream.detail.bytes'), value: detail.value?.stream.bytes || 0 },
  { label: t('jetstream.detail.subjects'), value: detail.value?.stream.subjects?.join(', ') || '-' },
  { label: t('jetstream.detail.retention'), value: detail.value?.stream.retention || '-' },
])

restoreViewState()

async function loadStreams() {
  loading.value = true
  try {
    const data = await getStreams({
      page: page.value,
      pageSize: pageSize.value,
      keyword: keyword.value || undefined,
    })
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
    keyword.value = saved.keyword || ''
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
      keyword: keyword.value,
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
  ElMessage.success(t('jetstream.messagesText.createSuccess'))
  dialogVisible.value = false
  await loadStreams()
}

async function removeStream(name) {
  await ElMessageBox.confirm(t('jetstream.messagesText.deleteConfirm', { name }), t('common.prompt'), { type: 'warning' })
  await deleteStream(name)
  ElMessage.success(t('jetstream.messagesText.deletedSuccess'))
  if (selected.value === name) {
    selected.value = null
    detail.value = null
  }
  await loadStreams()
}

async function removeSelectedStreams() {
  if (!selectedRows.value.length) {
    ElMessage.warning(t('jetstream.messagesText.selectWarning'))
    return
  }
  await ElMessageBox.confirm(
    t('jetstream.messagesText.batchDeleteConfirm', { count: selectedRows.value.length }),
    t('common.prompt'),
    { type: 'warning' },
  )
  const names = selectedRows.value.map((row) => row.name)
  const result = await batchDeleteStreams(names)
  if (names.includes(selected.value)) {
    selected.value = null
    detail.value = null
  }
  selectedRows.value = []
  await loadStreams()
  ElMessage.success(t('jetstream.messagesText.batchDeleteSummary', { deleted: result.deleted, failed: result.failed }))
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

watch(keyword, async () => {
  selectedRows.value = []
  if (page.value !== 1) {
    page.value = 1
    return
  }
  await loadStreams()
})

watch(
  () => [page.value, pageSize.value, keyword.value, selected.value],
  persistViewState,
  { deep: true },
)
</script>

<template>
  <div class="split-layout">
    <el-card shadow="never" class="split-side">
      <template #header>
        <div class="card-header">
          <span>{{ t('jetstream.streams') }}</span>
          <div class="data-toolbar">
            <el-button type="danger" plain @click="removeSelectedStreams">{{ t('jetstream.batchDelete') }}</el-button>
            <el-button type="primary" @click="dialogVisible = true">{{ t('jetstream.create') }}</el-button>
          </div>
        </div>
      </template>
      <el-input v-model="keyword" :placeholder="t('jetstream.searchPlaceholder')" clearable class="mb-16" />
      <el-table :data="streams" stripe v-loading="loading" @selection-change="selectedRows = $event">
        <el-table-column type="selection" width="46" />
        <el-table-column prop="name" :label="t('jetstream.name')" min-width="120">
          <template #default="{ row }">
            <el-link type="primary" @click="selectStream(row.name)">{{ row.name }}</el-link>
          </template>
        </el-table-column>
        <el-table-column prop="messages" :label="t('jetstream.messages')" width="110" />
        <el-table-column prop="storage" :label="t('jetstream.storage')" width="100" />
        <el-table-column prop="consumers" :label="t('jetstream.consumers')" width="110" />
        <el-table-column :label="t('jetstream.actions')" width="90">
          <template #default="{ row }">
            <el-button text type="danger" @click="removeStream(row.name)">{{ t('common.delete') }}</el-button>
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
        <span>{{ t('jetstream.detailsTitle') }}</span>
      </template>
      <template v-if="detail">
        <el-descriptions :column="2" border class="mb-16">
          <el-descriptions-item v-for="item in detailItems" :key="item.label" :label="item.label">
            {{ item.value }}
          </el-descriptions-item>
        </el-descriptions>

        <el-card shadow="never" class="mb-16">
          <template #header>
            <span>{{ t('jetstream.subjectStatus') }}</span>
          </template>
          <el-table :data="detail.stream.subjectsState" stripe>
            <el-table-column prop="subject" :label="t('jetstream.detail.subjects')" min-width="180" />
            <el-table-column prop="count" :label="t('jetstream.messages')" width="120" />
          </el-table>
        </el-card>

        <el-card shadow="never">
          <template #header>
            <span>{{ t('jetstream.consumers') }}</span>
          </template>
          <el-table :data="detail.consumers" stripe>
            <el-table-column prop="name" :label="t('jetstream.detail.name')" min-width="140" />
            <el-table-column prop="durable" :label="t('jetstream.detail.durable')" min-width="120" />
            <el-table-column prop="ackPolicy" :label="t('jetstream.detail.ackPolicy')" width="120" />
            <el-table-column prop="pending" :label="t('jetstream.detail.pending')" width="120" />
            <el-table-column prop="waiting" :label="t('jetstream.detail.waiting')" width="100" />
            <el-table-column prop="numRedelivered" :label="t('jetstream.detail.redelivered')" width="120" />
          </el-table>
        </el-card>
      </template>
      <el-empty v-else :description="t('jetstream.noSelection')" />
    </el-card>

    <el-dialog v-model="dialogVisible" :title="t('jetstream.createDialogTitle')" width="520px">
      <el-form label-width="110px">
        <el-form-item :label="t('jetstream.fields.name')">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item :label="t('jetstream.fields.subjects')">
          <el-input v-model="form.subjects" :placeholder="t('jetstream.placeholders.subjects')" />
        </el-form-item>
        <el-form-item :label="t('jetstream.fields.storageType')">
          <el-select v-model="form.storage" style="width: 100%">
            <el-option label="File" value="file" />
            <el-option label="Memory" value="memory" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('jetstream.fields.replicas')">
          <el-input-number v-model="form.replicas" :min="1" :max="5" />
        </el-form-item>
        <el-form-item :label="t('jetstream.fields.maxAgeSec')">
          <el-input-number v-model="form.maxAgeSec" :min="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="submitStream">{{ t('common.create') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>
