<script setup>
import { onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  batchDeleteBucketEntries,
  batchDeleteBuckets,
  createBucket,
  deleteBucket,
  deleteBucketEntry,
  getBucketEntries,
  getBuckets,
  onConnectionChanged,
  putBucketEntry,
} from '../api/nats'

const KV_VIEW_STATE_KEY = 'nats-ui-kv-view-state'

const buckets = ref([])
const bucketTotal = ref(0)
const selectedBucket = ref('')
const selectedBucketRows = ref([])
const selectedEntryRows = ref([])
const entries = ref([])
const entryTotal = ref(0)
const loading = ref(false)
const bucketDialog = ref(false)
const entryDialog = ref(false)
const bucketPage = ref(1)
const bucketPageSize = ref(8)
const entryPage = ref(1)
const entryPageSize = ref(10)
const bucketForm = reactive({
  name: '',
  description: '',
  history: 1,
  storage: 'file',
})
let unsubscribe
const entryForm = reactive({
  key: '',
  value: '',
})

restoreViewState()

async function loadBuckets() {
  const data = await getBuckets({ page: bucketPage.value, pageSize: bucketPageSize.value })
  buckets.value = data.items
  bucketTotal.value = data.total
  if (!selectedBucket.value && buckets.value.length > 0) {
    selectedBucket.value = buckets.value[0].name
  }
}

function restoreViewState() {
  try {
    const raw = window.localStorage.getItem(KV_VIEW_STATE_KEY)
    if (!raw) return
    const saved = JSON.parse(raw)
    selectedBucket.value = saved.selectedBucket || ''
    bucketPage.value = saved.bucketPage || 1
    bucketPageSize.value = saved.bucketPageSize || 8
    entryPage.value = saved.entryPage || 1
    entryPageSize.value = saved.entryPageSize || 10
  } catch {
    window.localStorage.removeItem(KV_VIEW_STATE_KEY)
  }
}

function persistViewState() {
  window.localStorage.setItem(
    KV_VIEW_STATE_KEY,
    JSON.stringify({
      selectedBucket: selectedBucket.value,
      bucketPage: bucketPage.value,
      bucketPageSize: bucketPageSize.value,
      entryPage: entryPage.value,
      entryPageSize: entryPageSize.value,
    }),
  )
}

async function loadEntries() {
  if (!selectedBucket.value) {
    entries.value = []
    return
  }
  loading.value = true
  try {
    const data = await getBucketEntries(selectedBucket.value, { page: entryPage.value, pageSize: entryPageSize.value })
    entries.value = data.items
    entryTotal.value = data.total
  } finally {
    loading.value = false
  }
}

async function submitBucket() {
  await createBucket(bucketForm)
  ElMessage.success('Bucket 创建成功')
  bucketDialog.value = false
  await loadBuckets()
}

async function submitEntry() {
  await putBucketEntry(selectedBucket.value, entryForm.key, entryForm.value)
  ElMessage.success('键值已保存')
  entryDialog.value = false
  await loadEntries()
}

async function removeBucket(name) {
  await ElMessageBox.confirm(`确认删除 Bucket ${name} ?`, '提示', { type: 'warning' })
  await deleteBucket(name)
  ElMessage.success('Bucket 已删除')
  if (selectedBucket.value === name) {
    selectedBucket.value = ''
  }
  await loadBuckets()
  await loadEntries()
}

async function removeEntry(key) {
  await ElMessageBox.confirm(`确认删除键 ${key} ?`, '提示', { type: 'warning' })
  await deleteBucketEntry(selectedBucket.value, key)
  ElMessage.success('键已删除')
  await loadEntries()
}

async function removeSelectedBuckets() {
  if (!selectedBucketRows.value.length) {
    ElMessage.warning('请先选择 Bucket')
    return
  }
  await ElMessageBox.confirm(`确认批量删除 ${selectedBucketRows.value.length} 个 Bucket ?`, '提示', { type: 'warning' })
  const names = selectedBucketRows.value.map((row) => row.name)
  const result = await batchDeleteBuckets(names)
  if (names.includes(selectedBucket.value)) {
    selectedBucket.value = ''
  }
  selectedBucketRows.value = []
  await loadBuckets()
  await loadEntries()
  ElMessage.success(`批量删除完成，成功 ${result.deleted}，失败 ${result.failed}`)
}

async function removeSelectedEntries() {
  if (!selectedEntryRows.value.length || !selectedBucket.value) {
    ElMessage.warning('请先选择键值')
    return
  }
  await ElMessageBox.confirm(`确认批量删除 ${selectedEntryRows.value.length} 个键 ?`, '提示', { type: 'warning' })
  const result = await batchDeleteBucketEntries(selectedBucket.value, selectedEntryRows.value.map((row) => row.key))
  selectedEntryRows.value = []
  await loadEntries()
  ElMessage.success(`批量删除完成，成功 ${result.deleted}，失败 ${result.failed}`)
}

