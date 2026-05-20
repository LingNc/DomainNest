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

    <el-card v-if="!domains?.length" class="empty-card">
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
      <el-form>
        <el-form-item :label="$t('admin.selectProvider')">
          <el-select v-model="selectedProviderId" :placeholder="$t('admin.selectProvider')" style="width:100%">
            <el-option v-for="p in providers" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('admin.selectDomain')">
          <el-select v-model="selectedDomain" :placeholder="$t('admin.selectDomain')" style="width:100%" :loading="loadingDomains" :disabled="!selectedProviderId">
            <el-option v-for="d in providerDomains" :key="d.domain_name" :label="d.domain_name" :value="d.domain_name" />
          </el-select>
        </el-form-item>
      </el-form>
      <el-empty v-if="providers.length === 0" :description="$t('admin.noProvidersHint')" :image-size="60" />
      <template #footer>
        <el-button @click="showCreateRoot = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :disabled="providers.length === 0" @click="handleCreateRoot">{{ $t('dashboard.create') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getDomains } from '../api/domain'
import { createRootDomain } from '../api/admin'
import { getMyPermissions } from '../api/permission'
import { listProviders, listProviderDomains } from '../api/provider'
import { useAuthStore } from '../stores/auth'
import { ElMessage } from 'element-plus'

const router = useRouter()
const auth = useAuthStore()
const { t } = useI18n()
const domains = ref([])
const permissions = ref([])
const showCreateRoot = ref(false)
const providers = ref([])
const providerDomains = ref([])
const selectedProviderId = ref('')
const selectedDomain = ref('')
const loadingDomains = ref(false)

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

const loadProviders = async () => {
  try {
    const res = await listProviders()
    providers.value = res.data || []
  } catch { /* ignore */ }
}

watch(selectedProviderId, async (id) => {
  providerDomains.value = []
  selectedDomain.value = ''
  if (!id) return
  loadingDomains.value = true
  try {
    const res = await listProviderDomains(id)
    providerDomains.value = res.data || []
  } catch { /* ignore */ } finally {
    loadingDomains.value = false
  }
})

const handleNodeClick = (data) => {
  router.push(`/domains/${data.id}`)
}

const handleCreateRoot = async () => {
  if (providers.value.length === 0) return
  if (!selectedProviderId.value || !selectedDomain.value) {
    ElMessage.warning(t('admin.selectProviderAndDomain'))
    return
  }
  try {
    await createRootDomain({ provider_id: selectedProviderId.value, domain_name: selectedDomain.value })
    ElMessage.success(t('dashboard.createSuccess'))
    showCreateRoot.value = false
    selectedProviderId.value = ''
    selectedDomain.value = ''
    loadDomains()
  } catch { /* error shown by interceptor */ }
}

onMounted(() => {
  loadDomains()
  loadPermissions()
  loadProviders()
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
