<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
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

const { t } = useI18n()
const buckets = ref([])
const bucketTotal = ref(0)
const selectedBucket = ref('')
const selectedBucketRows = ref([])
const selectedEntryRows = ref([])
const entries = ref([])
const entryTotal = ref(0)
const loading = ref(false)
const refreshing = ref(false)
const bucketDialog = ref(false)
const entryDialog = ref(false)
const bucketPage = ref(1)
const bucketPageSize = ref(8)
const entryPage = ref(1)
const entryPageSize = ref(10)
const bucketKeyword = ref('')
const entryKeyword = ref('')
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

const bucketHeader = computed(() =>
  t('kv.currentBucket', {
    name: selectedBucket.value || t('kv.noBucketSelected'),
  }),
)

restoreViewState()

async function loadBuckets() {
  const data = await getBuckets({
    page: bucketPage.value,
    pageSize: bucketPageSize.value,
    keyword: bucketKeyword.value || undefined,
  })
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
    bucketKeyword.value = saved.bucketKeyword || ''
    entryKeyword.value = saved.entryKeyword || ''
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
      bucketKeyword: bucketKeyword.value,
      entryKeyword: entryKeyword.value,
    }),
  )
}

async function loadEntries() {
  if (!selectedBucket.value) {
    entries.value = []
    entryTotal.value = 0
    return
  }
  loading.value = true
  try {
    const data = await getBucketEntries(selectedBucket.value, {
      page: entryPage.value,
      pageSize: entryPageSize.value,
      keyword: entryKeyword.value || undefined,
    })
    entries.value = data.items
    entryTotal.value = data.total
  } finally {
    loading.value = false
  }
}

async function refreshData() {
  refreshing.value = true
  try {
    await loadBuckets()
    await loadEntries()
  } finally {
    refreshing.value = false
  }
}

async function submitBucket() {
  await createBucket(bucketForm)
  ElMessage.success(t('kv.messages.createBucketSuccess'))
  bucketDialog.value = false
  await loadBuckets()
}

async function submitEntry() {
  await putBucketEntry(selectedBucket.value, entryForm.key, entryForm.value)
  ElMessage.success(t('kv.messages.saveEntrySuccess'))
  entryDialog.value = false
  await loadBuckets()
  await loadEntries()
}

async function removeBucket(name) {
  await ElMessageBox.confirm(t('kv.messages.deleteBucketConfirm', { name }), t('common.prompt'), { type: 'warning' })
  await deleteBucket(name)
  ElMessage.success(t('kv.messages.bucketDeletedSuccess'))
  if (selectedBucket.value === name) {
    selectedBucket.value = ''
  }
  await loadBuckets()
  await loadEntries()
}

async function removeEntry(key) {
  await ElMessageBox.confirm(t('kv.messages.deleteEntryConfirm', { key }), t('common.prompt'), { type: 'warning' })
  await deleteBucketEntry(selectedBucket.value, key)
  ElMessage.success(t('kv.messages.entryDeletedSuccess'))
  await loadBuckets()
  await loadEntries()
}

async function removeSelectedBuckets() {
  if (!selectedBucketRows.value.length) {
    ElMessage.warning(t('kv.messages.selectBucketWarning'))
    return
  }
  await ElMessageBox.confirm(
    t('kv.messages.batchDeleteBucketsConfirm', { count: selectedBucketRows.value.length }),
    t('common.prompt'),
    { type: 'warning' },
  )
  const names = selectedBucketRows.value.map((row) => row.name)
  const result = await batchDeleteBuckets(names)
  if (names.includes(selectedBucket.value)) {
    selectedBucket.value = ''
  }
  selectedBucketRows.value = []
  await loadBuckets()
  await loadEntries()
  ElMessage.success(t('kv.messages.batchDeleteBucketsSummary', { deleted: result.deleted, failed: result.failed }))
}

