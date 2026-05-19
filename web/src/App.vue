<!-- web/src/App.vue -->
<template>
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
          <span>域名管理</span>
        </el-menu-item>
        <el-menu-item index="/friends">
          <el-icon><component :is="'Avatar'" /></el-icon>
          <span>好友</span>
        </el-menu-item>
        <el-menu-item index="/messages">
          <el-icon><component :is="'ChatDotRound'" /></el-icon>
          <template #title>
            <span>消息</span>
            <el-badge v-if="unreadCount > 0" :value="unreadCount" :max="99" :offset="[10, 0]" />
          </template>
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><component :is="'Tools'" /></el-icon>
          <span>系统设置</span>
        </el-menu-item>
        <el-menu-item index="/providers">
          <el-icon><component :is="'Connection'" /></el-icon>
          <span>DNS 提供商</span>
        </el-menu-item>
        <el-menu-item index="/profile">
          <el-icon><component :is="'User'" /></el-icon>
          <span>个人信息</span>
        </el-menu-item>
        <el-menu-item index="/permissions">
          <el-icon><component :is="'Key'" /></el-icon>
          <span>我的权限</span>
        </el-menu-item>
        <el-menu-item index="/my-logs">
          <el-icon><component :is="'Document'" /></el-icon>
          <span>操作日志</span>
        </el-menu-item>
        <el-menu-item index="/ram-tokens">
          <el-icon><component :is="'Tickets'" /></el-icon>
          <span>API Tokens</span>
        </el-menu-item>
        <el-menu-item v-if="auth.isAdmin" index="/admin">
          <el-icon><component :is="'UserFilled'" /></el-icon>
          <span>管理后台</span>
        </el-menu-item>
      </el-menu>
      <div class="sidebar-footer">
        <div class="user-info" @click="$router.push('/profile')">
          <el-avatar v-if="auth.avatar" :src="auth.avatar" :size="28" />
          <el-icon v-else><component :is="'User'" /></el-icon>
          <span>{{ auth.nickname }}</span>
        </div>
        <el-button text size="small" @click="handleLogout" class="logout-btn">退出登录</el-button>
      </div>
    </aside>

    <main class="main-content">
      <router-view />
    </main>
  </div>
  <div v-else>
    <router-view />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { getUnreadCount } from './api/message'

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

onMounted(() => {
  loadUnread()
  unreadTimer = setInterval(loadUnread, 15000)
})

onUnmounted(() => {
  if (unreadTimer) clearInterval(unreadTimer)
})

const activeMenu = computed(() => {
  if (route.path.startsWith('/admin')) return '/admin'
  if (route.path.startsWith('/settings')) return '/settings'
  if (route.path.startsWith('/providers')) return '/providers'
  if (route.path.startsWith('/profile')) return '/profile'
  if (route.path.startsWith('/my-logs')) return '/my-logs'
  if (route.path.startsWith('/permissions')) return '/permissions'
  if (route.path.startsWith('/messages')) return '/messages'
  if (route.path.startsWith('/friends')) return '/friends'
  return '/dashboard'
})

const handleLogout = () => {
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

/* Ensure el-dialog is vertically centered on all screen sizes */
.el-overlay:has(.el-dialog) {
  display: flex !important;
  justify-content: center;
  align-items: center;
}

@media (max-width: 768px) {
  .el-dialog {
    --el-dialog-width: 92vw !important;
    width: 92vw !important;
    margin: 16px auto;
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
  width: 100%;
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
  }

  .sidebar {
    position: fixed;
    left: -220px;
    top: 0;
    bottom: 0;
    z-index: 1003;
    transition: left 0.25s ease;
  }
  .sidebar.open {
    left: 0;
  }

  .main-content {
    padding: 60px 12px 12px;
  }
}
</style>
