<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  activateConnection,
  createConnection,
  deleteConnection,
  getActiveConnectionId,
  getConnections,
  setActiveConnectionId,
  updateConnection,
} from '../api/nats'

const loading = ref(false)
const dialogVisible = ref(false)
const editingId = ref('')
const activeId = ref(getActiveConnectionId())
const connections = ref([])
const form = reactive({
  name: '',
  natsUrls: '',
  monitorEndpoints: '',
  username: '',
  password: '',
  token: '',
})

async function loadConnections() {
  loading.value = true
  try {
    const data = await getConnections()
    connections.value = data.items
    activeId.value = data.activeId
    if (data.activeId) {
      setActiveConnectionId(data.activeId)
    }
  } finally {
    loading.value = false
  }
}

function resetForm() {
  editingId.value = ''
  form.name = ''
  form.natsUrls = ''
  form.monitorEndpoints = ''
  form.username = ''
  form.password = ''
  form.token = ''
}

function openCreate() {
  resetForm()
  dialogVisible.value = true
}

function openEdit(row) {
  editingId.value = row.id
  form.name = row.name
  form.natsUrls = row.natsUrls.join(', ')
  form.monitorEndpoints = row.monitorEndpoints.join(', ')
  form.username = row.username || ''
  form.password = row.password || ''
  form.token = row.token || ''
  dialogVisible.value = true
}

async function submit() {
  const payload = {
    name: form.name,
    natsUrls: form.natsUrls.split(',').map((item) => item.trim()).filter(Boolean),
    monitorEndpoints: form.monitorEndpoints.split(',').map((item) => item.trim()).filter(Boolean),
    username: form.username,
    password: form.password,
    token: form.token,
  }

  if (editingId.value) {
    await updateConnection(editingId.value, payload)
    ElMessage.success('连接已更新')
  } else {
    await createConnection(payload)
    ElMessage.success('连接已创建')
  }
  dialogVisible.value = false
  await loadConnections()
}

async function switchActive(id) {
  await activateConnection(id)
  setActiveConnectionId(id)
  activeId.value = id
  ElMessage.success('已切换当前连接')
  await loadConnections()
}

async function removeConnection(id, name) {
  await ElMessageBox.confirm(`确认删除连接 ${name} ?`, '提示', { type: 'warning' })
  const data = await deleteConnection(id)
  setActiveConnectionId(data.activeId || '')
  ElMessage.success('连接已删除')
  await loadConnections()
}

onMounted(loadConnections)
</script>

<template>
  <el-card shadow="never">
    <template #header>
      <div class="card-header">
        <span>NATS 服务器连接管理</span>
        <el-button type="primary" @click="openCreate">新增连接</el-button>
      </div>
    </template>

    <el-table :data="connections" stripe v-loading="loading">
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column label="NATS 地址" min-width="220">
        <template #default="{ row }">{{ row.natsUrls.join(', ') }}</template>
      </el-table-column>
      <el-table-column label="监控地址" min-width="220">
        <template #default="{ row }">{{ row.monitorEndpoints.join(', ') || '-' }}</template>
      </el-table-column>
      <el-table-column prop="status" label="连接状态" width="120" />
      <el-table-column prop="connectedUrl" label="当前连接" min-width="180" />
      <el-table-column label="激活" width="100">
        <template #default="{ row }">
          <el-tag :type="row.id === activeId ? 'success' : 'info'">{{ row.id === activeId ? '当前' : '待用' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="220">
        <template #default="{ row }">
          <el-button text type="primary" @click="switchActive(row.id)">切换</el-button>
          <el-button text @click="openEdit(row)">编辑</el-button>
          <el-button text type="danger" @click="removeConnection(row.id, row.name)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="dialogVisible" :title="editingId ? '编辑连接' : '新增连接'" width="640px">
    <el-form label-width="110px">
      <el-form-item label="连接名称">
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="NATS URLs">
        <el-input v-model="form.natsUrls" placeholder="nats://127.0.0.1:4222, nats://127.0.0.1:4223" />
      </el-form-item>
      <el-form-item label="Monitor URLs">
        <el-input v-model="form.monitorEndpoints" placeholder="http://127.0.0.1:8222, http://127.0.0.1:8223" />
      </el-form-item>
      <el-form-item label="用户名">
        <el-input v-model="form.username" />
      </el-form-item>
      <el-form-item label="密码">
        <el-input v-model="form.password" type="password" show-password />
      </el-form-item>
      <el-form-item label="Token">
        <el-input v-model="form.token" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" @click="submit">保存</el-button>
    </template>
  </el-dialog>
</template>
