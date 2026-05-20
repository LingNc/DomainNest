<!-- web/src/views/Dashboard.vue -->
<template>
  <div>
    <div class="page-header">
      <div>
        <h2>{{ $t('dashboard.title') }}</h2>
        <p class="subtitle">{{ $t('dashboard.subtitle') }}</p>
      </div>
      <el-button v-if="auth.isAdmin" type="primary" size="small" @click="showCreateRoot = true">
        <el-icon><component :is="'Plus'" /></el-icon>
        {{ $t('dashboard.createRootDomain') }}
      </el-button>
    </div>

    <el-card v-if="domains.length === 0" class="empty-card">
      <el-empty :description="$t('dashboard.noDomains')">
        <el-button v-if="auth.isAdmin" type="primary" size="small" @click="showCreateRoot = true">{{ $t('dashboard.createRootDomain') }}</el-button>
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
            <span class="domain-name">
              <el-avatar v-if="data.owner?.avatar" :src="data.owner.avatar" :size="20" style="margin-right:4px" />
              {{ data.full_domain }}
            </span>
            <el-tag v-if="permMap[data.id]" :type="permTagType(permMap[data.id])" size="small" class="perm-badge">
              {{ permLabel(permMap[data.id]) }}
            </el-tag>
            <el-tag size="small" type="info">{{ $t('dashboard.recordsCount', { count: data.records?.length || 0 }) }}</el-tag>
          </div>
        </template>
      </el-tree>
    </el-card>

    <el-dialog v-model="showCreateRoot" :title="$t('dashboard.createRootDomain')" width="400px" destroy-on-close>
      <el-form :model="rootForm">
        <el-form-item :label="$t('dashboard.host')">
          <el-input v-model="rootForm.host" :placeholder="$t('dashboard.hostPlaceholder')" />
        </el-form-item>
        <el-form-item :label="$t('dashboard.domainSuffix')">
          <el-input v-model="rootForm.domain_suffix" :placeholder="$t('dashboard.domainSuffixPlaceholder')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateRoot = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreateRoot">{{ $t('dashboard.create') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getDomains } from '../api/domain'
import { createRootDomain } from '../api/admin'
import { getMyPermissions } from '../api/permission'
import { useAuthStore } from '../stores/auth'
import { ElMessage } from 'element-plus'

const router = useRouter()
const auth = useAuthStore()
const { t } = useI18n()
const domains = ref([])
const permissions = ref([])
const showCreateRoot = ref(false)
const rootForm = ref({ host: '', domain_suffix: '' })

const permMap = computed(() => {
  const map = {}
  for (const p of permissions.value) {
    map[p.domain_node_id] = p.permission_level
  }
  return map
})

const permTagType = (level) => ({ read: 'info', write: 'success', admin: 'warning' }[level] || 'info')
const permLabel = (level) => ({ read: t('common.read'), write: t('common.write'), admin: t('common.admin') }[level] || level)

const loadDomains = async () => {
  const res = await getDomains()
  domains.value = res.data
}

const loadPermissions = async () => {
  try {
    const res = await getMyPermissions()
    permissions.value = res.data || []
  } catch { /* ignore */ }
}

const handleNodeClick = (data) => {
  router.push(`/domains/${data.id}`)
}

const handleCreateRoot = async () => {
  await createRootDomain(rootForm.value)
  ElMessage.success(t('dashboard.createSuccess'))
  showCreateRoot.value = false
  rootForm.value = { host: '', domain_suffix: '' }
  loadDomains()
}

onMounted(() => {
  loadDomains()
  loadPermissions()
})
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
.perm-badge {
  margin-left: -4px;
}
.empty-card {
  min-height: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
