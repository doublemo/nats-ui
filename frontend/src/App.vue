<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Connection, DataAnalysis, Expand, FolderOpened, Key } from '@element-plus/icons-vue'
import {
  activateConnection,
  getActiveConnectionId,
  getConnections,
  onConnectionChanged,
  setActiveConnectionId,
} from './api/nats'

const route = useRoute()
const connections = ref([])
const activeId = ref(getActiveConnectionId())
const collapsed = ref(false)
let unsubscribe
const appVersion = __APP_VERSION__
const copyrightNotice = __APP_COPYRIGHT__

const VIEW_META = {
  '/connections': {
    title: '连接管理',
    description: '管理 NATS 连接、认证信息与监控地址。',
  },
  '/dashboard': {
    title: '集群概览',
    description: '实时查看节点状态、流量走势与连接详情。',
  },
  '/jetstream': {
    title: 'JetStream 管理',
    description: '查看 Stream、Consumer 与消息存储情况。',
  },
  '/kv': {
    title: 'KV 管理',
    description: '维护 Key-Value Bucket 与条目数据。',
  },
}

const runtime = computed(() => {
  const desktop = window.__NATS_UI_DESKTOP__
  return {
    isElectron: Boolean(desktop?.isElectron),
    platform: desktop?.platform || 'web',
  }
})

const currentView = computed(() => VIEW_META[route.path] || VIEW_META['/dashboard'])
const runtimeLabel = computed(() => (runtime.value.isElectron ? 'Desktop App' : 'Web App'))
const platformLabel = computed(() => {
  const labels = {
    win32: 'Windows',
    darwin: 'macOS',
    linux: 'Linux',
    web: 'Browser',
  }
  return labels[runtime.value.platform] || runtime.value.platform
})
const shellClasses = computed(() => ({
  'is-electron': runtime.value.isElectron,
  'is-web': !runtime.value.isElectron,
}))
const versionLabel = computed(() => `v${appVersion}`)

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
  <el-container class="layout-shell" :class="shellClasses">
    <el-aside :width="collapsed ? '76px' : '240px'" class="sidebar">
      <div class="brand" :class="{ collapsed }">
        <span class="brand-mark">N</span>
        <div v-if="!collapsed" class="brand-copy">
          <span class="brand-text">NATS UI</span>
          <span class="brand-subtext">{{ runtimeLabel }}</span>
        </div>
      </div>
      <el-menu :default-active="$route.path" router class="nav-menu" :collapse="collapsed">
        <el-menu-item index="/connections">
          <el-icon><Connection /></el-icon>
          <span>连接管理</span>
        </el-menu-item>
        <el-menu-item index="/dashboard">
          <el-icon><DataAnalysis /></el-icon>
          <span>集群概览</span>
        </el-menu-item>
        <el-menu-item index="/jetstream">
          <el-icon><FolderOpened /></el-icon>
          <span>JetStream 管理</span>
        </el-menu-item>
        <el-menu-item index="/kv">
          <el-icon><Key /></el-icon>
          <span>KV 管理</span>
        </el-menu-item>
      </el-menu>
      <div class="sidebar-footer" :class="{ collapsed }">
        <div class="sidebar-meta-pills">
          <span class="shell-pill">{{ platformLabel }}</span>
          <span class="shell-pill is-muted">{{ versionLabel }}</span>
        </div>
        <div v-if="!collapsed" class="sidebar-meta">
          <div class="sidebar-stats">
            <strong>{{ connections.length }}</strong>
            <span>Profiles</span>
          </div>
          <span class="sidebar-license">{{ copyrightNotice }}</span>
        </div>
      </div>
    </el-aside>

    <el-container class="content-shell">
      <el-header class="topbar" height="auto">
        <div class="topbar-title">
          <el-button circle plain class="collapse-btn app-control" @click="collapsed = !collapsed">
            <el-icon class="collapse-icon" :class="{ collapsed }"><Expand /></el-icon>
          </el-button>
          <div class="topbar-heading">
            <div class="topbar-kicker">
              <span class="view-pill is-primary">{{ currentView.title }}</span>
              <span class="view-pill">{{ runtimeLabel }}</span>
              <span v-if="runtime.isElectron" class="view-pill">{{ platformLabel }}</span>
            </div>
            <h1>NATS Server 可视化管理工具</h1>
            <p>{{ currentView.description }}</p>
          </div>
        </div>
        <div class="topbar-actions app-control">
          <div class="connection-panel">
            <span class="connection-panel-label">当前连接</span>
            <strong class="active-connection">{{ currentLabel }}</strong>
          </div>
          <el-select :model-value="activeId" placeholder="选择连接" class="connection-switch" @change="handleSwitch">
            <el-option v-for="item in connections" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </div>
      </el-header>
      <el-main class="page-body">
        <div class="page-content">
          <router-view />
        </div>
      </el-main>
    </el-container>
  </el-container>
</template>
