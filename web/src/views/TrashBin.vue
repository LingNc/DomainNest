<template>
  <div>
    <div class="page-header">
      <div>
        <h2>{{ $t('trash.title') }}</h2>
      </div>
      <el-button type="danger" size="small" @click="handleEmptyTrash" :disabled="total === 0">
        <el-icon><component :is="'Delete'" /></el-icon>
        {{ $t('trash.empty') }}
      </el-button>
    </div>

    <el-card>
      <!-- Filter bar -->
      <div class="filter-bar">
        <el-input v-model="filters.host" :placeholder="$t('trash.host')" clearable size="small" style="width:140px" @clear="loadTrash" @keyup.enter="loadTrash" />
        <el-select v-model="filters.record_type" :placeholder="$t('trash.type')" clearable size="small" style="width:120px" @change="loadTrash">
          <el-option v-for="t in recordTypes" :key="t.value" :label="t.label" :value="t.value" />
        </el-select>
        <el-select v-model="filters.domain_id" :placeholder="$t('trash.domain')" clearable filterable size="small" style="width:180px" @change="loadTrash">
          <el-option v-for="d in domains" :key="d.id" :label="d.full_domain" :value="d.id" />
        </el-select>
        <el-date-picker v-model="filters.dateRange" type="daterange" :range-separator="'-'" :start-placeholder="$t('trash.trashedAt')" :end-placeholder="$t('trash.trashedAt')" size="small" style="width:230px" value-format="YYYY-MM-DD" @change="loadTrash" />
        <div class="filter-actions">
          <el-button size="small" type="primary" @click="loadTrash">{{ $t('common.search') }}</el-button>
          <el-button size="small" @click="resetFilters">{{ $t('common.reset') }}</el-button>
        </div>
      </div>

      <!-- Batch action bar -->
      <div class="batch-bar" v-if="selectedIds.length > 0">
        <span>{{ $t('domainDetail.selectedCount', { count: selectedIds.length }) }}</span>
        <el-button size="small" type="success" @click="handleBatchRestore">{{ $t('trash.batchRestore') }}</el-button>
        <el-button size="small" type="danger" @click="handleBatchDelete">{{ $t('trash.batchDelete') }}</el-button>
      </div>

      <!-- Table -->
      <el-table :data="items" stripe v-loading="loading" style="width:100%" @selection-change="handleSelectionChange">
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

      <el-empty v-if="!loading && items.length === 0" :description="$t('trash.noRecords')" />

      <!-- Pagination -->
      <el-pagination v-if="total > pageSize" :current-page="page" :page-size="pageSize" :total="total"
        layout="total, sizes, prev, pager, next" :page-sizes="[20, 50, 100]"
        @current-change="handlePageChange" @size-change="handleSizeChange" style="margin-top:16px" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { listTrash, restoreRecord, permanentDelete, emptyTrash, batchRestore } from '../api/trash'
import { getDomains } from '../api/domain'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t } = useI18n()

const recordTypes = [
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

const items = ref([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const selectedIds = ref([])
const domains = ref([])

const filters = reactive({ host: '', record_type: '', domain_id: '', dateRange: null })

const loadTrash = async () => {
  loading.value = true
  try {
    const params = { page: page.value, page_size: pageSize.value }
    if (filters.host) params.host = filters.host
    if (filters.record_type) params.record_type = filters.record_type
    if (filters.domain_id) params.domain_id = filters.domain_id
    if (filters.dateRange && filters.dateRange.length === 2) {
      params.trashed_after = filters.dateRange[0]
      params.trashed_before = filters.dateRange[1]
    }
    const res = await listTrash(params)
    items.value = res.data.items || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

const loadDomains = async () => {
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
    domains.value = flatten(res.data || [])
  } catch { /* ignore */ }
}

const handleSelectionChange = (rows) => {
  selectedIds.value = rows.map(r => r.id)
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
  await batchRestore(selectedIds.value)
  ElMessage.success(t('trash.batchRestoreSuccess', { count: selectedIds.value.length }))
  selectedIds.value = []
  loadTrash()
}

const handleBatchDelete = async () => {
  try {
    await ElMessageBox.confirm(
      t('trash.permanentDeleteConfirm'),
      t('trash.batchDelete'),
      { type: 'warning', confirmButtonText: t('common.confirmDelete') }
    )
    for (const id of selectedIds.value) {
      await permanentDelete(id)
    }
    ElMessage.success(t('trash.batchDeleteSuccess', { count: selectedIds.value.length }))
    selectedIds.value = []
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

const resetFilters = () => {
  filters.host = ''
  filters.record_type = ''
  filters.domain_id = ''
  filters.dateRange = null
  page.value = 1
  loadTrash()
}

const handlePageChange = (p) => {
  page.value = p
  loadTrash()
}

const handleSizeChange = (s) => {
  pageSize.value = s
  page.value = 1
  loadTrash()
}

onMounted(() => {
  loadTrash()
  loadDomains()
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
.batch-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #f0f9ff;
  border-radius: 4px;
}
</style>
