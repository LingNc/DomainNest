<template>
  <div class="chat-page">
    <div class="chat-header">
      <el-button text @click="$router.push('/messages')">
        <el-icon><component :is="'ArrowLeft'" /></el-icon>
      </el-button>
      <div class="chat-user">
        <el-avatar v-if="otherUser.avatar" :src="otherUser.avatar" :size="36" />
        <el-avatar v-else :size="36">{{ (otherUser.username || '?')[0]?.toUpperCase() }}</el-avatar>
        <div>
          <div class="chat-name">{{ otherUser.nickname || otherUser.username }}</div>
          <div class="chat-handle">@{{ otherUser.username }}</div>
        </div>
      </div>
    </div>

    <div class="chat-messages" ref="messagesContainer">
      <div class="chat-scroll" ref="scrollContent">
        <div v-if="hasMore" class="load-more">
          <el-button text size="small" @click="loadMore" :loading="loadingMore">{{ $t('chat.loadMore') }}</el-button>
        </div>
        <div
          v-for="msg in messages"
          :key="msg.id"
          :class="['msg-row', msg.sender_id === myUserId ? 'msg-sent' : 'msg-received']"
        >
          <el-avatar
            v-if="msg.sender_id === myUserId"
            :size="32"
            style="visibility:hidden"
          />
          <div class="msg-bubble">
            <div class="msg-content">{{ msg.content }}</div>
            <div class="msg-time">{{ formatTime(msg.created_at) }}</div>
          </div>
          <el-avatar
            v-if="msg.sender_id !== myUserId"
            :size="32"
            style="visibility:hidden"
          />
        </div>
        <el-empty v-if="!loading && messages.length === 0" :description="$t('chat.emptyMessage')" />
      </div>
    </div>

    <div class="chat-input">
      <el-input
        v-model="newMessage"
        :placeholder="$t('chat.inputPlaceholder')"
        :rows="1"
        resize="none"
        type="textarea"
        @keydown.enter.exact.prevent="handleSend"
        :autosize="{ minRows: 1, maxRows: 4 }"
      />
      <el-button type="primary" @click="handleSend" :disabled="!newMessage.trim()" :loading="sending">
        {{ $t('chat.send') }}
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, nextTick, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { getMessages, sendMessage, markAsRead } from '../api/message'
import { useAuthStore } from '../stores/auth'

const { t } = useI18n()

const route = useRoute()
const auth = useAuthStore()
const myUserId = computed(() => auth.user?.id)

const otherId = computed(() => Number(route.params.id))
const otherUser = ref({})

const messages = ref([])
const loading = ref(false)
const loadingMore = ref(false)
const sending = ref(false)
const newMessage = ref('')
const page = ref(1)
const total = ref(0)
const hasMore = ref(false)

const messagesContainer = ref(null)
const scrollContent = ref(null)

let pollTimer = null

const formatTime = (t) => {
  if (!t) return ''
  const d = new Date(t)
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

const loadMessages = async (reset = true) => {
  if (reset) {
    page.value = 1
    loading.value = true
  }
  try {
    const res = await getMessages(otherId.value, { page: page.value, page_size: 50 })
    const items = res.data.items || []
    total.value = res.data.total
    hasMore.value = items.length === 50 && (page.value * 50) < total.value

    if (reset) {
      messages.value = items
    } else {
      messages.value = [...items, ...messages.value]
    }

    // Get other user info from first message
    if (items.length > 0) {
      const msg = items.find(m => m.sender_id !== myUserId.value) || items[0]
      if (msg.sender && msg.sender.id === otherId.value) {
        otherUser.value = msg.sender
      } else if (msg.receiver && msg.receiver.id === otherId.value) {
        otherUser.value = msg.receiver
      }
    }

    if (reset) {
      await nextTick()
      scrollToBottom()
    }
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

const loadMore = async () => {
  loadingMore.value = true
  page.value++
  await loadMessages(false)
}

const scrollToBottom = () => {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

const handleSend = async () => {
  const content = newMessage.value.trim()
  if (!content) return

  sending.value = true
  try {
    const res = await sendMessage({ receiver_id: otherId.value, content })
    messages.value.push(res.data)
    newMessage.value = ''
    await nextTick()
    scrollToBottom()
  } finally {
    sending.value = false
  }
}

const doMarkAsRead = async () => {
  try {
    await markAsRead(otherId.value)
  } catch { /* ignore */ }
}

const pollNew = async () => {
  if (messages.value.length === 0) return
  const lastId = messages.value[messages.value.length - 1].id
  try {
    const res = await getMessages(otherId.value, { page: 1, page_size: 1 })
    const items = res.data.items || []
    if (items.length > 0 && items[items.length - 1].id > lastId) {
      // Reload from last known
      await loadMessages(true)
      doMarkAsRead()
    }
  } catch { /* ignore */ }
}

onMounted(async () => {
  await loadMessages()
  doMarkAsRead()
  // Poll every 3 seconds for new messages
  pollTimer = setInterval(pollNew, 3000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style scoped>
.chat-page {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 48px);
  background: #f5f5f5;
  margin: -24px;
}

.chat-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  flex-shrink: 0;
}

.chat-user {
  display: flex;
  align-items: center;
  gap: 10px;
}

.chat-name {
  font-weight: 600;
  font-size: 15px;
}

.chat-handle {
  font-size: 12px;
  color: #909399;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.chat-scroll {
  display: flex;
  flex-direction: column;
  min-height: 100%;
  justify-content: flex-end;
}

.load-more {
  text-align: center;
  padding: 8px 0;
}

.msg-row {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  align-items: flex-end;
}

.msg-sent {
  flex-direction: row-reverse;
}

.msg-bubble {
  max-width: 65%;
  padding: 10px 14px;
  border-radius: 12px;
  word-wrap: break-word;
  white-space: pre-wrap;
}

.msg-sent .msg-bubble {
  background: #409eff;
  color: #fff;
  border-bottom-right-radius: 4px;
}

.msg-received .msg-bubble {
  background: #fff;
  color: #303133;
  border-bottom-left-radius: 4px;
  box-shadow: 0 1px 2px rgba(0,0,0,0.06);
}

.msg-content {
  font-size: 14px;
  line-height: 1.5;
}

.msg-time {
  font-size: 11px;
  opacity: 0.7;
  margin-top: 4px;
  text-align: right;
}

.msg-sent .msg-time {
  color: rgba(255,255,255,0.8);
}

.msg-received .msg-time {
  color: #909399;
}

.chat-input {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  background: #fff;
  border-top: 1px solid #e4e7ed;
  flex-shrink: 0;
  align-items: flex-end;
}

.chat-input .el-textarea {
  flex: 1;
}

@media (max-width: 768px) {
  .chat-page {
    height: calc(100vh - 60px);
    margin: -60px -12px -12px;
  }
  .msg-bubble {
    max-width: 80%;
  }
}
</style>
