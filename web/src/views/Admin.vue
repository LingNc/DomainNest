<!-- web/src/views/Admin.vue -->
<template>
  <el-container>
    <el-header>
      <div class="header-content">
        <h2>DomainNest Admin</h2>
        <el-button @click="$router.push('/dashboard')">Back</el-button>
      </div>
    </el-header>
    <el-main>
      <el-tabs v-model="activeTab">
        <el-tab-pane label="Users" name="users">
          <el-table :data="users" stripe>
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="username" label="Username" />
            <el-table-column prop="email" label="Email" />
            <el-table-column prop="role" label="Role" />
            <el-table-column prop="token" label="DDNS Token" />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="Root Domains" name="domains">
          <el-form :model="domainForm" @submit.prevent="handleCreateRoot" inline>
            <el-form-item label="Host">
              <el-input v-model="domainForm.host" placeholder="xxxx" />
            </el-form-item>
            <el-form-item label="Suffix">
              <el-input v-model="domainForm.domain_suffix" placeholder="com" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" native-type="submit">Create Root Domain</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="Logs" name="logs">
          <el-table :data="logs" stripe>
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="user_id" label="User ID" width="100" />
            <el-table-column prop="action" label="Action" />
            <el-table-column prop="target_type" label="Target" />
            <el-table-column prop="ip_address" label="IP" />
            <el-table-column prop="created_at" label="Time" />
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-main>
  </el-container>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { listUsers, listLogs, createRootDomain } from '../api/admin'
import { ElMessage } from 'element-plus'

const activeTab = ref('users')
const users = ref([])
const logs = ref([])
const domainForm = ref({ host: '', domain_suffix: '' })

const loadData = async () => {
  const [usersRes, logsRes] = await Promise.all([
    listUsers(),
    listLogs({ page: 1, page_size: 50 })
  ])
  users.value = usersRes.data
  logs.value = logsRes.data.items
}

const handleCreateRoot = async () => {
  await createRootDomain(domainForm.value)
  ElMessage.success('Root domain created')
  domainForm.value = { host: '', domain_suffix: '' }
}

onMounted(loadData)
</script>

<style scoped>
.el-header {
  background: #409eff;
  color: white;
  line-height: 60px;
}
.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