async function removeSelectedEntries() {
  if (!selectedEntryRows.value.length || !selectedBucket.value) {
    ElMessage.warning(t('kv.messages.selectEntryWarning'))
    return
  }
  await ElMessageBox.confirm(
    t('kv.messages.batchDeleteEntriesConfirm', { count: selectedEntryRows.value.length }),
    t('common.prompt'),
    { type: 'warning' },
  )
  const result = await batchDeleteBucketEntries(selectedBucket.value, selectedEntryRows.value.map((row) => row.key))
  selectedEntryRows.value = []
  await loadBuckets()
  await loadEntries()
  ElMessage.success(t('kv.messages.batchDeleteEntriesSummary', { deleted: result.deleted, failed: result.failed }))
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

watch(bucketKeyword, async () => {
  selectedBucketRows.value = []
  if (bucketPage.value !== 1) {
    bucketPage.value = 1
    return
  }
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

watch(entryKeyword, async () => {
  selectedEntryRows.value = []
  if (entryPage.value !== 1) {
    entryPage.value = 1
    return
  }
  await loadEntries()
})

onMounted(async () => {
  await refreshData()
  unsubscribe = onConnectionChanged(async () => {
    selectedBucket.value = ''
    entries.value = []
    selectedBucketRows.value = []
    selectedEntryRows.value = []
    bucketPage.value = 1
    entryPage.value = 1
    await refreshData()
  })
})

onBeforeUnmount(() => {
  unsubscribe?.()
})

watch(
  () => [
    selectedBucket.value,
    bucketPage.value,
    bucketPageSize.value,
    entryPage.value,
    entryPageSize.value,
    bucketKeyword.value,
    entryKeyword.value,
  ],
  persistViewState,
  { deep: true },
)
</script>

<template>
  <div class="split-layout">
    <el-card shadow="never" class="split-side">
      <template #header>
        <div class="card-header">
          <span>{{ t('kv.buckets') }}</span>
          <div class="data-toolbar">
            <el-button :loading="refreshing" @click="refreshData">{{ t('common.refresh') }}</el-button>
            <el-button type="danger" plain @click="removeSelectedBuckets">{{ t('kv.batchDelete') }}</el-button>
            <el-button type="primary" @click="bucketDialog = true">{{ t('kv.create') }}</el-button>
          </div>
        </div>
      </template>
      <el-input v-model="bucketKeyword" :placeholder="t('kv.bucketSearchPlaceholder')" clearable class="mb-16" />
      <el-table :data="buckets" stripe @selection-change="selectedBucketRows = $event">
        <el-table-column type="selection" width="46" />
        <el-table-column prop="name" :label="t('kv.buckets')" min-width="130">
          <template #default="{ row }">
            <el-link type="primary" @click="selectedBucket = row.name">{{ row.name }}</el-link>
          </template>
        </el-table-column>
        <el-table-column prop="values" :label="t('kv.keyCount')" width="90" />
        <el-table-column prop="storage" :label="t('kv.storage')" width="90" />
        <el-table-column :label="t('kv.actions')" width="90">
          <template #default="{ row }">
            <el-button text type="danger" @click="removeBucket(row.name)">{{ t('common.delete') }}</el-button>
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
          <span>{{ bucketHeader }}</span>
          <div class="data-toolbar">
            <el-button type="danger" plain :disabled="!selectedBucket" @click="removeSelectedEntries">{{ t('kv.batchDelete') }}</el-button>
            <el-button type="primary" :disabled="!selectedBucket" @click="openEntryDialog()">{{ t('kv.createEntry') }}</el-button>
          </div>
        </div>
      </template>
      <el-input
        v-model="entryKeyword"
        :placeholder="t('kv.entrySearchPlaceholder')"
        clearable
        class="mb-16"
        :disabled="!selectedBucket"
      />

      <el-table :data="entries" stripe v-loading="loading" @selection-change="selectedEntryRows = $event">
        <el-table-column type="selection" width="46" />
        <el-table-column prop="key" :label="t('kv.entryKey')" min-width="180" />
        <el-table-column prop="value" :label="t('kv.entryValue')" min-width="240" show-overflow-tooltip />
        <el-table-column prop="revision" :label="t('kv.revision')" width="100" />
        <el-table-column prop="createdAt" :label="t('kv.updatedAt')" width="180" />
        <el-table-column :label="t('kv.actions')" width="150">
          <template #default="{ row }">
            <el-button text type="primary" @click="openEntryDialog(row)">{{ t('common.edit') }}</el-button>
            <el-button text type="danger" @click="removeEntry(row.key)">{{ t('common.delete') }}</el-button>
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

    <el-dialog v-model="bucketDialog" :title="t('kv.createBucketDialogTitle')" width="520px">
      <el-form label-width="100px">
        <el-form-item :label="t('kv.fields.name')">
          <el-input v-model="bucketForm.name" />
        </el-form-item>
        <el-form-item :label="t('kv.fields.description')">
          <el-input v-model="bucketForm.description" />
        </el-form-item>
        <el-form-item :label="t('kv.fields.history')">
          <el-input-number v-model="bucketForm.history" :min="1" :max="64" />
        </el-form-item>
        <el-form-item :label="t('kv.fields.storageType')">
          <el-select v-model="bucketForm.storage" style="width: 100%">
            <el-option label="File" value="file" />
            <el-option label="Memory" value="memory" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="bucketDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="submitBucket">{{ t('common.create') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="entryDialog" :title="t('kv.entryDialogTitle')" width="560px">
      <el-form label-width="90px">
        <el-form-item :label="t('kv.fields.key')">
          <el-input v-model="entryForm.key" />
        </el-form-item>
        <el-form-item :label="t('kv.fields.value')">
          <el-input v-model="entryForm.value" type="textarea" :rows="8" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="entryDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="submitEntry">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>
