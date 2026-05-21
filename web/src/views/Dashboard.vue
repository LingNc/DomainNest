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
            <div class="node-actions" @click.stop v-if="data.owner_id === auth.user?.id">
              <el-button size="small" text type="warning" @click="handleTransferDomain(data)">{{ $t('dashboard.transferDomain') }}</el-button>
              <el-button size="small" text type="danger" @click="handleDeleteDomain(data)">{{ $t('dashboard.deleteDomain') }}</el-button>
            </div>
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

    <el-dialog v-model="showTransferDialog" :title="$t('dashboard.transferDomain')" width="420px" destroy-on-close>
      <el-alert type="warning" :closable="false" style="margin-bottom:16px">
        {{ $t('dashboard.transferWarning') }}
      </el-alert>
      <el-form label-width="80px">
        <el-form-item :label="$t('domainDetail.targetUser')">
          <el-select v-model="transferTargetUserId" filterable remote :remote-method="searchUsersRemote" :loading="searchingUsers" :placeholder="$t('domainDetail.searchUser')" style="width:100%">
            <el-option v-for="u in selectableUsers" :key="u.id" :label="`${u.nickname || u.username} (@${u.username})`" :value="u.id">
              <div style="display:flex;align-items:center;gap:8px">
                <el-avatar v-if="u.avatar" :src="u.avatar" :size="24" />
                <el-avatar v-else :size="24">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
                <span>{{ u.nickname || u.username }}</span>
                <span style="color:#909399;font-size:12px">@{{ u.username }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showTransferDialog = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="warning" @click="handleTransferConfirm">{{ $t('dashboard.confirmTransfer') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getDomains, transferDomain, deleteDomain } from '../api/domain'
import { createRootDomain } from '../api/admin'
import { getMyPermissions } from '../api/permission'
import { listProviders, listProviderDomains } from '../api/provider'
import { searchAllUsers } from '../api/auth'
import { useAuthStore } from '../stores/auth'
import { ElMessage, ElMessageBox } from 'element-plus'

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
const showTransferDialog = ref(false)
const transferDomainId = ref(null)
const transferTargetUserId = ref(null)
const selectableUsers = ref([])
const searchingUsers = ref(false)

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

const searchUsersRemote = async (query) => {
  if (!query) { selectableUsers.value = []; return }
  searchingUsers.value = true
  try {
    const res = await searchAllUsers(query)
    selectableUsers.value = (res.data || []).filter(u => u.id !== auth.user?.id)
  } catch { /* ignore */ }
  searchingUsers.value = false
}

const handleTransferDomain = (data) => {
  transferDomainId.value = data.id
  transferTargetUserId.value = null
  selectableUsers.value = []
  showTransferDialog.value = true
}

const handleTransferConfirm = async () => {
  if (!transferTargetUserId.value) {
    ElMessage.warning(t('domainDetail.searchUser'))
    return
  }
  await ElMessageBox.confirm(
    t('dashboard.confirmTransferMsg', { domain: domains.value.find(d => d.id === transferDomainId.value)?.full_domain || '' }),
    t('dashboard.transferDomain'),
    { type: 'warning', confirmButtonText: t('dashboard.confirmTransfer') }
  )
  await transferDomain(transferDomainId.value, { target_user_id: transferTargetUserId.value })
  ElMessage.success(t('dashboard.transferSuccess'))
  showTransferDialog.value = false
  loadDomains()
}

const handleDeleteDomain = async (data) => {
  await ElMessageBox.confirm(
    t('dashboard.confirmDeleteDomain', { domain: data.full_domain }),
    t('dashboard.deleteDomain'),
    { type: 'error', confirmButtonText: t('common.confirmDelete') }
  )
  await deleteDomain(data.id)
  ElMessage.success(t('dashboard.deleteDomainSuccess'))
  loadDomains()
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
  gap: 6px;
  padding: 2px 0;
  width: 100%;
  flex-wrap: wrap;
}
.domain-name {
  font-size: 14px;
  display: flex;
  align-items: center;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.perm-badge {
  flex-shrink: 0;
}
.node-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-left: auto;
}
.empty-card {
  min-height: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