function openEntryDialog(row) {
  entryForm.key = row?.key || ''
  entryForm.value = row?.value || ''
  entryDialog.value = true
}

watch(selectedBucket, async () => {
  entryPage.value = 1
  selectedEntryRows.value = []
  await loadEntries()
})

watch(bucketPageSize, async () => {
  bucketPage.value = 1
  selectedBucketRows.value = []
  await loadBuckets()
})

watch(bucketPage, async () => {
  selectedBucketRows.value = []
  await loadBuckets()
})

watch(entryPageSize, async () => {
  entryPage.value = 1
  selectedEntryRows.value = []
  await loadEntries()
})

watch(entryPage, async () => {
  selectedEntryRows.value = []
  await loadEntries()
})

onMounted(async () => {
  await loadBuckets()
  await loadEntries()
  unsubscribe = onConnectionChanged(async () => {
    selectedBucket.value = ''
    entries.value = []
    selectedBucketRows.value = []
    selectedEntryRows.value = []
    bucketPage.value = 1
    entryPage.value = 1
    await loadBuckets()
    await loadEntries()
  })
})

onBeforeUnmount(() => {
  unsubscribe?.()
})

watch(
  () => [selectedBucket.value, bucketPage.value, bucketPageSize.value, entryPage.value, entryPageSize.value],
  persistViewState,
  { deep: true },
)
</script>

<template>
  <div class="split-layout">
    <el-card shadow="never" class="split-side">
      <template #header>
        <div class="card-header">
          <span>KV Buckets</span>
          <div class="data-toolbar">
            <el-button type="danger" plain @click="removeSelectedBuckets">批量删除</el-button>
            <el-button type="primary" @click="bucketDialog = true">新建</el-button>
          </div>
        </div>
      </template>
      <el-table :data="buckets" stripe @selection-change="selectedBucketRows = $event">
        <el-table-column type="selection" width="46" />
        <el-table-column prop="name" label="Bucket" min-width="130">
          <template #default="{ row }">
            <el-link type="primary" @click="selectedBucket = row.name">{{ row.name }}</el-link>
          </template>
        </el-table-column>
        <el-table-column prop="values" label="键数量" width="90" />
        <el-table-column prop="storage" label="存储" width="90" />
        <el-table-column label="操作" width="90">
          <template #default="{ row }">
            <el-button text type="danger" @click="removeBucket(row.name)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="table-pagination">
        <el-pagination
          v-model:current-page="bucketPage"
          v-model:page-size="bucketPageSize"
          layout="total, sizes, prev, pager, next"
          :total="bucketTotal"
          :page-sizes="[8, 12, 20]"
        />
      </div>
    </el-card>

    <el-card shadow="never" class="split-main">
      <template #header>
        <div class="card-header">
          <span>Bucket: {{ selectedBucket || '未选择' }}</span>
          <div class="data-toolbar">
            <el-button type="danger" plain :disabled="!selectedBucket" @click="removeSelectedEntries">批量删除</el-button>
            <el-button type="primary" :disabled="!selectedBucket" @click="openEntryDialog()">新增键值</el-button>
          </div>
        </div>
      </template>

      <el-table :data="entries" stripe v-loading="loading" @selection-change="selectedEntryRows = $event">
        <el-table-column type="selection" width="46" />
        <el-table-column prop="key" label="Key" min-width="180" />
        <el-table-column prop="value" label="Value" min-width="240" show-overflow-tooltip />
        <el-table-column prop="revision" label="Revision" width="100" />
        <el-table-column prop="createdAt" label="更新时间" width="180" />
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <el-button text type="primary" @click="openEntryDialog(row)">编辑</el-button>
            <el-button text type="danger" @click="removeEntry(row.key)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="table-pagination">
        <el-pagination
          v-model:current-page="entryPage"
          v-model:page-size="entryPageSize"
          layout="total, sizes, prev, pager, next"
          :total="entryTotal"
          :page-sizes="[10, 20, 50]"
        />
      </div>
    </el-card>

    <el-dialog v-model="bucketDialog" title="创建 Bucket" width="520px">
      <el-form label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="bucketForm.name" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="bucketForm.description" />
        </el-form-item>
        <el-form-item label="History">
          <el-input-number v-model="bucketForm.history" :min="1" :max="64" />
        </el-form-item>
        <el-form-item label="存储类型">
          <el-select v-model="bucketForm.storage" style="width: 100%">
            <el-option label="File" value="file" />
            <el-option label="Memory" value="memory" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="bucketDialog = false">取消</el-button>
        <el-button type="primary" @click="submitBucket">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="entryDialog" title="新增 / 编辑键值" width="560px">
      <el-form label-width="90px">
        <el-form-item label="Key">
          <el-input v-model="entryForm.key" />
        </el-form-item>
        <el-form-item label="Value">
          <el-input v-model="entryForm.value" type="textarea" :rows="8" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="entryDialog = false">取消</el-button>
        <el-button type="primary" @click="submitEntry">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>
