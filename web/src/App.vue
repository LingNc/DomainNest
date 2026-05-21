<!-- web/src/App.vue -->
<template>
  <el-config-provider :locale="elLocale">
  <div v-if="auth.isAuthed" class="app-layout">
    <!-- 移动端顶栏 -->
    <div class="mobile-topbar">
      <el-button text @click="sidebarOpen = true" class="menu-btn">
        <el-icon size="22"><component :is="'Expand'" /></el-icon>
      </el-button>
      <span class="mobile-title">DomainNest</span>
    </div>

    <!-- 移动端遮罩 -->
    <div v-if="sidebarOpen" class="sidebar-overlay" @click="sidebarOpen = false" />

    <!-- 侧边栏 -->
    <aside class="sidebar" :class="{ open: sidebarOpen }">
      <div class="sidebar-logo">
        <h1>DomainNest</h1>
      </div>
      <el-menu
        :default-active="activeMenu"
        router
        background-color="#1d1e2c"
        text-color="#a3a6b4"
        active-text-color="#409eff"
        @select="sidebarOpen = false"
      >
        <el-menu-item index="/dashboard">
          <el-icon><component :is="'HomeFilled'" /></el-icon>
          <span>{{ $t('sidebar.domainManagement') }}</span>
        </el-menu-item>
        <el-menu-item index="/friends">
          <el-icon><component :is="'Avatar'" /></el-icon>
          <span>{{ $t('sidebar.friends') }}</span>
        </el-menu-item>
        <el-menu-item index="/messages">
          <el-icon><component :is="'ChatDotRound'" /></el-icon>
          <template #title>
            <span>{{ $t('sidebar.messages') }}</span>
            <el-badge v-if="unreadCount > 0" :value="unreadCount" :max="99" :offset="[10, 0]" />
          </template>
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><component :is="'Tools'" /></el-icon>
          <span>{{ $t('sidebar.settings') }}</span>
        </el-menu-item>
        <el-menu-item index="/providers">
          <el-icon><component :is="'Connection'" /></el-icon>
          <span>{{ $t('sidebar.providers') }}</span>
        </el-menu-item>
        <el-menu-item index="/profile">
          <el-icon><component :is="'User'" /></el-icon>
          <span>{{ $t('sidebar.profile') }}</span>
        </el-menu-item>
        <el-menu-item index="/my-logs">
          <el-icon><component :is="'Document'" /></el-icon>
          <span>{{ $t('sidebar.logs') }}</span>
        </el-menu-item>
        <el-menu-item index="/ram-tokens">
          <el-icon><component :is="'Tickets'" /></el-icon>
          <span>{{ $t('sidebar.apiTokens') }}</span>
        </el-menu-item>
        <el-menu-item v-if="auth.isAdmin" index="/admin">
          <el-icon><component :is="'UserFilled'" /></el-icon>
          <span>{{ $t('sidebar.admin') }}</span>
        </el-menu-item>
      </el-menu>
      <div class="sidebar-footer">
        <div class="user-info" @click="$router.push('/profile')">
          <el-avatar v-if="auth.avatar" :src="auth.avatar" :size="28" />
          <el-icon v-else><component :is="'User'" /></el-icon>
          <span>{{ auth.nickname }}</span>
        </div>
        <div class="footer-actions">
          <div class="lang-switch">
            <span :class="{ active: locale === 'zh-CN' }" @click="switchLang('zh-CN')">中</span>
            <span class="divider">/</span>
            <span :class="{ active: locale === 'en-US' }" @click="switchLang('en-US')">En</span>
          </div>
          <el-button text size="small" @click="handleLogout" class="logout-btn">{{ $t('common.logout') }}</el-button>
        </div>
      </div>
    </aside>

    <main class="main-content">
      <router-view />
    </main>
  </div>
  <div v-else style="min-height: 100vh">
    <router-view />
  </div>
  </el-config-provider>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import en from 'element-plus/dist/locale/en.mjs'
import { useAuthStore } from './stores/auth'
import { getUnreadCount } from './api/message'
import { setLang } from './i18n/utils'
import { useWebSocket, disconnect } from './composables/useWebSocket'

const { locale } = useI18n()
const elLocaleMap = { 'zh-CN': zhCn, 'en-US': en }
const elLocale = computed(() => elLocaleMap[locale.value] || zhCn)
const switchLang = (lang) => {
  locale.value = lang
  setLang(lang)
}

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const sidebarOpen = ref(false)
const unreadCount = ref(0)

let unreadTimer = null

const loadUnread = async () => {
  if (!auth.isAuthed) return
  try {
    const res = await getUnreadCount()
    unreadCount.value = res.data?.count || 0
  } catch { /* ignore */ }
}

