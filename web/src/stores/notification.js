import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  getNotifications,
  getUnreadCount,
  markAsRead as apiMarkAsRead,
  markAllAsRead as apiMarkAllAsRead,
  deleteNotification as apiDeleteNotification,
} from '../api/notification'

export const useNotificationStore = defineStore('notification', () => {
  const unreadCount = ref(0)
  const notifications = ref([])
  const total = ref(0)
  const loading = ref(false)

  async function fetchUnreadCount() {
    try {
      const res = await getUnreadCount()
      unreadCount.value = res.data?.count || 0
    } catch { /* ignore */ }
  }

  async function fetchNotifications(params = {}) {
    loading.value = true
    try {
      const res = await getNotifications(params)
      notifications.value = res.data?.items || []
      total.value = res.data?.total || 0
    } finally {
      loading.value = false
    }
  }

  async function markNotificationAsRead(id) {
    try {
      await apiMarkAsRead(id)
      const n = notifications.value.find(x => x.id === id)
      if (n && !n.read_at) {
        n.read_at = new Date().toISOString()
        unreadCount.value = Math.max(0, unreadCount.value - 1)
      }
    } catch { /* ignore */ }
  }

  async function markAllNotificationsAsRead() {
    try {
      await apiMarkAllAsRead()
      notifications.value.forEach(n => { n.read_at = new Date().toISOString() })
      unreadCount.value = 0
    } catch { /* ignore */ }
  }

  async function removeNotification(id) {
    try {
      await apiDeleteNotification(id)
      const idx = notifications.value.findIndex(x => x.id === id)
      if (idx !== -1) {
        const wasUnread = !notifications.value[idx].read_at
        notifications.value.splice(idx, 1)
        total.value = Math.max(0, total.value - 1)
        if (wasUnread) unreadCount.value = Math.max(0, unreadCount.value - 1)
      }
    } catch { /* ignore */ }
  }

  function handleWsNotification(payload) {
    unreadCount.value++
    // If the list is loaded, prepend the new notification
    if (notifications.value.length > 0) {
      notifications.value.unshift(payload)
    }
  }

  function handleWsUnreadUpdate(payload) {
    unreadCount.value = payload.count ?? 0
  }

  return {
    unreadCount,
    notifications,
    total,
    loading,
    fetchUnreadCount,
    fetchNotifications,
    markNotificationAsRead,
    markAllNotificationsAsRead,
    removeNotification,
    handleWsNotification,
    handleWsUnreadUpdate,
  }
})
