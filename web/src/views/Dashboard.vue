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
            <span>{{ $t('dashboard.selectedDomainCount', { count: checkedNodes.length }) }}</span>
            <el-button size="small" type="warning" @click="openBatchTransfer">{{ $t('domainDetail.batchTransfer') }}</el-button>
            <el-button size="small" type="primary" @click="openBatchAuthorize">{{ $t('domainDetail.batchAuthorize') }}</el-button>
            <el-button size="small" type="danger" @click="handleBatchCancelIndependence" v-if="hasMaterializedSelected">{{ $t('domainDetail.batchCancelIndependence') }}</el-button>
            <el-button size="small" type="danger" @click="handleBatchDelete">{{ $t('dashboard.batchDelete') }}</el-button>
          </div>

          <el-card class="tree-card">
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
                <div class="tree-node" :class="{ archived: data.status === 'archived' }">
                  <span class="domain-name">
                    <el-avatar v-if="auth.user?.id === data.owner_id ? data.owner?.avatar : (data.claimer?.avatar || data.owner?.avatar)" :src="auth.user?.id === data.owner_id ? data.owner?.avatar : (data.claimer?.avatar || data.owner?.avatar)" :size="20" style="margin-right:4px" />
                    {{ data.full_domain }}
                  </span>
                  <el-tag v-if="data.is_materialized" size="small" type="success">{{ $t('domainDetail.materialized') }}</el-tag>
                  <el-tag v-if="data.status === 'archived'" size="small" type="info">{{ $t('dashboard.archived') }}</el-tag>
                  <el-tag v-if="data.claimer_id && data.claimer_id !== data.owner_id" size="small" type="warning">{{ $t('common.claimer') }}: {{ data.claimer?.nickname || data.claimer?.username }}</el-tag>
                  <el-tag v-if="permMap[data.id]" :type="permTagType(permMap[data.id])" size="small" class="perm-badge">
                    {{ permLabel(permMap[data.id]) }}
                  </el-tag>
                  <el-tag size="small" type="info">{{ $t('dashboard.recordsCount', { count: data.records_count || 0 }) }}</el-tag>
                  <div class="node-actions" @click.stop v-if="data.owner_id === auth.user?.id">
                    <el-button size="small" text type="warning" @click="handleTransferDomain(data)">{{ $t('dashboard.transferDomain') }}</el-button>
                    <el-button v-if="!data.parent_id" size="small" text type="danger" @click="handleArchiveDomain(data)">{{ $t('dashboard.archiveDomain') }}</el-button>
                    <el-button v-else size="small" text type="danger" @click="handleReturnSubdomain(data)">{{ $t('dashboard.returnToClaimer') }}</el-button>
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
          <div class="group-header" @click="toggleGroupExpand(group.key)">
            <el-icon style="margin-right:8px"><component :is="expandedGroupKeys.has(group.key) ? 'ArrowDown' : 'ArrowRight'" /></el-icon>
            <span class="group-domain">{{ group.rootDomain }}</span>
            <el-divider direction="vertical" />
            <span class="group-grantor">
              {{ $t('domainDetail.grantedBy') }}:
              <el-avatar v-if="group.grantor?.avatar" :src="group.grantor.avatar" :size="20" style="vertical-align:middle;margin:0 4px" />
              {{ group.grantor?.username || $t('common.unknown') }}
            </span>
            <el-tag size="small" type="info" style="margin-left:8px">{{ group.entries.length }} {{ $t('domainDetail.permissions') }}</el-tag>
          </div>

          <div v-if="expandedGroupKeys.has(group.key)" class="group-entries">
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
                <el-button size="small" type="primary" @click="$router.push(`/domains/${entry.domain_node_id}`)">{{ $t('domainDetail.viewDomain') }}</el-button>
              </div>
            </div>
          </div>
        </el-card>
      </el-tab-pane>

      <!-- Transferred Away Tab -->
      <el-tab-pane :label="$t('dashboard.tabTransferredAway')" name="transferred">
        <el-card v-if="transferredAway.length === 0" class="empty-card">
          <el-empty :description="$t('dashboard.noTransferredAway')" />
        </el-card>

        <el-table v-else :data="transferredAway" stripe style="width:100%">
          <el-table-column :label="$t('admin.fullDomain')" min-width="200" show-overflow-tooltip>
            <template #default="{ row }">{{ row.node?.full_domain || '-' }}</template>
          </el-table-column>
          <el-table-column :label="$t('dashboard.transferredTo')" min-width="140">
            <template #default="{ row }">
              <div style="display:flex;align-items:center;gap:6px">
                <el-avatar v-if="row.to_user?.avatar" :src="row.to_user.avatar" :size="24" />
                <el-avatar v-else :size="24">{{ (row.to_user?.username || '?')[0]?.toUpperCase() }}</el-avatar>
                <span>{{ row.to_user?.username || ('User #' + row.to_user_id) }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" :label="$t('dashboard.transferredAt')" width="170" />
        </el-table>
      </el-tab-pane>

      <!-- Recycle Bin Tab -->
      <el-tab-pane :label="$t('dashboard.tabArchived')" name="archived">
        <el-card v-if="archivedDomains.length === 0" class="empty-card">
          <el-empty :description="$t('dashboard.noArchivedDomains')" />
        </el-card>

        <el-table v-else :data="archivedDomains" stripe style="width:100%">
          <el-table-column prop="full_domain" :label="$t('admin.fullDomain')" min-width="200" show-overflow-tooltip />
          <el-table-column :label="$t('admin.owner')" width="140">
            <template #default="{ row }">
              <span v-if="row.owner && row.owner.username">{{ row.owner.nickname || row.owner.username }}</span>
              <el-tag v-if="row.owner_id !== auth.user?.id && row.archived_by === auth.user?.id" type="warning" size="small" style="margin-left:6px">{{ $t('dashboard.transferred') }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="archived_at" :label="$t('dashboard.archivedAt')" width="170" />
          <el-table-column :label="$t('common.actions')" width="120" fixed="right">
            <template #default="{ row }">
              <el-button v-if="row.owner_id === auth.user?.id" link type="primary" size="small" @click="handleRestoreArchived(row)">{{ $t('dashboard.restoreDomain') }}</el-button>
              <el-tooltip v-else :content="$t('dashboard.transferredTooltip', { user: getOwnerName(row) })" placement="top">
                <el-button link type="info" size="small" disabled>{{ $t('dashboard.restoreDomain') }}</el-button>
              </el-tooltip>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- Trash Tab -->
      <el-tab-pane :label="$t('trash.title')" name="trash">
        <div style="display:flex;justify-content:flex-end;margin-bottom:12px">
          <el-button type="danger" size="small" @click="handleEmptyTrash" :disabled="trashTotal === 0">
            <el-icon><component :is="'Delete'" /></el-icon>
            {{ $t('trash.empty') }}
          </el-button>
        </div>

        <!-- Filter bar -->
        <div class="filter-bar">
          <el-input v-model="trashFilters.host" :placeholder="$t('trash.host')" clearable size="small" style="width:140px" @clear="loadTrash" @keyup.enter="loadTrash" />
          <el-select v-model="trashFilters.record_type" :placeholder="$t('trash.type')" clearable size="small" style="width:120px" @change="loadTrash">
            <el-option v-for="rt in trashRecordTypes" :key="rt.value" :label="rt.label" :value="rt.value" />
          </el-select>
          <el-select v-model="trashFilters.domain_id" :placeholder="$t('trash.domain')" clearable filterable size="small" style="width:180px" @change="loadTrash">
            <el-option v-for="d in trashDomains" :key="d.id" :label="d.full_domain" :value="d.id" />
          </el-select>
          <el-date-picker v-model="trashFilters.dateRange" type="daterange" :range-separator="'-'" :start-placeholder="$t('trash.trashedAt')" :end-placeholder="$t('trash.trashedAt')" size="small" style="width:230px" value-format="YYYY-MM-DD" @change="loadTrash" />
          <div class="filter-actions">
            <el-button size="small" type="primary" @click="loadTrash">{{ $t('common.search') }}</el-button>
            <el-button size="small" @click="resetTrashFilters">{{ $t('common.reset') }}</el-button>
          </div>
        </div>

        <!-- Batch action bar -->
        <div class="batch-bar" v-if="trashSelectedIds.length > 0">
          <span>{{ $t('domainDetail.selectedCount', { count: trashSelectedIds.length }) }}</span>
          <el-button size="small" type="success" @click="handleBatchRestore">{{ $t('trash.batchRestore') }}</el-button>
          <el-button size="small" type="danger" @click="handleTrashBatchDelete">{{ $t('trash.batchDelete') }}</el-button>
        </div>

        <!-- Table -->
        <el-table :data="trashItems" stripe v-loading="trashLoading" style="width:100%" @selection-change="handleTrashSelectionChange">
          <el-table-column type="selection" width="40" />
          <el-table-column prop="host" :label="$t('trash.host')" min-width="120" show-overflow-tooltip />
          <el-table-column prop="full_domain" :label="$t('trash.domain')" min-width="180" show-overflow-tooltip />
          <el-table-column prop="record_type" :label="$t('trash.type')" width="100" />
          <el-table-column prop="value" :label="$t('trash.value')" min-width="180" show-overflow-tooltip />
          <el-table-column prop="source" :label="$t('trash.source')" width="100">
            <template #default="{ row }">
              <el-tag v-if="row.source === 'provider'" size="small" type="warning">{{ $t('domainDetail.sourceProvider') }}</el-tag>
              <el-tag v-else size="small" type="success">{{ $t('domainDetail.sourcePlatform') }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="trashed_at" :label="$t('trash.trashedAt')" width="170" />
          <el-table-column :label="$t('common.actions')" width="150" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="handleRestore(row.id)">{{ $t('trash.restore') }}</el-button>
              <el-button link type="danger" size="small" @click="handlePermanentDelete(row.id)">{{ $t('trash.permanentDelete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>

        <el-empty v-if="!trashLoading && trashItems.length === 0" :description="$t('trash.noRecords')" />

        <el-pagination v-if="trashTotal > trashPageSize" :current-page="trashPage" :page-size="trashPageSize" :total="trashTotal"
          layout="total, sizes, prev, pager, next" :page-sizes="[20, 50, 100]"
          @current-change="handleTrashPageChange" @size-change="handleTrashSizeChange" style="margin-top:16px" />
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
          <el-select ref="transferSelectRef" v-model="transferTargetUserId" filterable remote :remote-method="searchUsersRemote" :loading="searchingUsers" :placeholder="$t('domainDetail.searchUser')" style="width:100%" @change="onTransferUserChange">
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
    <el-dialog v-model="showBatchAuthorize" :title="$t('domainDetail.batchAuthorize')" width="520px" destroy-on-close>
      <div style="margin-bottom:16px">
        <div style="font-weight:600;margin-bottom:8px">{{ $t('domainDetail.selectedDomains') }}</div>
        <div style="display:flex;flex-wrap:wrap;gap:4px">
          <el-tag v-for="node in checkedNodes" :key="node.id" size="small">{{ node.full_domain }}</el-tag>
        </div>
      </div>
      <el-form :model="batchGrantForm" label-width="100px">
        <el-form-item :label="$t('domainDetail.users')">
          <el-select ref="batchGrantSelectRef" v-model="batchGrantForm.target_user_ids" multiple filterable remote :remote-method="searchBatchUsersRemote" :loading="searchingBatchUsers" :placeholder="$t('domainDetail.searchUsers')" style="width:100%" collapse-tags collapse-tags-tooltip @change="onBatchGrantUserChange">
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
        <el-button type="primary" @click="handleBatchAuthorize">{{ $t('domainDetail.confirmBatchAuthorize') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getDomains, transferDomain, deleteDomain, demoteNode, getTransferredAway, batchDeleteDomains, archiveDomain, restoreDomain, getArchivedDomains, returnSubdomain } from '../api/domain'
import { createRootDomain } from '../api/admin'
import { getMyPermissions, batchGrantPermissions } from '../api/permission'
import { listProviders, listProviderDomains } from '../api/provider'
import { listTrash, restoreRecord, permanentDelete, emptyTrash, batchRestore } from '../api/trash'
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
const transferSelectRef = ref(null)
const batchGrantSelectRef = ref(null)

const activeTab = ref('domains')
const treeRef = ref(null)
const checkedKeys = ref([])
const transferredAway = ref([])
const archivedDomains = ref([])
const expandedGroupKeys = ref(new Set())
const toggleGroupExpand = (key) => {
  const s = new Set(expandedGroupKeys.value)
  if (s.has(key)) s.delete(key)
  else s.add(key)
  expandedGroupKeys.value = s
}

// Batch authorize dialog
const showBatchAuthorize = ref(false)
const searchingBatchUsers = ref(false)
const batchSelectableUsers = ref([])
const batchGrantForm = ref({
  level: 'write',
  allowed_types: [],
  target_user_ids: [],
})

// Recycle bin state
const trashItems = ref([])
const trashLoading = ref(false)
const trashPage = ref(1)
const trashPageSize = ref(20)
const trashTotal = ref(0)
const trashSelectedIds = ref([])
const trashDomains = ref([])
const trashFilters = reactive({ host: '', record_type: '', domain_id: '', dateRange: null })
const trashRecordTypes = [
  { label: 'A', value: 'A' },
  { label: 'AAAA', value: 'AAAA' },
  { label: 'CNAME', value: 'CNAME' },
  { label: 'ALIAS', value: 'ALIAS' },
  { label: 'MX', value: 'MX' },
  { label: 'TXT', value: 'TXT' },
  { label: 'CAA', value: 'CAA' },
  { label: 'NS', value: 'NS' },
  { label: 'SRV', value: 'SRV' },
]

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
        entries: [],
      })
    }

    let rules = null
    if (perm.host_rules) {
      try { rules = typeof perm.host_rules === 'string' ? JSON.parse(perm.host_rules) : perm.host_rules } catch { rules = null }
    }

    let scopeType = 'full'
    let scopeLabel = t('domainDetail.scopeFull')
    let ruleSummary = ''

    if (rules && Array.isArray(rules) && rules.length > 0) {
      const valid = rules.filter(r => r.value)
      if (valid.length === 1 && valid[0].type === 'exact') {
        scopeType = 'exact'
        scopeLabel = t('domainDetail.scopeExact')
      } else if (valid.length > 0) {
        scopeType = 'pattern'
        scopeLabel = t('domainDetail.scopePattern')
        const typeMap = { exact: t('domainDetail.ruleExact'), prefix: t('domainDetail.rulePrefix'), suffix: t('domainDetail.ruleSuffix'), contains: t('domainDetail.ruleContains'), regex: t('domainDetail.ruleRegex') }
        ruleSummary = valid.map(r => `${typeMap[r.type] || r.type}: ${r.value}`).join(', ')
      }
    } else if (perm.host_prefix) {
      scopeType = 'exact'
      scopeLabel = t('domainDetail.scopeExact')
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

const loadTransferredAway = async () => {
  try {
    const res = await getTransferredAway()
    transferredAway.value = res.data || []
  } catch { /* ignore */ }
}

const loadArchivedDomains = async () => {
  try {
    const res = await getArchivedDomains()
    archivedDomains.value = res.data || []
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

const onTransferUserChange = () => {
  nextTick(() => { transferSelectRef.value && (transferSelectRef.value.query = '') })
}

const onBatchGrantUserChange = () => {
  nextTick(() => { batchGrantSelectRef.value && (batchGrantSelectRef.value.query = '') })
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
      ElMessage.warning(t('domainDetail.noOwnedDomainsSelected'))
      return
    }
    await ElMessageBox.confirm(
      t('dashboard.confirmBatchTransferMsg', { count: nodes.length }),
      t('domainDetail.batchTransfer'),
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
    ElMessage.success(t('domainDetail.batchTransferResult', { success, fail: failed }))
  }

  showTransferDialog.value = false
  checkedKeys.value = []
  if (treeRef.value) treeRef.value.setCheckedKeys([])
  loadDomains()
  loadTransferredAway()
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

const handleArchiveDomain = async (data) => {
  try {
    await ElMessageBox.confirm(
      t('dashboard.archiveConfirm', { domain: data.full_domain }),
      t('dashboard.archiveDomain'),
      { type: 'warning', confirmButtonText: t('dashboard.confirmArchive') }
    )
  } catch { return }
  await archiveDomain(data.id)
  ElMessage.success(t('dashboard.archiveSuccess'))
  loadDomains()
}

const handleReturnSubdomain = async (data) => {
  try {
    await ElMessageBox.confirm(
      t('dashboard.returnConfirm', { domain: data.full_domain }),
      t('dashboard.returnToClaimer'),
      { type: 'warning', confirmButtonText: t('dashboard.confirmReturn') }
    )
  } catch { return }
  await returnSubdomain(data.id)
  ElMessage.success(t('dashboard.returnSuccess'))
  loadDomains()
}

const getOwnerName = (row) => row.owner?.nickname || row.owner?.username || t('dashboard.userId', { id: row.owner_id })

const handleRestoreArchived = async (data) => {
  await restoreDomain(data.id)
  ElMessage.success(t('dashboard.restoreSuccess'))
  loadArchivedDomains()
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
    ElMessage.success(t('domainDetail.batchAuthorizeResult', { success: successCount, fail: failCount }))
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
    t('domainDetail.batchCancelIndependenceConfirm', { count: materialized.length }),
    t('domainDetail.batchCancelIndependence'),
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
  ElMessage.success(t('domainDetail.batchCancelResult', { success, fail: failed }))
  checkedKeys.value = []
  if (treeRef.value) treeRef.value.setCheckedKeys([])
  loadDomains()
}

const handleBatchDelete = async () => {
  const owned = checkedNodes.value.filter(n => n.owner_id === auth.user?.id)
  if (owned.length === 0) {
    ElMessage.warning(t('domainDetail.noOwnedDomainsSelected'))
    return
  }

  try {
    await ElMessageBox.confirm(
      t('dashboard.batchDeleteConfirmText', { count: owned.length }),
      t('common.confirmDelete'),
      { type: 'warning' }
    )
  } catch {
    return
  }

  const ids = owned.map(n => n.id)
  try {
    const res = await batchDeleteDomains({ node_ids: ids })
    const data = res.data || {}
    let msg = t('dashboard.batchDeleteSuccess')
    if (data.skipped > 0) {
      msg += t('dashboard.skippedCount', { count: data.skipped })
    }
    ElMessage.success(msg)
  } catch (e) {
    ElMessage.error(e.response?.data?.message || t('dashboard.batchDeleteFailed'))
  }

  checkedKeys.value = []
  if (treeRef.value) treeRef.value.setCheckedKeys([])
  loadDomains()
}

// Recycle bin functions
const loadTrash = async () => {
  trashLoading.value = true
  try {
    const params = { page: trashPage.value, page_size: trashPageSize.value }
    if (trashFilters.host) params.host = trashFilters.host
    if (trashFilters.record_type) params.record_type = trashFilters.record_type
    if (trashFilters.domain_id) params.domain_id = trashFilters.domain_id
    if (trashFilters.dateRange && trashFilters.dateRange.length === 2) {
      params.trashed_after = trashFilters.dateRange[0]
      params.trashed_before = trashFilters.dateRange[1]
    }
    const res = await listTrash(params)
    trashItems.value = res.data.items || []
    trashTotal.value = res.data.total || 0
  } finally {
    trashLoading.value = false
  }
}

const loadTrashDomains = async () => {
  try {
    const res = await getDomains()
    const flatten = (nodes) => {
      const result = []
      for (const n of nodes) {
        result.push({ id: n.id, full_domain: n.full_domain })
        if (n.children) result.push(...flatten(n.children))
      }
      return result
    }
    trashDomains.value = flatten(res.data || [])
  } catch { /* ignore */ }
}

const handleTrashSelectionChange = (rows) => {
  trashSelectedIds.value = rows.map(r => r.id)
}

const handleRestore = async (id) => {
  await restoreRecord(id)
  ElMessage.success(t('trash.restoreSuccess'))
  loadTrash()
}

const handlePermanentDelete = async (id) => {
  try {
    await ElMessageBox.confirm(
      t('trash.permanentDeleteConfirm'),
      t('trash.permanentDelete'),
      { type: 'warning', confirmButtonText: t('common.confirmDelete') }
    )
    await permanentDelete(id)
    ElMessage.success(t('trash.deleteSuccess'))
    loadTrash()
  } catch (e) {
    if (e === 'cancel') return
  }
}

const handleBatchRestore = async () => {
  await batchRestore(trashSelectedIds.value)
  ElMessage.success(t('trash.batchRestoreSuccess', { count: trashSelectedIds.value.length }))
  trashSelectedIds.value = []
  loadTrash()
}

const handleTrashBatchDelete = async () => {
  try {
    await ElMessageBox.confirm(
      t('trash.permanentDeleteConfirm'),
      t('trash.batchDelete'),
      { type: 'warning', confirmButtonText: t('common.confirmDelete') }
    )
    for (const id of trashSelectedIds.value) {
      await permanentDelete(id)
    }
    ElMessage.success(t('trash.batchDeleteSuccess', { count: trashSelectedIds.value.length }))
    trashSelectedIds.value = []
    loadTrash()
  } catch (e) {
    if (e === 'cancel') return
  }
}

const handleEmptyTrash = async () => {
  try {
    await ElMessageBox.confirm(
      t('trash.emptyConfirm'),
      t('trash.empty'),
      { type: 'error', confirmButtonText: t('common.confirmDelete') }
    )
    await emptyTrash()
    ElMessage.success(t('trash.emptySuccess'))
    loadTrash()
  } catch (e) {
    if (e === 'cancel') return
  }
}

const resetTrashFilters = () => {
  trashFilters.host = ''
  trashFilters.record_type = ''
  trashFilters.domain_id = ''
  trashFilters.dateRange = null
  trashPage.value = 1
  loadTrash()
}

const handleTrashPageChange = (p) => {
  trashPage.value = p
  loadTrash()
}

const handleTrashSizeChange = (s) => {
  trashPageSize.value = s
  trashPage.value = 1
  loadTrash()
}

onMounted(() => {
  loadDomains()
  loadPermissions()
  loadProviders()
  loadTransferredAway()
  loadArchivedDomains()
  loadTrashDomains()

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

// Load trash/archived when tab becomes active
watch(activeTab, (tab) => {
  if (tab === 'trash') {
    loadTrash()
  } else if (tab === 'archived') {
    loadArchivedDomains()
  }
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
  line-height: 1.6;
}
.tree-node.archived {
  opacity: 0.6;
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
.tree-card {
  overflow-x: auto;
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
.filter-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
  align-items: center;
}
.filter-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
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

/* Mobile responsiveness */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }
  .page-header h2 {
    font-size: 18px;
  }
  .page-header .el-button {
    width: 100%;
  }
  .batch-bar {
    flex-wrap: wrap;
    gap: 6px;
  }
  .batch-bar .el-button {
    flex: 1 1 auto;
    min-width: 80px;
  }
  .tree-card {
    overflow-x: auto;
  }
  /* Allow tree node content to grow vertically when wrapping on mobile */
  .tree-card :deep(.el-tree-node__content) {
    height: auto;
    min-height: 30px;
    padding: 2px 0;
  }
  .tree-node {
    gap: 4px;
    padding: 4px 0;
    line-height: 1.6;
    align-items: flex-start;
  }
  .tree-node .el-tag {
    font-size: 11px;
    padding: 0 5px;
    height: 20px;
    line-height: 20px;
    margin: 1px 0;
    flex-shrink: 0;
  }
  .domain-name {
    font-size: 13px;
    flex: 1 1 100%;
    min-width: 0;
    max-width: none;
    word-break: break-all;
    white-space: normal;
    line-height: 1.6;
    align-items: flex-start;
    margin-bottom: 2px;
  }
  .node-actions {
    flex: 1 1 100%;
    margin-left: 0;
    margin-top: 2px;
    justify-content: flex-start;
    gap: 4px;
    flex-wrap: wrap;
  }
  .node-actions .el-button {
    padding: 3px 8px;
    font-size: 12px;
    min-height: 28px;
    margin: 1px 0;
  }
  .filter-bar {
    gap: 6px;
  }
  .filter-bar .el-input,
  .filter-bar .el-select,
  .filter-bar .el-date-editor {
    flex: 1 1 100%;
    width: 100% !important;
  }
  .filter-actions {
    width: 100%;
    justify-content: stretch;
  }
  .filter-actions .el-button {
    flex: 1;
  }
  .group-header {
    flex-wrap: wrap;
    gap: 6px;
  }
  .group-grantor {
    flex-basis: 100%;
    margin-left: 24px;
  }
  .managed-entry {
    flex-direction: column;
    align-items: flex-start;
  }
  .entry-actions {
    width: 100%;
  }
  .entry-actions .el-button {
    width: 100%;
  }
}

/* Extra-small screens */
@media (max-width: 480px) {
  .page-header h2 {
    font-size: 16px;
  }
  .subtitle {
    font-size: 12px;
  }
  .tree-card :deep(.el-tree-node__content) {
    min-height: 28px;
  }
  .tree-node {
    gap: 3px;
    padding: 2px 0;
    align-items: flex-start;
  }
  .domain-name {
    font-size: 12px;
    flex: 1 1 100%;
    margin-bottom: 1px;
  }
  .tree-node .el-tag {
    font-size: 10px;
    padding: 0 4px;
    height: 18px;
    line-height: 18px;
    flex-shrink: 0;
  }
  .node-actions {
    gap: 3px;
    margin-top: 1px;
  }
  .node-actions .el-button {
    padding: 2px 6px;
    font-size: 11px;
    min-height: 26px;
  }
  .batch-bar {
    gap: 4px;
    padding: 6px 8px;
  }
  .batch-bar .el-button {
    min-width: 60px;
    font-size: 11px;
    padding: 4px 6px;
  }
  .group-domain {
    font-size: 13px;
  }
  .managed-entry {
    gap: 6px;
  }
}
</style>
