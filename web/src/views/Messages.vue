<template>
  <div>
    <div class="page-header">
      <h2>{{ $t('messages.title') }}</h2>
      <p class="subtitle">{{ $t('messages.subtitle') }}</p>
    </div>

    <el-tabs v-model="activeTab" @tab-change="handleTabChange">
      <!-- Notifications Tab -->
      <el-tab-pane name="notifications">
        <template #label>
          <el-badge :value="notifUnread" :hidden="notifUnread === 0" :max="99">
            <span>{{ $t('messages.notifications') }}</span>
          </el-badge>
        </template>
        <div class="toolbar" v-if="notifications.length > 0">
          <el-button size="small" @click="handleMarkAllRead">{{ $t('messages.markAllRead') }}</el-button>
        </div>
        <el-card>
          <div v-loading="loadingNotifs">
            <div
              v-for="notif in notifications"
              :key="notif.id"
              class="notif-item"
              :class="{ unread: !notif.read_at }"
              @click="handleNotifClick(notif)"
            >
              <div class="notif-icon">
                <el-icon :size="20" color="#409eff"><Bell /></el-icon>
              </div>
              <div class="notif-body">
                <div class="notif-top">
                  <span class="notif-title">{{ notif.title || $t('messages.systemNotification') }}</span>
                  <span class="notif-time">{{ formatTime(notif.created_at) }}</span>
                </div>
                <div class="notif-content">{{ notif.content }}</div>
              </div>
              <div v-if="!notif.read_at" class="notif-dot"></div>
            </div>
            <el-empty v-if="!loadingNotifs && notifications.length === 0" :description="$t('messages.noNotifications')" />
          </div>
          <el-pagination
            v-if="notifTotal > 20"
            v-model:current-page="notifPage"
            :page-size="20"
            :total="notifTotal"
            layout="total, prev, pager, next"
            @current-change="loadNotifications"
            style="margin-top: 12px"
          />
        </el-card>
      </el-tab-pane>

      <!-- System Notifications Tab -->
      <el-tab-pane name="systemNotifications">
        <template #label>
          <el-badge :value="notifStore.unreadCount" :hidden="notifStore.unreadCount === 0" :max="99">
            <span>{{ $t('notif.systemNotifications') }}</span>
          </el-badge>
        </template>
        <div class="toolbar" style="display:flex;align-items:center;gap:8px;flex-wrap:wrap">
          <el-select v-model="sysNotifCategory" :placeholder="$t('notif.allCategories')" clearable size="small" style="width:140px" @change="loadSystemNotifications(1)">
            <el-option :label="$t('notif.allCategories')" value="" />
            <el-option :label="$t('notif.catSystem')" value="system" />
            <el-option :label="$t('notif.catWarning')" value="warning" />
            <el-option :label="$t('notif.catPromotion')" value="promotion" />
          </el-select>
          <el-button v-if="notifStore.unreadCount > 0" size="small" @click="handleSysMarkAllRead">{{ $t('notif.markAllRead') }}</el-button>
        </div>
        <el-card>
          <div v-loading="notifStore.loading">
            <div
              v-for="n in notifStore.notifications"
              :key="n.id"
              class="notif-item"
              :class="[!n.read_at ? 'unread' : '', 'priority-' + (n.priority || 'normal')]"
              @click="handleSysNotifClick(n)"
            >
              <div class="notif-icon" :class="'cat-' + (n.category || 'system')">
                <el-icon :size="20"><component :is="sysCategoryIcon(n.category)" /></el-icon>
              </div>
              <div class="notif-body">
                <div class="notif-top">
                  <span class="notif-title">{{ n.title || $t('notif.systemNotification') }}</span>
                  <el-tag v-if="n.priority === 'warning'" type="warning" size="small">{{ $t('notif.priorityWarning') }}</el-tag>
                  <el-tag v-else-if="n.priority === 'error'" type="danger" size="small">{{ $t('notif.priorityError') }}</el-tag>
                </div>
                <div class="notif-content">{{ n.content }}</div>
                <div class="notif-time">{{ formatTime(n.created_at) }}</div>
              </div>
              <div class="notif-actions-col">
                <el-button
                  v-if="!n.read_at"
                  text
                  type="primary"
                  size="small"
                  @click.stop="handleSysMarkRead(n)"
                >
                  {{ $t('notif.markRead') }}
                </el-button>
                <el-button
                  text
                  type="danger"
                  size="small"
                  @click.stop="handleSysDelete(n)"
                >
                  {{ $t('common.delete') }}
                </el-button>
              </div>
              <div v-if="!n.read_at" class="notif-dot"></div>
            </div>
            <el-empty v-if="!notifStore.loading && notifStore.notifications.length === 0" :description="$t('notif.empty')" />
          </div>
          <el-pagination
            v-if="notifStore.total > 20"
            v-model:current-page="sysNotifPage"
            :page-size="20"
            :total="notifStore.total"
            layout="total, prev, pager, next"
            @current-change="loadSystemNotifications"
            style="margin-top: 12px"
          />
        </el-card>
      </el-tab-pane>

      <!-- Conversations Tab (private messages) -->
      <el-tab-pane :label="$t('messages.privateMessages')" name="conversations">
        <!-- New Message Dialog -->
        <el-dialog v-model="newMsgVisible" :title="$t('messages.newMessage')" width="480px" destroy-on-close>
          <el-form @submit.prevent="handleSearchUser">
            <el-form-item :label="$t('messages.sendTo')">
              <el-input v-model="searchKeyword" :placeholder="$t('messages.searchFriendPlaceholder')" clearable @input="handleSearchDebounced">
                <template #append>
                  <el-button @click="handleSearchUser">{{ $t('common.search') }}</el-button>
                </template>
              </el-input>
            </el-form-item>
          </el-form>
          <div v-if="searchResults.length > 0" class="search-results">
            <div v-for="u in searchResults" :key="u.id" class="search-item" @click="openChat(u.id)">
              <div style="display:flex;align-items:center;gap:10px">
                <el-avatar v-if="u.avatar" :src="u.avatar" :size="32" />
                <el-avatar v-else :size="32">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
                <div>
                  <div class="username">{{ u.nickname || u.username }}</div>
                  <div class="user-handle">@{{ u.username }}</div>
                </div>
              </div>
            </div>
          </div>
          <el-empty v-else-if="searched && searchResults.length === 0" :description="$t('common.userNotFound')" />
        </el-dialog>

        <div class="toolbar">
          <el-button type="primary" size="small" @click="newMsgVisible = true">{{ $t('messages.newMessage') }}</el-button>
        </div>

        <el-card>
          <div v-loading="loading">
            <div
              v-for="conv in conversations"
              :key="conv.user.id"
              class="conv-item"
              @click="openChat(conv.user.id)"
            >
              <div class="conv-left">
                <el-badge :value="conv.unread_count" :hidden="conv.unread_count === 0" :max="99">
                  <el-avatar v-if="conv.user.avatar" :src="conv.user.avatar" :size="44" />
                  <el-avatar v-else :size="44">{{ (conv.user.username || '?')[0]?.toUpperCase() }}</el-avatar>
                </el-badge>
              </div>
              <div class="conv-body">
                <div class="conv-top">
                  <span class="conv-name">{{ conv.user.nickname || conv.user.username }}</span>
                  <span class="conv-time">{{ formatTime(conv.last_msg_time) }}</span>
                </div>
                <div class="conv-preview">{{ conv.last_message }}</div>
              </div>
            </div>
            <el-empty v-if="!loading && conversations.length === 0" :description="$t('messages.noMessages')">
              <el-button type="primary" @click="newMsgVisible = true">{{ $t('messages.sendFirstMessage') }}</el-button>
            </el-empty>
          </div>
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Bell, WarningFilled, InfoFilled, Promotion } from '@element-plus/icons-vue'
import { getConversations } from '../api/message'
import { getNotifications, getNotificationUnreadCount, markNotificationAsRead, markAllNotificationsAsRead } from '../api/message'
import { searchAllUsers } from '../api/auth'
import { useNotificationStore } from '../stores/notification'
import { useWebSocket } from '../composables/useWebSocket'
import { useError } from '../composables/useError'

