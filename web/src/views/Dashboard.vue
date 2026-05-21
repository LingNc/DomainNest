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

    <el-tabs v-model="activeTab">
      <!-- My Domains Tab -->
      <el-tab-pane :label="$t('dashboard.tabMyDomains')" name="domains">
        <el-card v-if="!domains?.length" class="empty-card">
          <el-empty :description="$t('dashboard.noDomains')">
            <el-button v-if="auth.isAdmin" type="primary" size="small" @click="showCreateRoot = true">{{ $t('dashboard.createRootDomain') }}</el-button>
          </el-empty>
        </el-card>

        <template v-else>
          <!-- Batch action bar -->
          <div class="batch-bar" v-if="checkedNodes.length > 0">
            <span>{{ $t('dashboard.selectedCount', { count: checkedNodes.length }) }}</span>
            <el-button size="small" type="warning" @click="openBatchTransfer">{{ $t('dashboard.batchTransfer') }}</el-button>
            <el-button size="small" type="primary" @click="openBatchAuthorize">{{ $t('dashboard.batchAuthorize') }}</el-button>
            <el-button size="small" type="danger" @click="handleBatchCancelIndependence" v-if="hasMaterializedSelected">{{ $t('dashboard.batchCancelIndependence') }}</el-button>
          </div>

          <el-card>
            <el-tree
              ref="treeRef"
              :data="domains"
              :props="{ label: 'full_domain', children: 'children' }"
              show-checkbox
              check-strictly
              node-key="id"
              @check="handleTreeCheck"
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
                  <el-tag v-if="data.is_materialized" size="small" type="success">{{ $t('domainDetail.materialized') }}</el-tag>
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
        </template>
      </el-tab-pane>

      <!-- Managed Tab -->
      <el-tab-pane :label="$t('dashboard.tabManaged')" name="managed">
        <el-card v-if="permissionGroups.length === 0" class="empty-card">
          <el-empty :description="$t('dashboard.noManagedDomains')" />
        </el-card>

        <el-card v-for="group in permissionGroups" :key="group.key" class="managed-group">
          <div class="group-header" @click="group.expanded = !group.expanded">
            <el-icon style="margin-right:8px"><component :is="group.expanded ? 'ArrowDown' : 'ArrowRight'" /></el-icon>
            <span class="group-domain">{{ group.rootDomain }}</span>
            <el-divider direction="vertical" />
            <span class="group-grantor">
              {{ $t('dashboard.grantedBy') }}:
              <el-avatar v-if="group.grantor?.avatar" :src="group.grantor.avatar" :size="20" style="vertical-align:middle;margin:0 4px" />
              {{ group.grantor?.username || $t('common.unknown') }}
            </span>
            <el-tag size="small" type="info" style="margin-left:8px">{{ group.entries.length }} {{ $t('dashboard.permissions') }}</el-tag>
          </div>

          <div v-if="group.expanded" class="group-entries">
            <div v-for="entry in group.entries" :key="entry.id" class="managed-entry">
              <div class="entry-info">
                <el-tag :type="entry.scopeType === 'full' ? 'success' : entry.scopeType === 'exact' ? '' : 'warning'" size="small">
                  {{ entry.scopeLabel }}
                </el-tag>
                <el-tag :type="permTagType(entry.permission_level)" size="small">
                  {{ permLabel(entry.permission_level) }}
                </el-tag>
                <el-button v-if="entry.scopeType === 'exact' || entry.scopeType === 'full'" link type="primary" size="small" @click="$router.push(`/domains/${entry.domain_node_id}`)">
                  {{ entry.domainNode?.full_domain }}
                </el-button>
                <span v-if="entry.ruleSummary" class="entry-rule">{{ entry.ruleSummary }}</span>
                <el-tag v-if="entry.max_depth" size="small" type="info" style="margin-left:4px">{{ $t('domainDetail.depthLabel') }} {{ entry.max_depth }}</el-tag>
                <el-tag v-if="entry.allowedTypesStr" size="small" type="info" style="margin-left:4px">{{ entry.allowedTypesStr }}</el-tag>
              </div>
              <div class="entry-actions">
                <el-button size="small" type="primary" @click="$router.push(`/domains/${entry.domain_node_id}`)">{{ $t('dashboard.viewDomain') }}</el-button>
              </div>
            </div>
          </div>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <!-- Create root domain dialog -->
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

    <!-- Transfer dialog (single + batch) -->
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

    <!-- Batch authorize dialog -->
    <el-dialog v-model="showBatchAuthorize" :title="$t('dashboard.batchAuthorize')" width="520px" destroy-on-close>
      <div style="margin-bottom:16px">
        <div style="font-weight:600;margin-bottom:8px">{{ $t('dashboard.selectedDomains') }}</div>
        <div style="display:flex;flex-wrap:wrap;gap:4px">
          <el-tag v-for="node in checkedNodes" :key="node.id" size="small">{{ node.full_domain }}</el-tag>
        </div>
      </div>
      <el-form :model="batchGrantForm" label-width="100px">
        <el-form-item :label="$t('domainDetail.users')">
          <el-select v-model="batchGrantForm.target_user_ids" multiple filterable remote :remote-method="searchBatchUsersRemote" :loading="searchingBatchUsers" :placeholder="$t('domainDetail.searchUsers')" style="width:100%" collapse-tags collapse-tags-tooltip>
            <el-option v-for="u in batchSelectableUsers" :key="u.id" :label="`${u.nickname || u.username} (@${u.username})`" :value="u.id">
              <div style="display:flex;align-items:center;gap:8px">
                <el-avatar v-if="u.avatar" :src="u.avatar" :size="24" />
                <el-avatar v-else :size="24">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
                <span>{{ u.nickname || u.username }}</span>
                <span style="color:#909399;font-size:12px">@{{ u.username }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('domainDetail.permLevel')">
          <el-select v-model="batchGrantForm.level" style="width:100%">
            <el-option :label="$t('domainDetail.readOnlyOption')" value="read" />
            <el-option :label="$t('domainDetail.readWriteOption')" value="write" />
            <el-option :label="$t('domainDetail.adminOption')" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('domainDetail.recordTypes')">
          <el-checkbox-group v-model="batchGrantForm.allowed_types">
            <el-checkbox v-for="rt in recordTypes" :key="rt.value" :label="rt.value">{{ rt.value }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showBatchAuthorize = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleBatchAuthorize">{{ $t('dashboard.confirmBatchAuthorize') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getDomains, transferDomain, deleteDomain, demoteNode } from '../api/domain'
import { createRootDomain } from '../api/admin'
import { getMyPermissions, batchGrantPermissions } from '../api/permission'
import { listProviders, listProviderDomains } from '../api/provider'
import { searchAllUsers } from '../api/auth'
import { useAuthStore } from '../stores/auth'
import { useWebSocket } from '../composables/useWebSocket'
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

const activeTab = ref('domains')
const treeRef = ref(null)
const checkedKeys = ref([])

// Batch authorize dialog
const showBatchAuthorize = ref(false)
const searchingBatchUsers = ref(false)
const batchSelectableUsers = ref([])
const batchGrantForm = ref({
  level: 'write',
  allowed_types: [],
  target_user_ids: [],
})

const recordTypes = [
  { label: 'A', value: 'A' },
  { label: 'AAAA', value: 'AAAA' },
  { label: 'CNAME', value: 'CNAME' },
  { label: 'MX', value: 'MX' },
  { label: 'TXT', value: 'TXT' },
  { label: 'NS', value: 'NS' },
  { label: 'CAA', value: 'CAA' },
  { label: 'SRV', value: 'SRV' },
]

const permMap = computed(() => {
  const map = {}
  for (const p of permissions.value) {
    map[p.domain_node_id] = p.permission_level
  }
  return map
})

const permTagType = (level) => ({ read: 'info', write: 'success', admin: 'warning' }[level] || 'info')
const permLabel = (level) => ({ read: t('common.read'), write: t('common.write'), admin: t('common.admin') }[level] || level)

const handleTreeCheck = () => {
  if (treeRef.value) {
    checkedKeys.value = treeRef.value.getCheckedKeys()
  }
}

const checkedNodes = computed(() => {
  const nodeMap = new Map()
  const collect = (nodes) => {
    for (const n of nodes) {
      nodeMap.set(n.id, n)
      if (n.children) collect(n.children)
    }
  }
  collect(domains.value)
  return checkedKeys.value.map(id => nodeMap.get(id)).filter(Boolean)
})

const hasMaterializedSelected = computed(() =>
  checkedNodes.value.some(n => n.is_materialized && n.owner_id === auth.user?.id)
)

const permissionGroups = computed(() => {
  const groups = new Map()
  for (const perm of permissions.value) {
    const fullDomain = perm.domain_node?.full_domain || ''
    const parts = fullDomain.split('.')
    const rootDomain = parts.length >= 2 ? parts.slice(-2).join('.') : fullDomain
    const grantorId = perm.created_by
    const key = `${rootDomain}::${grantorId}`

    if (!groups.has(key)) {
      groups.set(key, {
        key,
        rootDomain,
        grantor: perm.creator,
        expanded: false,
        entries: [],
      })
    }

    let rules = null
    if (perm.host_rules) {
      try { rules = typeof perm.host_rules === 'string' ? JSON.parse(perm.host_rules) : perm.host_rules } catch { rules = null }
    }

    let scopeType = 'full'
    let scopeLabel = t('dashboard.scopeFull')
    let ruleSummary = ''

    if (rules && Array.isArray(rules) && rules.length > 0) {
      const valid = rules.filter(r => r.value)
      if (valid.length === 1 && valid[0].type === 'exact') {
        scopeType = 'exact'
        scopeLabel = t('dashboard.scopeExact')
      } else if (valid.length > 0) {
        scopeType = 'pattern'
        scopeLabel = t('dashboard.scopePattern')
        const typeMap = { exact: t('domainDetail.ruleExact'), prefix: t('domainDetail.rulePrefix'), suffix: t('domainDetail.ruleSuffix'), contains: t('domainDetail.ruleContains'), regex: t('domainDetail.ruleRegex') }
        ruleSummary = valid.map(r => `${typeMap[r.type] || r.type}: ${r.value}`).join(', ')
      }
    } else if (perm.host_prefix) {
      scopeType = 'exact'
      scopeLabel = t('dashboard.scopeExact')
      ruleSummary = perm.host_prefix
    }

    const allowedTypesStr = (perm.allowed_types && perm.allowed_types !== '[]')
      ? (typeof perm.allowed_types === 'string' ? JSON.parse(perm.allowed_types) : perm.allowed_types).join(', ')
      : ''

    groups.get(key).entries.push({
      ...perm,
      scopeType,
      scopeLabel,
      ruleSummary,
      allowedTypesStr,
    })
  }
  return Array.from(groups.values())
})

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

const searchBatchUsersRemote = async (query) => {
  if (!query) { batchSelectableUsers.value = []; return }
  searchingBatchUsers.value = true
  try {
    const res = await searchAllUsers(query)
    batchSelectableUsers.value = (res.data || []).filter(u => u.id !== auth.user?.id)
  } catch { /* ignore */ }
  searchingBatchUsers.value = false
}

const handleTransferDomain = (data) => {
  transferDomainId.value = data.id
  transferTargetUserId.value = null
  selectableUsers.value = []
  showTransferDialog.value = true
}

const openBatchTransfer = () => {
  transferDomainId.value = null // null means batch mode
  transferTargetUserId.value = null
  selectableUsers.value = []
  showTransferDialog.value = true
}

const handleTransferConfirm = async () => {
  if (!transferTargetUserId.value) {
    ElMessage.warning(t('domainDetail.searchUser'))
    return
  }

  if (transferDomainId.value) {
    // Single domain transfer
    await ElMessageBox.confirm(
      t('dashboard.confirmTransferMsg', { domain: domains.value.find(d => d.id === transferDomainId.value)?.full_domain || '' }),
      t('dashboard.transferDomain'),
      { type: 'warning', confirmButtonText: t('dashboard.confirmTransfer') }
    )
    await transferDomain(transferDomainId.value, { target_user_id: transferTargetUserId.value })
    ElMessage.success(t('dashboard.transferSuccess'))
  } else {
    // Batch transfer
    const nodes = checkedNodes.value.filter(n => n.owner_id === auth.user?.id)
    if (nodes.length === 0) {
      ElMessage.warning(t('dashboard.noOwnedDomainsSelected'))
      return
    }
    await ElMessageBox.confirm(
      t('dashboard.confirmBatchTransferMsg', { count: nodes.length }),
      t('dashboard.batchTransfer'),
      { type: 'warning' }
    )
    let success = 0
    let failed = 0
    for (const node of nodes) {
      try {
        await transferDomain(node.id, { target_user_id: transferTargetUserId.value })
        success++
      } catch {
        failed++
      }
    }
    ElMessage.success(t('dashboard.batchTransferResult', { success, fail: failed }))
  }

  showTransferDialog.value = false
  checkedKeys.value = []
  if (treeRef.value) treeRef.value.setCheckedKeys([])
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

const openBatchAuthorize = () => {
  batchGrantForm.value = { level: 'write', allowed_types: [], target_user_ids: [] }
  batchSelectableUsers.value = []
  showBatchAuthorize.value = true
}

const handleBatchAuthorize = async () => {
  if (!batchGrantForm.value.target_user_ids?.length) {
    ElMessage.warning(t('domainDetail.searchUsers'))
    return
  }

  const data = {
    domain_node_ids: checkedNodes.value.map(n => n.id),
    target_user_ids: batchGrantForm.value.target_user_ids,
    level: batchGrantForm.value.level,
  }
  if (batchGrantForm.value.allowed_types.length > 0) {
    data.allowed_types = batchGrantForm.value.allowed_types
  }

  try {
    const res = await batchGrantPermissions(data)
    const results = res.data || []
    const successCount = results.filter(r => r.success).length
    const failCount = results.filter(r => !r.success).length
    ElMessage.success(t('dashboard.batchAuthorizeResult', { success: successCount, fail: failCount }))
    showBatchAuthorize.value = false
    checkedKeys.value = []
    if (treeRef.value) treeRef.value.setCheckedKeys([])
  } catch (e) {
    // error shown by interceptor
  }
}

const handleBatchCancelIndependence = async () => {
  const materialized = checkedNodes.value.filter(n => n.is_materialized && n.owner_id === auth.user?.id)
  if (materialized.length === 0) return

  await ElMessageBox.confirm(
    t('dashboard.batchCancelIndependenceConfirm', { count: materialized.length }),
    t('dashboard.batchCancelIndependence'),
    { type: 'warning' }
  )

  let success = 0
  let failed = 0
  for (const node of materialized) {
    try {
      await demoteNode(node.id)
      success++
    } catch {
      failed++
    }
  }
  ElMessage.success(t('dashboard.batchCancelResult', { success, fail: failed }))
  checkedKeys.value = []
  if (treeRef.value) treeRef.value.setCheckedKeys([])
  loadDomains()
}

onMounted(() => {
  loadDomains()
  loadPermissions()
  loadProviders()

  // WebSocket listener for tree updates
  const { on: wsOn, off: wsOff } = useWebSocket()
  const handleTreeUpdate = () => {
    loadDomains()
    loadPermissions()
  }
  wsOn('domain_tree_update', handleTreeUpdate)

  onUnmounted(() => {
    wsOff('domain_tree_update', handleTreeUpdate)
  })
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
.batch-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #f0f9ff;
  border-radius: 4px;
}
.managed-group {
  margin-bottom: 12px;
}
.group-header {
  display: flex;
  align-items: center;
  cursor: pointer;
  padding: 4px 0;
}
.group-domain {
  font-weight: 600;
  font-size: 15px;
}
.group-grantor {
  display: flex;
  align-items: center;
  font-size: 13px;
  color: #606266;
}
.group-entries {
  margin-top: 12px;
  padding-left: 24px;
}
.managed-entry {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
  flex-wrap: wrap;
  gap: 8px;
}
.managed-entry:last-child {
  border-bottom: none;
}
.entry-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.entry-rule {
  font-size: 12px;
  color: #909399;
  background: #f5f5f5;
  padding: 1px 6px;
  border-radius: 3px;
}
</style>
