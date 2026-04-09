<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  activateConnection,
  getActiveConnectionId,
  getConnections,
  onConnectionChanged,
  setActiveConnectionId,
} from './api/nats'

const connections = ref([])
const activeId = ref(getActiveConnectionId())
let unsubscribe

const currentLabel = computed(() => {
  const item = connections.value.find((entry) => entry.id === activeId.value)
  return item?.name || '未选择连接'
})

async function loadConnections() {
  const data = await getConnections()
  connections.value = data.items
  if (data.activeId && !activeId.value) {
    activeId.value = data.activeId
    setActiveConnectionId(data.activeId)
  }
}

async function handleSwitch(id) {
  await activateConnection(id)
  activeId.value = id
  setActiveConnectionId(id)
  ElMessage.success('当前连接已切换')
}

onMounted(() => {
  loadConnections()
  unsubscribe = onConnectionChanged((event) => {
    activeId.value = event.detail || getActiveConnectionId()
  })
})

onBeforeUnmount(() => {
  unsubscribe?.()
})
</script>

<template>
  <el-container class="layout-shell">
    <el-aside width="220px" class="sidebar">
      <div class="brand">NATS UI</div>
      <el-menu :default-active="$route.path" router class="nav-menu">
        <el-menu-item index="/connections">连接管理</el-menu-item>
        <el-menu-item index="/dashboard">集群概览</el-menu-item>
        <el-menu-item index="/jetstream">JetStream 管理</el-menu-item>
        <el-menu-item index="/kv">KV 管理</el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="topbar">
        <div>
          <h1>NATS Server 可视化管理工具</h1>
          <p>Golang + Vue 3 + JetStream</p>
        </div>
        <div class="topbar-actions">
          <span class="active-connection">{{ currentLabel }}</span>
          <el-select :model-value="activeId" placeholder="选择连接" style="width: 260px" @change="handleSwitch">
            <el-option v-for="item in connections" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </div>
      </el-header>
      <el-main class="page-body">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>
