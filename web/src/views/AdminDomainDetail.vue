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
                  <el-checkbox v-model="includeTrashed" @change="loadRecords">{{ $t('admin.includeTrashed') }}</el-checkbox>
                </div>
              </template>
              <el-table :data="records" style="width:100%" size="small" v-loading="recordsLoading" empty-text="">
                <el-table-column :label="$t('admin.host')" prop="host" min-width="100" />
                <el-table-column :label="$t('admin.type')" prop="record_type" min-width="80">
                  <template #default="{ row }">
                    <el-tag size="small" type="primary">{{ row.record_type }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="$t('admin.value')" prop="value" min-width="200" show-overflow-tooltip />
                <el-table-column :label="$t('admin.ttl')" prop="ttl" min-width="70" />
                <el-table-column :label="$t('common.status')" min-width="80">
                  <template #default="{ row }">
                    <el-switch
                      :model-value="row.enabled"
                      size="small"
                      @change="(val) => handleToggleRecord(row, val)"
                    />
                  </template>
                </el-table-column>
                <el-table-column :label="$t('admin.source')" min-width="80">
                  <template #default="{ row }">
                    <el-tag :type="row.source === 'provider' ? 'warning' : 'info'" size="small">
                      {{ row.source === 'provider' ? $t('admin.recordSourceProvider') : $t('admin.recordSourcePlatform') }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="$t('admin.syncStatus')" min-width="90">
                  <template #default="{ row }">
                    <el-tag
                      :type="row.sync_status === 'synced' ? 'success' : row.sync_status === 'failed' ? 'danger' : 'info'"
                      size="small"
                    >
                      {{ syncStatusLabel(row.sync_status) }}
                    </el-tag>
                  </template>
                </el-table-column>
                <el-table-column :label="$t('common.actions')" min-width="80" fixed="right">
                  <template #default="{ row }">
                    <el-button link type="danger" size="small" @click="handleDeleteRecord(row)">{{ $t('common.delete') }}</el-button>
                  </template>
                </el-table-column>
              </el-table>
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

const syncStatusLabel = (status) => {
  switch (status) {
    case 'synced': return t('admin.syncSynced')
    case 'pending': return t('admin.syncPending')
    case 'failed': return t('admin.syncFailed')
    default: return t('admin.syncUnknown')
  }
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
</style>
