<template>
  <!-- This component has no visual output; it only listens for WS events and shows toasts -->
</template>

<script setup>
import { onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElNotification } from 'element-plus'
import { Bell, WarningFilled, InfoFilled } from '@element-plus/icons-vue'
import { useWebSocket } from '../composables/useWebSocket'
import { useNotificationStore } from '../stores/notification'

const router = useRouter()
const { t } = useI18n()
const store = useNotificationStore()
const { on: wsOn, off: wsOff } = useWebSocket()

const categoryIcon = (cat) => {
  const map = {
    system: Bell,
    warning: WarningFilled,
  }
  return map[cat] || InfoFilled
}

const handleNewNotification = (payload) => {
  store.handleWsNotification(payload)

  ElNotification({
    title: payload.title || t('notif.systemNotification'),
    message: payload.content || '',
    type: payload.category === 'warning' ? 'warning' : 'info',
    icon: categoryIcon(payload.category),
    duration: 5000,
    onClick: () => {
      if (!payload.read_at) {
        store.markNotificationAsRead(payload.id)
      }
      router.push('/messages?tab=systemNotifications')
    },
  })
}

const handleUnreadUpdate = (payload) => {
  store.handleWsUnreadUpdate(payload)
}

onMounted(() => {
  wsOn('new_notification', handleNewNotification)
  wsOn('unread_update', handleUnreadUpdate)
  // Fetch initial unread count
  store.fetchUnreadCount()
})

onUnmounted(() => {
  wsOff('new_notification', handleNewNotification)
  wsOff('unread_update', handleUnreadUpdate)
})
</script>
