<template>
  <div>
    <div class="page-header">
      <h2>好友管理</h2>
      <p class="subtitle">管理你的好友和好友请求</p>
    </div>

    <!-- Add Friend Dialog -->
    <el-dialog v-model="addDialogVisible" title="添加好友" width="480px">
      <el-form @submit.prevent="handleSearch">
        <el-form-item>
          <el-input v-model="searchKeyword" placeholder="输入用户名或昵称搜索" clearable @input="handleSearchDebounced">
            <template #append>
              <el-button @click="handleSearch">搜索</el-button>
            </template>
          </el-input>
        </el-form-item>
      </el-form>
      <div v-if="searchResults.length > 0" class="search-results">
        <div v-for="u in searchResults" :key="u.id" class="search-item">
          <div class="user-brief">
            <el-avatar v-if="u.avatar" :src="u.avatar" :size="32" />
            <el-avatar v-else :size="32">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
            <div>
              <div class="username">{{ u.nickname || u.username }}</div>
              <div class="user-handle">@{{ u.username }}</div>
            </div>
          </div>
          <el-button type="primary" size="small" @click="handleSendRequest(u)" :loading="sendingTo === u.id">
            添加
          </el-button>
        </div>
      </div>
      <el-empty v-else-if="searched && searchResults.length === 0" description="未找到用户" />
    </el-dialog>

    <el-tabs v-model="activeTab">
      <!-- Friends List -->
      <el-tab-pane label="好友列表" name="friends">
        <div class="toolbar">
          <el-button type="primary" @click="addDialogVisible = true">添加好友</el-button>
        </div>
        <el-card>
          <el-table :data="friends" stripe v-loading="loading" style="width:100%">
            <el-table-column label="好友" min-width="200">
              <template #default="{ row }">
                <div style="display:flex;align-items:center;gap:8px">
                  <el-avatar v-if="row.friend?.avatar" :src="row.friend.avatar" :size="32" />
                  <el-avatar v-else :size="32">{{ (row.friend?.username || '?')[0]?.toUpperCase() }}</el-avatar>
                  <div>
                    <div>{{ row.friend?.nickname || row.friend?.username }}</div>
                    <div style="font-size:12px;color:#909399">@{{ row.friend?.username }}</div>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="100">
              <template #default="{ row }">
                <el-tag v-if="isOnline(row.friend)" type="success" size="small">在线</el-tag>
                <el-tag v-else type="info" size="small">离线</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="200">
              <template #default="{ row }">
                <el-button type="primary" link @click="openChat(row.friend_id)">
                  <el-icon><component :is="'ChatDotRound'" /></el-icon> 发消息
                </el-button>
                <el-popconfirm title="确定删除该好友？" @confirm="handleRemove(row.friend_id)">
                  <template #reference>
                    <el-button type="danger" link>删除</el-button>
                  </template>
                </el-popconfirm>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!loading && friends.length === 0" description="暂无好友，点击上方按钮添加">
            <el-button type="primary" @click="addDialogVisible = true">添加好友</el-button>
          </el-empty>
        </el-card>
      </el-tab-pane>

      <!-- Pending Requests -->
      <el-tab-pane name="requests">
        <template #label>
          <el-badge :value="pendingRequests.length" :hidden="pendingRequests.length === 0" :max="99">
            <span>收到的请求</span>
          </el-badge>
        </template>
        <el-card>
          <el-table :data="pendingRequests" stripe v-loading="loadingRequests" style="width:100%">
            <el-table-column label="请求人" min-width="200">
              <template #default="{ row }">
                <div style="display:flex;align-items:center;gap:8px">
                  <el-avatar v-if="row.sender?.avatar" :src="row.sender.avatar" :size="32" />
                  <el-avatar v-else :size="32">{{ (row.sender?.username || '?')[0]?.toUpperCase() }}</el-avatar>
                  <div>
                    <div>{{ row.sender?.nickname || row.sender?.username }}</div>
                    <div style="font-size:12px;color:#909399">@{{ row.sender?.username }}</div>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="时间" width="170" />
            <el-table-column label="操作" width="200">
              <template #default="{ row }">
                <el-button type="success" size="small" @click="handleAccept(row.id)">接受</el-button>
                <el-button type="danger" size="small" @click="handleReject(row.id)">拒绝</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!loadingRequests && pendingRequests.length === 0" description="暂无好友请求" />
        </el-card>
      </el-tab-pane>

      <!-- Sent Requests -->
      <el-tab-pane label="发出的请求" name="sent">
        <el-card>
          <el-table :data="sentRequests" stripe v-loading="loadingSent" style="width:100%">
            <el-table-column label="接收人" min-width="200">
              <template #default="{ row }">
                <div style="display:flex;align-items:center;gap:8px">
                  <el-avatar v-if="row.receiver?.avatar" :src="row.receiver.avatar" :size="32" />
                  <el-avatar v-else :size="32">{{ (row.receiver?.username || '?')[0]?.toUpperCase() }}</el-avatar>
                  <div>
                    <div>{{ row.receiver?.nickname || row.receiver?.username }}</div>
                    <div style="font-size:12px;color:#909399">@{{ row.receiver?.username }}</div>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="时间" width="170" />
            <el-table-column label="状态" width="100">
              <el-tag type="warning" size="small">等待中</el-tag>
            </el-table-column>
          </el-table>
          <el-empty v-if="!loadingSent && sentRequests.length === 0" description="暂无发出的请求" />
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { listFriends, sendFriendRequest, removeFriend, listPendingRequests, listSentRequests, acceptRequest, rejectRequest, searchUsers } from '../api/friend'

