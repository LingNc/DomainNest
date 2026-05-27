<template>
  <el-popover
    placement="bottom-end"
    :width="360"
    trigger="click"
    :show-after="0"
    @before-enter="onOpen"
  >
    <template #reference>
      <el-badge :value="store.unreadCount" :hidden="store.unreadCount === 0" :max="99" class="bell-badge">
        <el-icon :size="20" class="bell-icon"><Bell /></el-icon>
      </el-badge>
    </template>

    <div class="notif-popover">
      <div class="notif-popover-header">
        <span class="notif-popover-title">{{ $t('notif.title') }}</span>
        <el-button
          v-if="store.unreadCount > 0"
          text
          type="primary"
          size="small"
          @click="handleMarkAllRead"
        >
          {{ $t('notif.markAllRead') }}
        </el-button>
      </div>

      <div v-loading="loading" class="notif-popover-list">
        <div
          v-for="n in latest"
          :key="n.id"
          class="notif-popover-item"
          :class="{ unread: !n.read_at }"
          @click="handleClick(n)"
        >
          <div class="notif-popover-icon" :class="'cat-' + (n.category || 'system')">
            <el-icon :size="16"><component :is="categoryIcon(n.category)" /></el-icon>
          </div>
          <div class="notif-popover-body">
            <div class="notif-popover-item-title">{{ n.title || $t('notif.systemNotification') }}</div>
            <div class="notif-popover-item-content">{{ n.content }}</div>
            <div class="notif-popover-item-time">{{ formatTimeAgo(n.created_at) }}</div>
          </div>
          <div v-if="!n.read_at" class="notif-popover-dot"></div>
        </div>
        <el-empty v-if="!loading && latest.length === 0" :description="$t('notif.empty')" :image-size="60" />
      </div>

      <div class="notif-popover-footer" @click="goToAll">
        <el-icon><component :is="'View'" /></el-icon>
        <span>{{ $t('notif.viewAll') }}</span>
      </div>
    </div>
  </el-popover>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Bell, View, BellFilled, WarningFilled, InfoFilled, Promotion } from '@element-plus/icons-vue'
import { useNotificationStore } from '../stores/notification'

const router = useRouter()
const { t } = useI18n()
const store = useNotificationStore()
const loading = ref(false)

onMounted(() => store.fetchUnreadCount())

const latest = computed(() => store.notifications.slice(0, 8))

const categoryIcon = (cat) => {
  const map = {
    system: 'Bell',
    warning: 'WarningFilled',
    promotion: 'Promotion',
  }
  return map[cat] || 'InfoFilled'
}

const formatTimeAgo = (time) => {
  if (!time) return ''
  const d = new Date(time)
  const now = new Date()
  const diff = Math.floor((now - d) / 1000)
  if (diff < 60) return t('notif.justNow')
  if (diff < 3600) return t('notif.minutesAgo', { n: Math.floor(diff / 60) })
  if (diff < 86400) return t('notif.hoursAgo', { n: Math.floor(diff / 3600) })
  return t('notif.daysAgo', { n: Math.floor(diff / 86400) })
}

const onOpen = async () => {
  loading.value = true
  try {
    await store.fetchNotifications({ page: 1, page_size: 8 })
  } finally {
    loading.value = false
  }
}

const handleClick = async (n) => {
  if (!n.read_at) {
    await store.markNotificationAsRead(n.id)
  }
}

const handleMarkAllRead = async () => {
  await store.markAllNotificationsAsRead()
}

const goToAll = () => {
  router.push('/messages?tab=systemNotifications')
}
</script>

<style scoped>
.bell-badge {
  cursor: pointer;
  line-height: 1;
}
.bell-icon {
  color: #606266;
  transition: color 0.2s;
}
.bell-icon:hover {
  color: #409eff;
}

.notif-popover {
  margin: -12px;
}
.notif-popover-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid #f0f0f0;
}
.notif-popover-title {
  font-weight: 600;
  font-size: 14px;
}

.notif-popover-list {
  max-height: 360px;
  overflow-y: auto;
  min-height: 80px;
}

.notif-popover-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 12px;
  cursor: pointer;
  transition: background 0.15s;
  position: relative;
}
.notif-popover-item:hover {
  background: #f5f7fa;
}
.notif-popover-item.unread {
  background: #f0f7ff;
}

.notif-popover-icon {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #ecf5ff;
  color: #409eff;
}
.notif-popover-icon.cat-warning {
  background: #fdf6ec;
  color: #e6a23c;
}
.notif-popover-icon.cat-promotion {
  background: #f0f9eb;
  color: #67c23a;
}

.notif-popover-body {
  flex: 1;
  min-width: 0;
}
.notif-popover-item-title {
  font-size: 13px;
  font-weight: 500;
  margin-bottom: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.notif-popover-item-content {
  font-size: 12px;
  color: #909399;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.notif-popover-item-time {
  font-size: 11px;
  color: #c0c4cc;
  margin-top: 2px;
}

.notif-popover-dot {
  flex-shrink: 0;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #409eff;
  margin-top: 6px;
}

.notif-popover-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 8px;
  border-top: 1px solid #f0f0f0;
  font-size: 13px;
  color: #409eff;
  cursor: pointer;
  transition: background 0.15s;
}
.notif-popover-footer:hover {
  background: #f5f7fa;
}
</style>
