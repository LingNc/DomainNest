<template>
  <div>
    <div class="page-header">
      <h2>消息</h2>
      <p class="subtitle">与好友的聊天记录</p>
    </div>

    <!-- New Message Dialog -->
    <el-dialog v-model="newMsgVisible" title="新消息" width="480px">
      <el-form @submit.prevent="handleSearchUser">
        <el-form-item label="发送给">
          <el-input v-model="searchKeyword" placeholder="搜索好友用户名" clearable @input="handleSearchDebounced">
            <template #append>
              <el-button @click="handleSearchUser">搜索</el-button>
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
      <el-empty v-else-if="searched && searchResults.length === 0" description="未找到用户" />
    </el-dialog>

    <div class="toolbar">
      <el-button type="primary" @click="newMsgVisible = true">新消息</el-button>
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
        <el-empty v-if="!loading && conversations.length === 0" description="暂无消息">
          <el-button type="primary" @click="newMsgVisible = true">发送第一条消息</el-button>
        </el-empty>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getConversations } from '../api/message'
import { searchUsers } from '../api/friend'

const router = useRouter()
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

const formatTime = (t) => {
  if (!t) return ''
  const d = new Date(t)
  const now = new Date()
  const isToday = d.toDateString() === now.toDateString()
  if (isToday) {
    return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  }
  return d.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' }) + ' ' +
    d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
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
    const res = await searchUsers(searchKeyword.value)
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

onMounted(loadConversations)
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
</style>
