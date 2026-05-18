<!-- web/src/views/Admin.vue -->
<template>
  <div>
    <div class="page-header">
      <h2>管理后台</h2>
      <p class="subtitle">用户管理、根域名配置与操作日志</p>
    </div>

    <el-card>
      <el-tabs v-model="activeTab">
        <el-tab-pane label="用户管理" name="users">
          <el-table :data="users" stripe v-loading="loading">
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="username" label="用户名" />
            <el-table-column prop="email" label="邮箱" />
            <el-table-column prop="role" label="角色">
              <template #default="{ row }">
                <el-tag :type="row.role === 'admin' ? 'danger' : 'info'" size="small">
                  {{ row.role === 'admin' ? '管理员' : '普通用户' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="token" label="DDNS Token" show-overflow-tooltip />
            <el-table-column prop="created_at" label="注册时间" width="180" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="根域名管理" name="domains">
          <el-form :model="domainForm" @submit.prevent="handleCreateRoot" inline>
            <el-form-item label="主机名">
              <el-input v-model="domainForm.host" placeholder="例如 example" />
            </el-form-item>
            <el-form-item label="后缀">
              <el-input v-model="domainForm.domain_suffix" placeholder="例如 com" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" native-type="submit">创建根域名</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="操作日志" name="logs">
          <el-table :data="logs" stripe v-loading="loading">
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="user_id" label="用户 ID" width="100" />
            <el-table-column prop="action" label="操作" />
            <el-table-column prop="target_type" label="目标类型" />
            <el-table-column prop="ip_address" label="IP 地址" width="140" />
            <el-table-column prop="created_at" label="时间" width="180" />
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { listUsers, listLogs, createRootDomain } from '../api/admin'
import { ElMessage } from 'element-plus'

const activeTab = ref('users')
const users = ref([])
const logs = ref([])
const loading = ref(false)
const domainForm = ref({ host: '', domain_suffix: '' })

const loadData = async () => {
  loading.value = true
  try {
    const [usersRes, logsRes] = await Promise.all([
      listUsers(),
      listLogs({ page: 1, page_size: 50 })
    ])
    users.value = usersRes.data
    logs.value = logsRes.data.items
  } finally {
    loading.value = false
  }
}

const handleCreateRoot = async () => {
  await createRootDomain(domainForm.value)
  ElMessage.success('根域名创建成功')
  domainForm.value = { host: '', domain_suffix: '' }
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
