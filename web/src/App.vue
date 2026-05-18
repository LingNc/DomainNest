<!-- web/src/App.vue -->
<template>
  <div v-if="isAuthed" class="app-layout">
    <aside class="sidebar">
      <div class="sidebar-logo">
        <h1>DomainNest</h1>
      </div>
      <el-menu
        :default-active="activeMenu"
        router
        background-color="#1d1e2c"
        text-color="#a3a6b4"
        active-text-color="#409eff"
      >
        <el-menu-item index="/dashboard">
          <el-icon><component :is="'HomeFilled'" /></el-icon>
          <span>域名管理</span>
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><component :is="'Setting'" /></el-icon>
          <span>系统设置</span>
        </el-menu-item>
        <el-menu-item v-if="isAdmin" index="/admin">
          <el-icon><component :is="'UserFilled'" /></el-icon>
          <span>管理后台</span>
        </el-menu-item>
      </el-menu>
      <div class="sidebar-footer">
        <div class="user-info">
          <el-icon><component :is="'User'" /></el-icon>
          <span>{{ username }}</span>
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
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()

const isAuthed = computed(() => !!localStorage.getItem('token'))
const isAdmin = computed(() => {
  try {
    return JSON.parse(localStorage.getItem('user'))?.role === 'admin'
  } catch { return false }
})
const username = computed(() => {
  try {
    return JSON.parse(localStorage.getItem('user'))?.username || ''
  } catch { return '' }
})
const activeMenu = computed(() => {
  if (route.path.startsWith('/admin')) return '/admin'
  if (route.path.startsWith('/settings')) return '/settings'
  return '/dashboard'
})

const handleLogout = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('user')
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
</style>
