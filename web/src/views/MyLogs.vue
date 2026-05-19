<template>
  <div>
    <div class="page-header">
      <h2>我的日志</h2>
      <p class="subtitle">查看您的操作记录</p>
    </div>
    <el-card>
      <div class="filter-bar">
        <el-input v-model="filters.q" placeholder="关键词搜索" clearable size="small" style="width:160px" />
        <el-select v-model="filters.action" placeholder="操作类型" clearable filterable size="small" style="width:160px" @change="loadLogs">
          <el-option-group v-for="group in actionGroups" :key="group.label" :label="group.label">
            <el-option v-for="item in group.options" :key="item.value" :label="item.label" :value="item.value" />
          </el-option-group>
        </el-select>
        <el-select v-model="filters.target_type" placeholder="目标类型" clearable multiple collapse-tags collapse-tags-tooltip size="small" style="width:200px" @change="loadLogs">
          <el-option v-for="item in targetTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
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
import { actionGroups, targetTypeOptions } from '../constants/operationLogs'

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

const filters = reactive({ action: '', target_type: [], q: '', dateRange: null })

const loadLogs = async () => {
  loading.value = true
  try {
    const params = { page: page.value, page_size: pageSize.value }
    if (filters.action) params.action = filters.action
    if (filters.target_type.length > 0) params.target_type = filters.target_type.join(',')
    if (filters.q) params.q = filters.q
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
  filters.target_type = []
  filters.q = ''
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
