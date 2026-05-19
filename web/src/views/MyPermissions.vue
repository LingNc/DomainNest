<!-- web/src/views/MyPermissions.vue -->
<template>
  <div>
    <div class="page-header">
      <h2>我的权限</h2>
      <p class="subtitle">其他用户授予我的域名访问权限</p>
    </div>

    <el-card>
      <el-table :data="permissions" stripe v-loading="loading" style="width:100%">
        <el-table-column prop="domain_node.full_domain" label="域名" min-width="200">
          <template #default="{ row }">
            <el-button link type="primary" @click="$router.push(`/domains/${row.domain_node_id}`)">
              {{ row.domain_node?.full_domain || 'Domain #' + row.domain_node_id }}
            </el-button>
          </template>
        </el-table-column>
        <el-table-column prop="permission_level" label="权限级别" width="120">
          <template #default="{ row }">
            <el-tag :type="permTagType(row.permission_level)" size="small">
              {{ permLabel(row.permission_level) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="allowed_types" label="记录类型限制" min-width="160">
          <template #default="{ row }">
            <span v-if="row.allowed_types">{{ row.allowed_types }}</span>
            <span v-else style="color:#909399">全部</span>
          </template>
        </el-table-column>
        <el-table-column prop="allowed_ips" label="IP 限制" min-width="160">
          <template #default="{ row }">
            <span v-if="row.allowed_ips">{{ row.allowed_ips }}</span>
            <span v-else style="color:#909399">不限</span>
          </template>
        </el-table-column>
        <el-table-column label="授权人" min-width="120">
          <template #default="{ row }">
            <div style="display:flex;align-items:center;gap:6px">
              <el-avatar v-if="row.creator?.avatar" :src="row.creator.avatar" :size="24" />
              <el-avatar v-else :size="24">{{ (row.creator?.username || '?')[0]?.toUpperCase() }}</el-avatar>
              <span>{{ row.creator?.username || '—' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="授权时间" width="170" />
      </el-table>
      <el-empty v-if="!loading && permissions.length === 0" description="暂无被授予的权限" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getMyPermissions } from '../api/permission'

const permissions = ref([])
const loading = ref(false)

const permTagType = (level) => ({ read: 'info', write: 'success', admin: 'warning' }[level] || 'info')
const permLabel = (level) => ({ read: '只读', write: '读写', admin: '管理员' }[level] || level)

const loadData = async () => {
  loading.value = true
  try {
    const res = await getMyPermissions()
    permissions.value = res.data || []
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
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