const router = useRouter()
const route = useRoute()
const { t, locale } = useI18n()
const { showError } = useError()
const notifStore = useNotificationStore()

const activeTab = ref('notifications')

// --- System Notifications (new notification API) ---
const sysNotifCategory = ref('')
const sysNotifPage = ref(1)

const loadSystemNotifications = async (page = 1) => {
  sysNotifPage.value = page
  const params = { page, page_size: 20 }
  if (sysNotifCategory.value) params.category = sysNotifCategory.value
  await notifStore.fetchNotifications(params)
}

const sysCategoryIcon = (cat) => {
  const map = { system: 'Bell', warning: 'WarningFilled', promotion: 'Promotion' }
  return map[cat] || 'InfoFilled'
}

const handleSysNotifClick = async (n) => {
  if (!n.read_at) {
    await notifStore.markNotificationAsRead(n.id)
  }
}

const handleSysMarkRead = async (n) => {
  await notifStore.markNotificationAsRead(n.id)
}

const handleSysMarkAllRead = async () => {
  await notifStore.markAllNotificationsAsRead()
}

const handleSysDelete = async (n) => {
  await notifStore.removeNotification(n.id)
}

// --- Conversations ---
const conversations = ref([])
const loading = ref(false)
const newMsgVisible = ref(false)
const searchKeyword = ref('')
const searchResults = ref([])
const searched = ref(false)
let searchTimer = null

