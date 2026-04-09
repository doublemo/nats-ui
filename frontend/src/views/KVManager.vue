<script setup>
import { onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  createBucket,
  deleteBucket,
  deleteBucketEntry,
  getBucketEntries,
  getBuckets,
  onConnectionChanged,
  putBucketEntry,
} from '../api/nats'

const buckets = ref([])
const selectedBucket = ref('')
const entries = ref([])
const loading = ref(false)
const bucketDialog = ref(false)
const entryDialog = ref(false)
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

async function loadBuckets() {
  buckets.value = await getBuckets()
  if (!selectedBucket.value && buckets.value.length > 0) {
    selectedBucket.value = buckets.value[0].name
  }
}

async function loadEntries() {
  if (!selectedBucket.value) {
    entries.value = []
    return
  }
  loading.value = true
  try {
    entries.value = await getBucketEntries(selectedBucket.value)
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

function openEntryDialog(row) {
  entryForm.key = row?.key || ''
  entryForm.value = row?.value || ''
  entryDialog.value = true
}

watch(selectedBucket, loadEntries)
onMounted(async () => {
  await loadBuckets()
  await loadEntries()
  unsubscribe = onConnectionChanged(async () => {
    selectedBucket.value = ''
    entries.value = []
    await loadBuckets()
    await loadEntries()
  })
})

onBeforeUnmount(() => {
  unsubscribe?.()
})
</script>

<template>
  <div class="split-layout">
    <el-card shadow="never" class="split-side">
      <template #header>
        <div class="card-header">
          <span>KV Buckets</span>
          <el-button type="primary" @click="bucketDialog = true">新建</el-button>
        </div>
      </template>
      <el-table :data="buckets" stripe>
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
    </el-card>

    <el-card shadow="never" class="split-main">
      <template #header>
        <div class="card-header">
          <span>Bucket: {{ selectedBucket || '未选择' }}</span>
          <el-button type="primary" :disabled="!selectedBucket" @click="openEntryDialog()">新增键值</el-button>
        </div>
      </template>

      <el-table :data="entries" stripe v-loading="loading">
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