const router = useRouter()
const activeTab = ref('friends')

const friends = ref([])
const loading = ref(false)
const pendingRequests = ref([])
const loadingRequests = ref(false)
const sentRequests = ref([])
const loadingSent = ref(false)

const addDialogVisible = ref(false)
const searchKeyword = ref('')
const searchResults = ref([])
const searched = ref(false)
const sendingTo = ref(null)

let searchTimer = null

const isOnline = (user) => {
  if (!user?.last_active_at) return false
  const diff = Date.now() - new Date(user.last_active_at).getTime()
  return diff < 5 * 60 * 1000 // 5 minutes
}

const loadFriends = async () => {
  loading.value = true
  try {
    const res = await listFriends()
    friends.value = res.data || []
  } finally {
    loading.value = false
  }
}

const loadPending = async () => {
  loadingRequests.value = true
  try {
    const res = await listPendingRequests()
    pendingRequests.value = res.data || []
  } finally {
    loadingRequests.value = false
  }
}

const loadSent = async () => {
  loadingSent.value = true
  try {
    const res = await listSentRequests()
    sentRequests.value = res.data || []
  } finally {
    loadingSent.value = false
  }
}

const handleSearchDebounced = () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(handleSearch, 400)
}

const handleSearch = async () => {
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

const handleSendRequest = async (user) => {
  sendingTo.value = user.id
  try {
    await sendFriendRequest({ receiver_id: user.id })
    ElMessage.success('好友请求已发送')
    searchResults.value = searchResults.value.filter(u => u.id !== user.id)
    loadSent()
  } finally {
    sendingTo.value = null
  }
}

const handleAccept = async (id) => {
  await acceptRequest(id)
  ElMessage.success('已接受好友请求')
  loadFriends()
  loadPending()
}

const handleReject = async (id) => {
  await rejectRequest(id)
  ElMessage.success('已拒绝好友请求')
  loadPending()
}

const handleRemove = async (id) => {
  await removeFriend(id)
  ElMessage.success('已删除好友')
  loadFriends()
}

const openChat = (friendId) => {
  router.push(`/messages/${friendId}`)
}

onMounted(() => {
  loadFriends()
  loadPending()
  loadSent()
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
.search-results {
  max-height: 320px;
  overflow-y: auto;
}
.search-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
}
.search-item:last-child {
  border-bottom: none;
}
.user-brief {
  display: flex;
  align-items: center;
  gap: 10px;
}
.user-handle {
  font-size: 12px;
  color: #909399;
}
.username {
  font-weight: 500;
}
</style>
