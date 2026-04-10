<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import en from 'element-plus/es/locale/lang/en'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { useI18n } from 'vue-i18n'
import { Connection, DataAnalysis, Expand, FolderOpened, Key } from '@element-plus/icons-vue'
import {
  activateConnection,
  getActiveConnectionId,
  getConnections,
  onConnectionChanged,
  setActiveConnectionId,
} from './api/nats'
import { setAppLocale, SUPPORTED_LOCALES } from './i18n'

const VIEW_META = {
  '/connections': {
    titleKey: 'layout.views.connections.title',
    descriptionKey: 'layout.views.connections.description',
  },
  '/dashboard': {
    titleKey: 'layout.views.dashboard.title',
    descriptionKey: 'layout.views.dashboard.description',
  },
  '/jetstream': {
    titleKey: 'layout.views.jetstream.title',
    descriptionKey: 'layout.views.jetstream.description',
  },
  '/kv': {
    titleKey: 'layout.views.kv.title',
    descriptionKey: 'layout.views.kv.description',
  },
}

const route = useRoute()
const { t, locale } = useI18n()
const connections = ref([])
const activeId = ref(getActiveConnectionId())
const collapsed = ref(false)
let unsubscribe
const appVersion = __APP_VERSION__
const copyrightNotice = __APP_COPYRIGHT__

const runtime = computed(() => {
  const desktop = window.__NATS_UI_DESKTOP__
  return {
    isElectron: Boolean(desktop?.isElectron),
    platform: desktop?.platform || 'web',
  }
})

const currentView = computed(() => {
  const meta = VIEW_META[route.path] || VIEW_META['/dashboard']
  return {
    title: t(meta.titleKey),
    description: t(meta.descriptionKey),
  }
})

const runtimeLabel = computed(() => t(runtime.value.isElectron ? 'common.runtime.desktop' : 'common.runtime.web'))
const platformLabel = computed(() => {
  const key = `common.platforms.${runtime.value.platform}`
  const label = t(key)
  return label === key ? runtime.value.platform : label
})
const shellClasses = computed(() => ({
  'is-electron': runtime.value.isElectron,
  'is-web': !runtime.value.isElectron,
}))
const versionLabel = computed(() => `v${appVersion}`)
const elementLocale = computed(() => (locale.value === 'zh-CN' ? zhCn : en))
const currentLocale = computed({
  get: () => locale.value,
  set: (value) => setAppLocale(value),
})
const localeOptions = computed(() =>
  SUPPORTED_LOCALES.map((value) => ({
    value,
    label: value === 'zh-CN' ? t('language.options.zhCN') : t('language.options.enUS'),
  })),
)
const currentLabel = computed(() => {
  const item = connections.value.find((entry) => entry.id === activeId.value)
  return item?.name || t('layout.noConnectionSelected')
})

async function loadConnections() {
  const data = await getConnections()
  connections.value = data.items || []
  if (data.activeId && !activeId.value) {
    activeId.value = data.activeId
    setActiveConnectionId(data.activeId)
  }
}

async function handleSwitch(id) {
  await activateConnection(id)
  activeId.value = id
  setActiveConnectionId(id)
  ElMessage.success(t('layout.header.switchSuccess'))
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
  <el-config-provider :locale="elementLocale">
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
            <span>{{ t('layout.nav.connections') }}</span>
          </el-menu-item>
          <el-menu-item index="/dashboard">
            <el-icon><DataAnalysis /></el-icon>
            <span>{{ t('layout.nav.dashboard') }}</span>
          </el-menu-item>
          <el-menu-item index="/jetstream">
            <el-icon><FolderOpened /></el-icon>
            <span>{{ t('layout.nav.jetstream') }}</span>
          </el-menu-item>
          <el-menu-item index="/kv">
            <el-icon><Key /></el-icon>
            <span>{{ t('layout.nav.kv') }}</span>
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
              <span>{{ t('layout.sidebar.profiles') }}</span>
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
              <h1>{{ t('layout.header.title') }}</h1>
              <p>{{ currentView.description }}</p>
            </div>
          </div>
          <div class="topbar-actions app-control">
            <div class="locale-panel">
              <span class="connection-panel-label">{{ t('language.label') }}</span>
              <el-select v-model="currentLocale" class="language-switch">
                <el-option v-for="item in localeOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </div>
            <div class="connection-panel">
              <span class="connection-panel-label">{{ t('layout.header.currentConnection') }}</span>
              <strong class="active-connection">{{ currentLabel }}</strong>
            </div>
            <el-select
              :model-value="activeId"
              :placeholder="t('layout.header.selectConnection')"
              class="connection-switch"
              @change="handleSwitch"
            >
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
  </el-config-provider>
</template>
