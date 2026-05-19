<template>
  <div>
    <div class="page-header">
      <h2>我的日志</h2>
      <p class="subtitle">查看您的操作记录</p>
    </div>
    <el-card>
      <div class="filter-bar">
        <el-select v-model="filters.action" placeholder="操作类型" clearable size="small" style="width:140px" @change="loadLogs">
          <el-option label="注册" value="register" />
          <el-option label="登录" value="login" />
          <el-option label="创建域名" value="create_domain" />
          <el-option label="转让域名" value="transfer_domain" />
          <el-option label="删除域名" value="delete_domain" />
          <el-option label="创建记录" value="create_record" />
          <el-option label="更新记录" value="update_record" />
          <el-option label="删除记录" value="delete_record" />
          <el-option label="切换记录" value="toggle_record" />
          <el-option label="批量删除" value="batch_delete" />
          <el-option label="授权" value="grant_permission" />
          <el-option label="撤销权限" value="revoke_permission" />
          <el-option label="分配邀请" value="grant_invite" />
        </el-select>
        <el-select v-model="filters.target_type" placeholder="目标类型" clearable size="small" style="width:120px" @change="loadLogs">
          <el-option label="域名节点" value="domain_node" />
          <el-option label="DNS 记录" value="dns_record" />
          <el-option label="用户" value="user" />
          <el-option label="设置" value="setting" />
        </el-select>
        <el-date-picker v-model="filters.dateRange" type="daterange" range-separator="至" start-placeholder="开始日期" end-placeholder="结束日期" size="small" style="width:260px" value-format="YYYY-MM-DD" />
        <el-button size="small" type="primary" @click="loadLogs">搜索</el-button>
        <el-button size="small" @click="resetFilters">重置</el-button>
      </div>
      <el-table :data="logs" stripe v-loading="loading" style="width:100%">
        <el-table-column prop="action" label="操作" min-width="120" />
        <el-table-column prop="target_type" label="目标类型" min-width="100" />
        <el-table-column prop="ip_address" label="IP 地址" width="130" />
        <el-table-column label="详情" min-width="200">
          <template #default="{ row }">
            <template v-if="row.detail && row.detail !== 'null'">
              <el-tag v-for="(val, key) in parseDetail(row.detail)" :key="key" size="small" style="margin:2px" type="info">
                {{ formatDetailKey(key) }}: {{ formatDetailValue(val) }}
              </el-tag>
            </template>
            <span v-else style="color:#909399">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="时间" width="170" />
      </el-table>
      <el-empty v-if="!loading && logs.length === 0" description="暂无操作记录" />
      <el-pagination v-if="total > pageSize" :current-page="page" :page-size="pageSize" :total="total"
        layout="total, sizes, prev, pager, next" :page-sizes="[20, 50, 100]"
        @current-change="handlePageChange" @size-change="handleSizeChange" style="margin-top:16px" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { getMyLogs } from '../api/auth'

const logs = ref([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const keyMap = { username: '用户名', full_domain: '域名', host: '主机记录', type: '记录类型', value: '记录值', amount: '数量', enabled: '状态', ids: '记录ID', target_user_id: '目标用户', invited_by: '邀请人', provider_name: '提供商' }

const parseDetail = (json) => {
  try { return JSON.parse(json) } catch { return {} }
}

const formatDetailKey = (key) => keyMap[key] || key

const formatDetailValue = (val) => {
  if (Array.isArray(val)) return val.join(', ')
  if (typeof val === 'boolean') return val ? '启用' : '禁用'
  return String(val)
}

const filters = reactive({ action: '', target_type: '', dateRange: null })

const loadLogs = async () => {
  loading.value = true
  try {
    const params = { page: page.value, page_size: pageSize.value }
    if (filters.action) params.action = filters.action
    if (filters.target_type) params.target_type = filters.target_type
    if (filters.dateRange && filters.dateRange.length === 2) {
      params.start_time = filters.dateRange[0]
      params.end_time = filters.dateRange[1]
    }
    const res = await getMyLogs(params)
    logs.value = res.data.items || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

const resetFilters = () => {
  filters.action = ''
  filters.target_type = ''
  filters.dateRange = null
  page.value = 1
  loadLogs()
}

const handlePageChange = (p) => {
  page.value = p
  loadLogs()
}

const handleSizeChange = (s) => {
  pageSize.value = s
  page.value = 1
  loadLogs()
}

onMounted(loadLogs)
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
.filter-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
  align-items: center;
}
</style>
