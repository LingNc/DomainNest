<template>
  <div>
    <div class="page-header">
      <h2>我的日志</h2>
      <p class="subtitle">查看您的操作记录</p>
    </div>
    <el-card>
      <el-table :data="logs" stripe v-loading="loading" style="width:100%">
        <el-table-column prop="action" label="操作" min-width="120" />
        <el-table-column prop="target_type" label="目标类型" min-width="100" />
        <el-table-column prop="ip_address" label="IP 地址" width="130" />
        <el-table-column prop="detail" label="详情" min-width="200" show-overflow-tooltip />
        <el-table-column prop="created_at" label="时间" width="170" />
      </el-table>
      <el-pagination v-if="total > pageSize" :current-page="page" :page-size="pageSize" :total="total"
        layout="total, prev, pager, next" @current-change="handlePageChange" style="margin-top:16px" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getMyLogs } from '../api/auth'

const logs = ref([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const loadLogs = async () => {
  loading.value = true
  try {
    const res = await getMyLogs({ page: page.value, page_size: pageSize.value })
    logs.value = res.data.items || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

const handlePageChange = (p) => {
  page.value = p
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
</style>