const loadConversations = async () => {
  loading.value = true
  try {
    const res = await getConversations()
    conversations.value = res.data || []
  } finally {
    loading.value = false
  }
}

// --- Notifications ---
const notifications = ref([])
const loadingNotifs = ref(false)
const notifUnread = ref(0)
const notifPage = ref(1)
const notifTotal = ref(0)

const loadNotifications = async (page = 1) => {
  loadingNotifs.value = true
  notifPage.value = page
  try {
    const res = await getNotifications({ page, page_size: 20 })
    notifications.value = res.data.items || []
    notifTotal.value = res.data.total || 0
  } finally {
    loadingNotifs.value = false
  }
}

const loadNotifUnreadCount = async () => {
  try {
    const res = await getNotificationUnreadCount()
    notifUnread.value = res.data?.count || 0
  } catch { /* ignore */ }
}

const handleNotifClick = async (notif) => {
  if (!notif.read_at) {
    try {
      await markNotificationAsRead(notif.id)
      notif.read_at = new Date().toISOString()
      notifUnread.value = Math.max(0, notifUnread.value - 1)
    } catch { /* ignore */ }
  }
}

const handleMarkAllRead = async () => {
  try {
    await markAllNotificationsAsRead()
    notifications.value.forEach(n => { n.read_at = new Date().toISOString() })
    notifUnread.value = 0
  } catch { /* ignore */ }
}

// --- Shared ---
const formatTime = (t) => {
  if (!t) return ''
  const d = new Date(t)
  const now = new Date()
  const isToday = d.toDateString() === now.toDateString()
  if (isToday) {
    return d.toLocaleTimeString(locale.value, { hour: '2-digit', minute: '2-digit' })
  }
  return d.toLocaleDateString(locale.value, { month: '2-digit', day: '2-digit' }) + ' ' +
    d.toLocaleTimeString(locale.value, { hour: '2-digit', minute: '2-digit' })
}

const handleSearchDebounced = () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(handleSearchUser, 400)
}

const handleSearchUser = async () => {
  if (searchKeyword.value.length < 2) {
    searchResults.value = []
    searched.value = false
    return
  }
  try {
    const res = await searchAllUsers(searchKeyword.value)
    searchResults.value = res.data || []
    searched.value = true
  } catch {
    searchResults.value = []
  }
}

