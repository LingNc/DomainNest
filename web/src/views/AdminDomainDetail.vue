<!-- web/src/views/AdminDomainDetail.vue -->
<template>
  <div>
    <el-button text @click="$router.push('/admin?tab=domains')" style="margin-bottom:12px">
      <el-icon><component :is="'ArrowLeft'" /></el-icon>
      {{ $t('common.back') }}
    </el-button>

    <div v-loading="loading">
      <template v-if="detail">
        <!-- Stats card -->
        <el-card shadow="never" style="margin-bottom:16px">
          <div style="display:flex;flex-wrap:wrap;gap:24px;align-items:center">
            <div style="display:flex;align-items:center;gap:8px">
              <el-avatar v-if="detail.domain?.owner?.avatar" :src="detail.domain.owner.avatar" :size="28" />
              <el-avatar v-else :size="28">{{ (detail.domain?.owner?.username || '?')[0]?.toUpperCase() }}</el-avatar>
              <span style="font-weight:600;font-size:16px">{{ detail.domain?.full_domain }}</span>
              <el-tag :type="detail.domain?.status === 'active' ? 'success' : 'info'" size="small">
                {{ detail.domain?.status === 'active' ? $t('admin.statusNormal') : $t('dashboard.archived') }}
              </el-tag>
            </div>
            <el-divider direction="vertical" />
            <div style="display:flex;gap:16px;font-size:13px;color:#606266">
              <span>{{ $t('admin.totalRecords') }}: <b>{{ recordStats.total }}</b></span>
              <span>{{ $t('admin.enabledRecords') }}: <b style="color:#67c23a">{{ recordStats.enabled }}</b></span>
              <span>{{ $t('admin.disabledRecords') }}: <b style="color:#909399">{{ recordStats.disabled }}</b></span>
              <span>{{ $t('admin.platformRecords') }}: <b>{{ recordStats.platform_count }}</b></span>
              <span>{{ $t('admin.providerRecords') }}: <b>{{ recordStats.provider_count }}</b></span>
            </div>
            <el-divider v-if="detail.domain?.provider" direction="vertical" />
            <div v-if="detail.domain?.provider" style="font-size:13px;color:#606266">
              <el-tag type="primary" size="small" style="margin-right:6px">{{ $t('admin.providerBound') }}</el-tag>
              {{ detail.domain.provider.name }}
            </div>
            <div v-else style="font-size:13px;color:#909399">
              <el-tag type="info" size="small" style="margin-right:6px">{{ $t('admin.noProvider') }}</el-tag>
            </div>
          </div>
        </el-card>

        <!-- Tabs -->
        <el-tabs v-model="activeTab" style="margin-bottom:16px">
          <!-- Info tab -->
          <el-tab-pane :label="$t('admin.info')" name="info">
            <el-card shadow="never">
              <template #header>
                <div style="display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:8px">
                  <span style="font-weight:600;font-size:16px">{{ detail.domain?.full_domain }}</span>
                  <div style="display:flex;gap:8px">
                    <el-button size="small" type="primary" @click="handleChangeOwner">{{ $t('admin.changeOwner') }}</el-button>
                    <el-button size="small" type="danger" @click="handleDeleteDomain">{{ $t('domainDetail.deleteDomain') }}</el-button>
                  </div>
                </div>
              </template>
              <el-descriptions :column="2" size="small">
                <el-descriptions-item :label="$t('admin.fullDomain')">{{ detail.domain?.full_domain }}</el-descriptions-item>
                <el-descriptions-item :label="$t('admin.detailKeyHost')">{{ detail.domain?.host }}</el-descriptions-item>
                <el-descriptions-item :label="$t('common.status')">
                  <el-tag :type="detail.domain?.status === 'active' ? 'success' : 'info'" size="small">
                    {{ detail.domain?.status === 'active' ? $t('admin.statusNormal') : $t('dashboard.archived') }}
                  </el-tag>
                </el-descriptions-item>
                <el-descriptions-item :label="$t('admin.currentOwner')">
                  <div style="display:flex;align-items:center;gap:6px">
                    <el-avatar v-if="detail.domain?.owner?.avatar" :src="detail.domain.owner.avatar" :size="20" />
                    <el-avatar v-else :size="20">{{ (detail.domain?.owner?.username || '?')[0]?.toUpperCase() }}</el-avatar>
                    <span>{{ detail.domain?.owner?.username || detail.domain?.owner_id }}</span>
                  </div>
                </el-descriptions-item>
                <el-descriptions-item :label="$t('admin.detailKeyProvider')">
                  {{ detail.domain?.provider?.name || '-' }}
                </el-descriptions-item>
                <el-descriptions-item :label="$t('common.createdAt')">{{ detail.domain?.created_at }}</el-descriptions-item>
              </el-descriptions>

              <!-- Provider info -->
              <div v-if="detail.domain?.provider" style="margin-top:16px">
                <el-divider content-position="left">{{ $t('admin.providerBound') }}</el-divider>
                <el-descriptions :column="3" size="small">
                  <el-descriptions-item :label="$t('admin.providerName')">{{ detail.domain.provider.name }}</el-descriptions-item>
                  <el-descriptions-item :label="$t('admin.providerType')">{{ detail.domain.provider.provider_type || '-' }}</el-descriptions-item>
                  <el-descriptions-item :label="$t('admin.providerStatus')">
                    <el-tag :type="detail.domain.provider.status === 'active' ? 'success' : 'info'" size="small">
                      {{ detail.domain.provider.status || '-' }}
                    </el-tag>
                  </el-descriptions-item>
                </el-descriptions>
              </div>
            </el-card>
          </el-tab-pane>

          <!-- Records tab -->
          <el-tab-pane :label="$t('admin.records')" name="records">
            <el-card shadow="never">
              <template #header>
                <div style="display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:8px">
                  <span style="font-weight:600">{{ $t('admin.records') }}</span>
                  <div style="display:flex;align-items:center;gap:8px">
                    <el-checkbox v-model="includeTrashed" @change="loadRecords">{{ $t('admin.includeTrashed') }}</el-checkbox>
                    <el-radio-group v-model="recordViewMode" size="small">
                      <el-radio-button value="tree">{{ $t('domainDetail.treeView') }}</el-radio-button>
                      <el-radio-button value="flat">{{ $t('domainDetail.flatView') }}</el-radio-button>
                    </el-radio-group>
                  </div>
                </div>
              </template>

              <!-- tree view -->
              <el-table
                v-if="recordViewMode === 'tree'"
                :data="treeRecords"
                row-key="treeId"
                :tree-props="{ children: 'children' }"
                stripe
                v-loading="recordsLoading"
                :row-class-name="treeRowClass"
              >
                <el-table-column prop="host" :label="$t('admin.host')" min-width="140">
                  <template #default="{ row }">
                    <template v-if="row.isGroup">
                      <el-icon style="margin-right:4px"><component :is="'Folder'" /></el-icon>
                      <strong>{{ row.host }}</strong>
                    </template>
                    <template v-else>
                      <span :style="row.virtual ? 'color:#909399;font-style:italic' : ''">{{ row.host }}</span>
                    </template>
                  </template>
                </el-table-column>
                <el-table-column prop="record_type" :label="$t('admin.type')" width="80">
                  <template #default="{ row }">
                    <span :style="row.virtual ? 'color:#909399' : ''">{{ row.record_type }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="value" :label="$t('admin.value')" width="140" show-overflow-tooltip>
                  <template #default="{ row }">
                    <span :style="row.virtual ? 'color:#909399;font-style:italic' : ''">{{ row.value }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="priority" :label="$t('domainDetail.priority')" width="70">
                  <template #default="{ row }">{{ row.virtual ? '' : (row.priority ?? '-') }}</template>
                </el-table-column>
                <el-table-column prop="ttl" :label="$t('admin.ttl')" width="70">
                  <template #default="{ row }">{{ row.virtual ? '' : row.ttl }}</template>
                </el-table-column>
                <el-table-column :label="$t('common.status')" min-width="80">
                  <template #default="{ row }">
                    <template v-if="!row.virtual">
                      <el-switch :model-value="row.enabled" size="small" @change="(val) => handleToggleRecord(row, val)" />
                    </template>
                  </template>
                </el-table-column>
                <el-table-column :label="$t('admin.source')" min-width="80">
                  <template #default="{ row }">
                    <template v-if="!row.virtual">
                      <el-tag :type="row.source === 'provider' ? 'warning' : 'success'" size="small">
                        {{ row.source === 'provider' ? $t('admin.recordSourceProvider') : $t('admin.recordSourcePlatform') }}
                      </el-tag>
                    </template>
                  </template>
                </el-table-column>
                <el-table-column :label="$t('common.actions')" min-width="80" fixed="right">
                  <template #default="{ row }">
                    <template v-if="row.isGroup || row.virtual" />
                    <template v-else>
                      <el-button link type="danger" size="small" @click="handleDeleteRecord(row)">{{ $t('common.delete') }}</el-button>
                    </template>
                  </template>
                </el-table-column>
              </el-table>

              <!-- flat view -->
              <el-table v-else :data="paginatedFlatRecords" stripe v-loading="recordsLoading" row-key="id" :row-class-name="flatRowClass" :span-method="flatSpanMethod">
                <el-table-column prop="host" :label="$t('admin.host')" min-width="180">
                  <template #default="{ row }">
                    <template v-if="row.isGroupHeader">
                      <div class="group-header-content">
                        <el-icon class="group-chevron" @click="toggleGroup(row.groupKey)"><component :is="isGroupCollapsed(row.groupKey) ? 'ArrowRight' : 'ArrowDown'" /></el-icon>
                        <strong @click="toggleGroup(row.groupKey)">{{ row.groupLabel }}</strong>
                        <el-tag size="small" type="info" style="margin-left:8px" @click="toggleGroup(row.groupKey)">{{ row.count }}</el-tag>
                      </div>
                    </template>
                    <template v-else>
                      <span>{{ row.host }}</span>
                    </template>
                  </template>
                </el-table-column>
                <el-table-column prop="record_type" :label="$t('admin.type')" width="80">
                  <template #default="{ row }">
                    <span v-if="!row.isGroupHeader">{{ row.record_type }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="value" :label="$t('admin.value')" width="140" show-overflow-tooltip>
                  <template #default="{ row }">
                    <span v-if="!row.isGroupHeader">{{ row.value }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="priority" :label="$t('domainDetail.priority')" width="70">
                  <template #default="{ row }">
                    <span v-if="!row.isGroupHeader">{{ row.priority ?? '-' }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="ttl" :label="$t('admin.ttl')" width="70">
                  <template #default="{ row }">
                    <span v-if="!row.isGroupHeader">{{ row.ttl }}</span>
                  </template>
                </el-table-column>
                <el-table-column :label="$t('common.status')" min-width="80">
                  <template #default="{ row }">
                    <template v-if="!row.isGroupHeader">
                      <el-switch :model-value="row.enabled" size="small" @change="(val) => handleToggleRecord(row, val)" />
                    </template>
                  </template>
                </el-table-column>
                <el-table-column :label="$t('admin.source')" min-width="80">
                  <template #default="{ row }">
                    <template v-if="!row.isGroupHeader">
                      <el-tag :type="row.source === 'provider' ? 'warning' : 'success'" size="small">
                        {{ row.source === 'provider' ? $t('admin.recordSourceProvider') : $t('admin.recordSourcePlatform') }}
                      </el-tag>
                    </template>
                  </template>
                </el-table-column>
                <el-table-column :label="$t('common.actions')" min-width="80" fixed="right">
                  <template #default="{ row }">
                    <template v-if="!row.isGroupHeader">
                      <el-button link type="danger" size="small" @click="handleDeleteRecord(row)">{{ $t('common.delete') }}</el-button>
                    </template>
                  </template>
                </el-table-column>
              </el-table>
              <el-pagination
                v-if="recordViewMode === 'flat' && groupedFlatRecords.length > 0"
                v-model:current-page="flatPage"
                v-model:page-size="flatPageSize"
                :total="groupedFlatRecords.length"
                :page-sizes="[20, 50, 100, 200]"
                :pager-count="5"
                layout="total, sizes, prev, pager, next"
                style="margin-top:16px"
              />
              <el-empty v-if="!recordsLoading && !records.length" :description="$t('admin.noRecords')" :image-size="60" />
            </el-card>
          </el-tab-pane>

          <!-- Permissions tab -->
          <el-tab-pane :label="$t('admin.permissions')" name="permissions">
            <el-card shadow="never">
              <template #header>
                <span style="font-weight:600">{{ $t('admin.domainPermissions') }}</span>
              </template>
              <div v-if="!detail.permissions?.length" style="color:#909399;font-size:13px">{{ $t('admin.noPermissions') }}</div>
              <div v-for="p in detail.permissions" :key="p.id" class="perm-item">
                <div style="display:flex;align-items:center;gap:8px">
                  <el-avatar v-if="p.user?.avatar" :src="p.user.avatar" :size="22" />
                  <el-avatar v-else :size="22">{{ (p.user?.username || '?')[0]?.toUpperCase() }}</el-avatar>
                  <span style="font-size:13px">{{ p.user?.username || ('User #' + p.user_id) }}</span>
                  <el-tag :type="p.permission_level === 'read' ? 'info' : p.permission_level === 'write' ? 'success' : 'warning'" size="small">
                    {{ p.permission_level }}
                  </el-tag>
                </div>
                <el-button link type="danger" size="small" @click="handleRevokePermission(p.user_id)">{{ $t('admin.revokePermissionAction') }}</el-button>
              </div>
            </el-card>
          </el-tab-pane>

          <!-- Transfer history tab -->
          <el-tab-pane :label="$t('admin.transferHistory')" name="transferHistory">
            <el-card shadow="never">
              <template #header>
                <span style="font-weight:600">{{ $t('admin.transferHistory') }}</span>
              </template>
              <div v-if="!detail.transfer_history?.length" style="color:#909399;font-size:13px">{{ $t('admin.noTransferHistory') }}</div>
              <el-timeline v-else>
                <el-timeline-item v-for="log in detail.transfer_history" :key="log.id" :timestamp="log.created_at" placement="top">
                  <div style="font-size:13px">
                    <span>{{ log.from_user?.username || ('User #' + log.from_user_id) }}</span>
                    <span style="margin:0 6px">&rarr;</span>
                    <span>{{ log.to_user?.username || ('User #' + log.to_user_id) }}</span>
                  </div>
                </el-timeline-item>
              </el-timeline>
            </el-card>
          </el-tab-pane>
        </el-tabs>
      </template>
      <el-empty v-else-if="!loading" :description="$t('admin.noDetailData')" :image-size="80" />
    </div>

    <!-- Change Owner dialog -->
    <el-dialog v-model="showChangeOwner" :title="$t('admin.changeOwner')" width="420px" destroy-on-close>
      <p style="margin-bottom:16px">
        {{ $t('admin.assignTo', { domain: detail?.domain?.full_domain }) }}
      </p>
      <el-select v-model="newOwnerId" :placeholder="$t('admin.selectUser')" style="width:100%">
        <el-option
          v-for="u in users"
          :key="u.id"
          :label="`${u.nickname || u.username} (@${u.username})`"
          :value="u.id"
        >
          <div style="display:flex;align-items:center;gap:8px">
            <el-avatar v-if="u.avatar" :src="u.avatar" :size="24" />
            <el-avatar v-else :size="24">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
            <span>{{ u.nickname || u.username }}</span>
            <span style="color:#909399;font-size:12px">@{{ u.username }}</span>
          </div>
        </el-option>
      </el-select>
      <template #footer>
        <el-button @click="showChangeOwner = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="confirmChangeOwner">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getAdminDomainDetail, adminChangeOwner, adminRevokePermission, adminBatchDeleteDomains, listUsers, getAdminDomainRecords, adminDeleteRecord, adminToggleRecord } from '../api/admin'
import { ElMessage, ElMessageBox } from 'element-plus'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const detail = ref(null)
const loading = ref(false)
const users = ref([])
const activeTab = ref('info')

const showChangeOwner = ref(false)
const newOwnerId = ref(null)

// Records state
const records = ref([])
const recordsLoading = ref(false)
const includeTrashed = ref(false)
const recordStats = ref({ total: 0, enabled: 0, disabled: 0, platform_count: 0, provider_count: 0 })

// View mode state (flat default)
const recordViewMode = ref('flat')
const sortBy = ref('')
const sortOrder = ref('asc')

// Collapsed groups for flat view
const collapsedGroups = ref(new Set())

const toggleGroup = (groupKey) => {
  if (collapsedGroups.value.has(groupKey)) {
    collapsedGroups.value.delete(groupKey)
  } else {
    collapsedGroups.value.add(groupKey)
  }
}
const isGroupCollapsed = (groupKey) => collapsedGroups.value.has(groupKey)

const syncStatusLabel = (status) => {
  switch (status) {
    case 'synced': return t('admin.syncSynced')
    case 'pending': return t('admin.syncPending')
    case 'failed': return t('admin.syncFailed')
    default: return t('admin.syncUnknown')
  }
}

// Sorting
const sortedRecords = computed(() => {
  const recs = [...records.value]
  if (!sortBy.value) return recs
  const order = sortOrder.value === 'asc' ? 1 : -1
  recs.sort((a, b) => {
    const va = a[sortBy.value] ?? ''
    const vb = b[sortBy.value] ?? ''
    if (typeof va === 'string') return va.localeCompare(vb) * order
    if (typeof va === 'number') return (va - vb) * order
    return 0
  })
  return recs
})

// Tree view computed
const treeRecords = computed(() => {
  const recs = sortedRecords.value

  // Separate provider, grouped, and ungrouped records
  const grouped = new Map()
  const ungrouped = []
  const providerRecords = []
  for (const rec of recs) {
    if (rec.source === 'provider') {
      providerRecords.push(rec)
    } else if (rec.group_tag) {
      if (!grouped.has(rec.group_tag)) grouped.set(rec.group_tag, [])
      grouped.get(rec.group_tag).push(rec)
    } else {
      ungrouped.push(rec)
    }
  }

  // Build subdomain tree from a list of records
  const buildSubdomainTree = (recs) => {
    const hostMap = new Map()
    for (const rec of recs) {
      if (!hostMap.has(rec.host)) {
        hostMap.set(rec.host, { host: rec.host, records: [], children: [] })
      }
      hostMap.get(rec.host).records.push(rec)
    }

    const getParent = (host) => {
      if (host === '@') return null
      const parts = host.split('.')
      const parent = parts.slice(1).join('.')
      return parent || '@'
    }

    const ensureNode = (host) => {
      if (!hostMap.has(host)) {
        hostMap.set(host, { host, records: [], children: [], virtual: true })
      }
      return hostMap.get(host)
    }

    const linked = new Set()
    const linkToParent = (host) => {
      if (linked.has(host)) return
      linked.add(host)
      const node = hostMap.get(host)
      const parentHost = getParent(host)
      if (parentHost === null) return
      const parent = ensureNode(parentHost)
      parent.children.push(node)
      linkToParent(parentHost)
    }

    for (const host of hostMap.keys()) {
      linkToParent(host)
    }

    const roots = []
    for (const [host, node] of hostMap) {
      if (getParent(host) === null) roots.push(node)
    }

    const flatten = (nodes) => {
      const rows = []
      for (const node of nodes) {
        if (node.records.length > 0) {
          const parentRow = { ...node.records[0], treeId: `${node.host}:${node.records[0].id}`, children: [] }
          for (let i = 1; i < node.records.length; i++) {
            parentRow.children.push({ ...node.records[i], treeId: `${node.host}:${node.records[i].id}` })
          }
          if (node.children.length > 0) {
            parentRow.children.push(...flatten(node.children))
          }
          rows.push(parentRow)
        } else if (node.virtual && node.children.length > 0) {
          const childRows = flatten(node.children)
          rows.push({
            treeId: `virtual:${node.host}`,
            host: node.host,
            record_type: '—',
            value: `${childRows.length} records`,
            virtual: true,
            children: childRows,
          })
        }
      }
      return rows
    }

    return flatten(roots)
  }

  const result = []

  if (providerRecords.length > 0) {
    result.push({
      treeId: 'group:provider',
      host: t('domainDetail.sourceProvider'),
      record_type: '—',
      value: `${providerRecords.length} records`,
      virtual: true,
      isGroup: true,
      children: buildSubdomainTree(providerRecords),
    })
  }

  for (const [tag, recs] of grouped) {
    result.push({
      treeId: `group:${tag}`,
      host: tag,
      record_type: '—',
      value: `${recs.length} records`,
      virtual: true,
      isGroup: true,
      children: buildSubdomainTree(recs),
    })
  }

  result.push(...buildSubdomainTree(ungrouped))
  return result
})

const treeRowClass = ({ row }) => row.virtual ? 'virtual-row' : ''

// Grouped flat records
const groupedFlatRecords = computed(() => {
  const recs = [...sortedRecords.value]
  recs.sort((a, b) => {
    if (a.source === 'provider' && b.source !== 'provider') return -1
    if (a.source !== 'provider' && b.source === 'provider') return 1
    const tagA = a.source === 'provider' ? '___provider___' : (a.group_tag || '___ungrouped___')
    const tagB = b.source === 'provider' ? '___provider___' : (b.group_tag || '___ungrouped___')
    if (tagA !== tagB) return tagA.localeCompare(tagB)
    return a.host.localeCompare(b.host)
  })

  const result = []
  let lastGroup = null
  for (const rec of recs) {
    const currentGroup = rec.source === 'provider' ? '__provider__' : (rec.group_tag || '__ungrouped__')
    if (currentGroup === '__ungrouped__') {
      result.push(rec)
      lastGroup = currentGroup
      continue
    }
    if (currentGroup !== lastGroup) {
      let label
      if (currentGroup === '__provider__') {
        label = t('domainDetail.sourceProvider')
      } else {
        label = rec.group_tag
      }
      result.push({
        id: `header_${currentGroup}`,
        isGroupHeader: true,
        groupLabel: label,
        groupKey: currentGroup,
        count: recs.filter(r => {
          const g = r.source === 'provider' ? '__provider__' : (r.group_tag || '__ungrouped__')
          return g === currentGroup
        }).length,
      })
      lastGroup = currentGroup
    }
    if (!collapsedGroups.value.has(currentGroup)) {
      result.push(rec)
    }
  }
  return result
})

const flatPage = ref(1)
const flatPageSize = ref(50)

const paginatedFlatRecords = computed(() => {
  const all = groupedFlatRecords.value
  const start = (flatPage.value - 1) * flatPageSize.value
  return all.slice(start, start + flatPageSize.value)
})

const flatSpanMethod = ({ row, columnIndex }) => {
  if (row.isGroupHeader) {
    if (columnIndex === 0) return { rowspan: 1, colspan: 1 }
    if (columnIndex === 1) return { rowspan: 1, colspan: 99 }
    return { rowspan: 0, colspan: 0 }
  }
}

const flatRowClass = ({ row }) => {
  if (row.isGroupHeader) return 'group-header-row'
  if (row.group_tag) return 'group-row'
  return ''
}

const loadDetail = async () => {
  loading.value = true
  try {
    const res = await getAdminDomainDetail(route.params.id)
    detail.value = res.data
  } catch {
    detail.value = null
  } finally {
    loading.value = false
  }
}

const loadRecords = async () => {
  recordsLoading.value = true
  try {
    const params = {}
    if (includeTrashed.value) params.include_trashed = 'true'
    const res = await getAdminDomainRecords(route.params.id, params)
    records.value = res.data?.records || []
    recordStats.value = res.data?.stats || { total: 0, enabled: 0, disabled: 0, platform_count: 0, provider_count: 0 }
  } catch {
    records.value = []
  } finally {
    recordsLoading.value = false
  }
}

const loadUsers = async () => {
  try {
    const res = await listUsers()
    users.value = res.data || []
  } catch { /* ignore */ }
}

const handleToggleRecord = async (row, val) => {
  try {
    await adminToggleRecord(route.params.id, row.id, { enabled: val })
    row.enabled = val
    // Update stats
    if (val) {
      recordStats.value.enabled++
      recordStats.value.disabled--
    } else {
      recordStats.value.enabled--
      recordStats.value.disabled++
    }
    ElMessage.success(t('admin.recordStatusUpdated'))
  } catch {
    // revert on failure
  }
}

const handleDeleteRecord = async (row) => {
  await ElMessageBox.confirm(t('admin.deleteRecordConfirm'), t('admin.deleteRecord'), { type: 'warning' })
  await adminDeleteRecord(route.params.id, row.id)
  ElMessage.success(t('admin.recordDeleted'))
  loadRecords()
}

const handleChangeOwner = () => {
  newOwnerId.value = null
  showChangeOwner.value = true
}

const confirmChangeOwner = async () => {
  if (!newOwnerId.value) {
    ElMessage.warning(t('admin.selectTargetUser'))
    return
  }
  await adminChangeOwner(route.params.id, newOwnerId.value)
  ElMessage.success(t('admin.domainAssigned'))
  showChangeOwner.value = false
  loadDetail()
}

const handleRevokePermission = async (userId) => {
  await ElMessageBox.confirm(t('domainDetail.confirmRevoke'), t('domainDetail.revokePermTitle'), { type: 'warning' })
  await adminRevokePermission(route.params.id, userId)
  ElMessage.success(t('domainDetail.revokeSuccess'))
  loadDetail()
}

const handleDeleteDomain = async () => {
  await ElMessageBox.confirm(
    t('common.confirmDelete'),
    t('common.confirmDelete'),
    { type: 'warning' }
  )
  await adminBatchDeleteDomains({ node_ids: [Number(route.params.id)] })
  ElMessage.success(t('admin.batchDeleteSuccess'))
  router.push('/admin?tab=domains')
}

// Load records when records tab is activated
watch(activeTab, (tab) => {
  if (tab === 'records' && records.value.length === 0 && detail.value) {
    loadRecords()
  }
})

onMounted(() => {
  loadDetail()
  loadUsers()
})
</script>

<style scoped>
.perm-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 0;
  border-bottom: 1px solid #f0f0f0;
}
.perm-item:last-child {
  border-bottom: none;
}
.group-header-content {
  display: flex;
  align-items: center;
  cursor: pointer;
  user-select: none;
}
.group-header-content:hover {
  opacity: 0.85;
}
.group-chevron {
  margin-right: 6px;
}
:deep(.virtual-row) {
  opacity: 0.6;
}
:deep(.group-row) {
  background-color: #fafafa;
}
:deep(.group-header-row) {
  background-color: #f0f9eb !important;
  cursor: pointer;
}
:deep(.group-header-row td) {
  background-color: #f0f9eb !important;
  font-weight: bold;
}
:deep(.group-header-row:hover td) {
  background-color: #e8f5d6 !important;
}
</style>
