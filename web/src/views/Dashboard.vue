<!-- web/src/views/Dashboard.vue -->
<template>
  <el-container>
    <el-header>
      <div class="header-content">
        <h2>DomainNest</h2>
        <div>
          <el-button @click="$router.push('/settings')">Settings</el-button>
          <el-button @click="$router.push('/admin')">Admin</el-button>
          <el-button @click="handleLogout">Logout</el-button>
        </div>
      </div>
    </el-header>
    <el-main>
      <el-card>
        <template #header>
          <div class="card-header">
            <span>My Domains</span>
          </div>
        </template>
        <el-tree
          :data="domains"
          :props="{ label: 'full_domain', children: 'children' }"
          @node-click="handleNodeClick"
        >
          <template #default="{ data }">
            <div class="tree-node">
              <span>{{ data.full_domain }}</span>
              <el-tag size="small" type="info">{{ data.records?.length || 0 }} records</el-tag>
            </div>
          </template>
        </el-tree>
      </el-card>
    </el-main>
  </el-container>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getDomains } from '../api/domain'

const router = useRouter()
const domains = ref([])

const loadDomains = async () => {
  const res = await getDomains()
  domains.value = res.data
}

const handleNodeClick = (data) => {
  router.push(`/domains/${data.id}`)
}

const handleLogout = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('user')
  router.push('/login')
}

onMounted(loadDomains)
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
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.tree-node {
  display: flex;
  align-items: center;
  gap: 10px;
}
</style>