const openChat = (userId) => {
  newMsgVisible.value = false
  router.push(`/messages/${userId}`)
}

const handleTabChange = (tab) => {
  if (tab === 'conversations') {
    loadConversations()
  } else if (tab === 'systemNotifications') {
    loadSystemNotifications()
    notifStore.fetchUnreadCount()
  }
}

const { on: wsOn, off: wsOff } = useWebSocket()

const handleNewNotification = (payload) => {
  notifications.value.unshift(payload)
  notifUnread.value++
}

const handleUnreadUpdate = (payload) => {
  notifUnread.value = payload.count ?? 0
}

onMounted(() => {
  // Support deep-linking via ?tab=systemNotifications
  if (route.query.tab === 'systemNotifications') {
    activeTab.value = 'systemNotifications'
    loadSystemNotifications()
    notifStore.fetchUnreadCount()
  } else {
    loadNotifications()
    loadNotifUnreadCount()
  }
  wsOn('new_notification', handleNewNotification)
  wsOn('unread_update', handleUnreadUpdate)
})

onUnmounted(() => {
  wsOff('new_notification', handleNewNotification)
  wsOff('unread_update', handleUnreadUpdate)
})
</script>

<style scoped>
.page-header {
  margin-bottom: 20px;
}
.page-header h2 {
  font-size: 22px;
  font-weight: 600;
  margin-bottom: 4px;
}
.subtitle {
  color: #909399;
  font-size: 14px;
}
.toolbar {
  margin-bottom: 16px;
}

/* Conversations */
.conv-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 0;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
  transition: background 0.15s;
}
.conv-item:hover {
  background: #f5f7fa;
  margin: 0 -20px;
  padding-left: 20px;
  padding-right: 20px;
}
.conv-item:last-child {
  border-bottom: none;
}
.conv-body {
  flex: 1;
  min-width: 0;
}
.conv-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}
.conv-name {
  font-weight: 500;
  font-size: 15px;
}
.conv-time {
  font-size: 12px;
  color: #909399;
}
.conv-preview {
  font-size: 13px;
  color: #909399;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Search */
.search-results {
  max-height: 320px;
  overflow-y: auto;
}
.search-item {
  display: flex;
  align-items: center;
  padding: 10px 4px;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
  border-radius: 6px;
  transition: background 0.15s;
}
.search-item:hover {
  background: #f5f7fa;
}
.search-item:last-child {
  border-bottom: none;
}
.username {
  font-weight: 500;
}
.user-handle {
  font-size: 12px;
  color: #909399;
}

/* Notifications */
.notif-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 0;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
  transition: background 0.15s;
  position: relative;
}
.notif-item:hover {
  background: #f5f7fa;
  margin: 0 -20px;
  padding-left: 20px;
  padding-right: 20px;
}
.notif-item:last-child {
  border-bottom: none;
}
.notif-item.unread {
  background: #f0f7ff;
}
.notif-icon {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #ecf5ff;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #409eff;
}
.notif-icon.cat-warning {
  background: #fdf6ec;
  color: #e6a23c;
}
.notif-icon.cat-promotion {
  background: #f0f9eb;
  color: #67c23a;
}
.notif-item.priority-warning {
  border-left: 3px solid #e6a23c;
}
.notif-item.priority-error {
  border-left: 3px solid #f56c6c;
}
.notif-time {
  font-size: 12px;
  color: #c0c4cc;
  margin-top: 4px;
}
.notif-actions-col {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-left: 8px;
}
.notif-body {
  flex: 1;
  min-width: 0;
}
.notif-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}
.notif-title {
  font-weight: 500;
  font-size: 14px;
}
.notif-content {
  font-size: 13px;
  color: #606266;
  line-height: 1.5;
  word-break: break-word;
}
.notif-dot {
  flex-shrink: 0;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #409eff;
  margin-top: 6px;
}
</style>