const { connected, on: wsOn, off: wsOff } = useWebSocket()

const handleUnreadUpdate = (payload) => {
  unreadCount.value = payload.count ?? 0
}

onMounted(() => {
  loadUnread()
  unreadTimer = setInterval(loadUnread, 30000)
  wsOn('unread_update', handleUnreadUpdate)
})

onUnmounted(() => {
  if (unreadTimer) clearInterval(unreadTimer)
  wsOff('unread_update', handleUnreadUpdate)
})

const activeMenu = computed(() => {
  if (route.path.startsWith('/admin')) return '/admin'
  if (route.path.startsWith('/settings')) return '/settings'
  if (route.path.startsWith('/providers')) return '/providers'
  if (route.path.startsWith('/profile')) return '/profile'
  if (route.path.startsWith('/my-logs')) return '/my-logs'
  if (route.path.startsWith('/messages')) return '/messages'
  if (route.path.startsWith('/friends')) return '/friends'
  return '/dashboard'
})

const handleLogout = () => {
  disconnect()
  auth.clearAuth()
  router.push('/login')
}
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif;
  background: #f0f2f5;
  color: #303133;
}

/* Fix dialog form padding balance: left-align labels so visual
   distance from edge is symmetric with the input side */
.el-dialog .el-form-item__label {
  text-align: left;
  padding-left: 0 !important;
}

/* Ensure el-dialog is vertically centered on all screen sizes */
.el-overlay-dialog {
  display: flex !important;
}
.el-dialog {
  margin: auto !important;
}

@media (max-width: 768px) {
  .el-dialog {
    --el-dialog-width: 92vw !important;
    width: 92vw !important;
    margin: auto !important;
    max-height: calc(100vh - 32px);
  }
  .el-dialog__body {
    max-height: calc(100vh - 160px);
    overflow-y: auto;
  }
  .el-message-box {
    --el-messagebox-width: 90vw !important;
    width: 90vw !important;
  }
  .el-table {
    font-size: 12px;
  }
  .el-table .cell {
    padding: 0 6px;
    white-space: nowrap;
  }
  .el-form-item__label {
    font-size: 13px;
  }
  .el-card__body {
    padding: 12px;
  }
}
</style>

<style scoped>
.app-layout {
  display: flex;
  height: 100vh;
}

.sidebar {
  width: 220px;
  background: #1d1e2c;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.sidebar-logo {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-bottom: 1px solid rgba(255,255,255,0.08);
}
.sidebar-logo h1 {
  color: #fff;
  font-size: 18px;
  font-weight: 600;
  letter-spacing: 1px;
}

.sidebar .el-menu {
  border-right: none;
  flex: 1;
}

.sidebar-footer {
  padding: 16px;
  border-top: 1px solid rgba(255,255,255,0.08);
}
.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #a3a6b4;
  font-size: 14px;
  margin-bottom: 8px;
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  transition: background 0.2s;
}
.user-info:hover {
  background: rgba(255,255,255,0.08);
}
.logout-btn {
  color: #f56c6c !important;
}
.footer-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 0;
}
.lang-switch {
  display: flex;
  align-items: center;
  gap: 2px;
  font-size: 12px;
  color: #6c6e7e;
}
.lang-switch span {
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 3px;
  transition: all 0.2s;
}
.lang-switch span:hover {
  color: #a3a6b4;
}
.lang-switch span.active {
  color: #409eff;
  font-weight: 500;
}
.lang-switch .divider {
  color: #4a4b5a;
  cursor: default;
  padding: 0;
}
.lang-switch .divider:hover {
  color: #4a4b5a;
}

.main-content {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

/* 移动端顶栏 - 默认隐藏 */
.mobile-topbar {
  display: none;
}
.sidebar-overlay {
  display: none;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .mobile-topbar {
    display: flex;
    align-items: center;
    gap: 8px;
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    height: 50px;
    background: #1d1e2c;
    z-index: 1001;
    padding: 0 12px;
  }
  .mobile-title {
    color: #fff;
    font-size: 16px;
    font-weight: 600;
  }
  .menu-btn {
    color: #fff !important;
  }

  .sidebar-overlay {
    display: block;
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.5);
    z-index: 1002;
    transition: opacity 0.25s ease;
  }

  .sidebar {
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    z-index: 1003;
    transition: transform 0.25s ease;
    transform: translateX(-220px);
    will-change: transform;
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
    overscroll-behavior: contain;
  }
  .sidebar.open {
    transform: translateX(0);
  }

  .main-content {
    padding: 60px 12px 12px;
  }
}
</style>
