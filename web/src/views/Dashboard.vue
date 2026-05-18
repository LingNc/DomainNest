<!-- web/src/views/Dashboard.vue -->
<template>
  <div>
    <div class="page-header">
      <div>
        <h2>域名管理</h2>
        <p class="subtitle">管理您的域名节点和子域名</p>
      </div>
      <el-button v-if="isAdmin" type="primary" @click="showCreateRoot = true">
        <el-icon><component :is="'Plus'" /></el-icon>
        创建根域名
      </el-button>
    </div>

    <el-card v-if="domains.length === 0" class="empty-card">
      <el-empty description="暂无域名">
        <el-button v-if="isAdmin" type="primary" @click="showCreateRoot = true">创建根域名</el-button>
      </el-empty>
    </el-card>

    <el-card v-else>
      <el-tree
        :data="domains"
        :props="{ label: 'full_domain', children: 'children' }"
        @node-click="handleNodeClick"
        default-expand-all
        :expand-on-click-node="false"
      >
        <template #default="{ data }">
          <div class="tree-node">
            <span class="domain-name">{{ data.full_domain }}</span>
            <el-tag size="small" type="info">{{ data.records?.length || 0 }} 条记录</el-tag>
          </div>
        </template>
      </el-tree>
    </el-card>

    <el-dialog v-model="showCreateRoot" title="创建根域名" width="400px">
      <el-form :model="rootForm">
        <el-form-item label="主机名">
          <el-input v-model="rootForm.host" placeholder="例如 example" />
        </el-form-item>
        <el-form-item label="域名后缀">
          <el-input v-model="rootForm.domain_suffix" placeholder="例如 com" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateRoot = false">取消</el-button>
        <el-button type="primary" @click="handleCreateRoot">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getDomains } from '../api/domain'
import { createRootDomain } from '../api/admin'
import { ElMessage } from 'element-plus'

const router = useRouter()
const domains = ref([])
const showCreateRoot = ref(false)
const rootForm = ref({ host: '', domain_suffix: '' })

const isAdmin = (() => {
  try { return JSON.parse(localStorage.getItem('user'))?.role === 'admin' }
  catch { return false }
})()

const loadDomains = async () => {
  const res = await getDomains()
  domains.value = res.data
}

const handleNodeClick = (data) => {
  router.push(`/domains/${data.id}`)
}

const handleCreateRoot = async () => {
  await createRootDomain(rootForm.value)
  ElMessage.success('根域名创建成功')
  showCreateRoot.value = false
  rootForm.value = { host: '', domain_suffix: '' }
  loadDomains()
}

onMounted(loadDomains)
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
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
.tree-node {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 4px 0;
  width: 100%;
}
.domain-name {
  font-size: 14px;
}
.empty-card {
  min-height: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
