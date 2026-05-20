<template>
  <div>
    <div class="page-header">
      <h2>{{ $t('myLogs.title') }}</h2>
      <p class="subtitle">{{ $t('myLogs.subtitle') }}</p>
    </div>
    <el-card>
      <div class="view-filter">
        <el-radio-group v-model="viewFilter" size="small" @change="handleViewChange">
          <el-radio-button value="all">{{ $t('myLogs.viewAll') }}</el-radio-button>
          <el-radio-button value="actor">{{ $t('myLogs.viewMyActions') }}</el-radio-button>
          <el-radio-button value="target">{{ $t('myLogs.viewActionsOnMe') }}</el-radio-button>
        </el-radio-group>
      </div>
      <div class="filter-bar">
        <el-input v-model="filters.q" :placeholder="$t('myLogs.searchPlaceholder')" clearable size="small" style="width:140px" />
        <el-input v-model="filters.q_exclude" :placeholder="$t('myLogs.excludePlaceholder')" clearable size="small" style="width:120px" />
        <div class="filter-group">
          <el-select v-model="filters.action" :placeholder="$t('myLogs.actionPlaceholder')" clearable filterable multiple collapse-tags collapse-tags-tooltip size="small" style="width:170px" @change="loadLogs">
            <el-option-group v-for="group in actionGroups" :key="group.label" :label="group.label">
              <el-option v-for="item in group.options" :key="item.value" :label="item.label" :value="item.value" />
            </el-option-group>
          </el-select>
          <el-tooltip :content="actionExcludeMode ? $t('myLogs.excludeMode') : $t('myLogs.includeMode')">
            <el-button size="small" :type="actionExcludeMode ? 'danger' : ''" @click="actionExcludeMode = !actionExcludeMode; loadLogs()" style="padding: 5px">
              <el-icon :size="14"><component :is="actionExcludeMode ? 'CircleClose' : 'CircleCheck'" /></el-icon>
            </el-button>
          </el-tooltip>
        </div>
        <div class="filter-group">
          <el-select v-model="filters.target_type" :placeholder="$t('myLogs.targetTypePlaceholder')" clearable multiple collapse-tags collapse-tags-tooltip size="small" style="width:170px" @change="loadLogs">
            <el-option v-for="item in targetTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
          <el-tooltip :content="targetTypeExcludeMode ? $t('myLogs.excludeMode') : $t('myLogs.includeMode')">
            <el-button size="small" :type="targetTypeExcludeMode ? 'danger' : ''" @click="targetTypeExcludeMode = !targetTypeExcludeMode; loadLogs()" style="padding: 5px">
              <el-icon :size="14"><component :is="targetTypeExcludeMode ? 'CircleClose' : 'CircleCheck'" /></el-icon>
            </el-button>
          </el-tooltip>
        </div>
        <el-date-picker v-model="filters.dateRange" type="daterange" :range-separator="$t('myLogs.dateRangeSeparator')" :start-placeholder="$t('myLogs.startDatePlaceholder')" :end-placeholder="$t('myLogs.endDatePlaceholder')" size="small" style="width:230px" value-format="YYYY-MM-DD" />
        <div class="filter-actions">
          <el-button size="small" type="primary" @click="loadLogs">{{ $t('common.search') }}</el-button>
          <el-button size="small" @click="resetFilters">{{ $t('common.reset') }}</el-button>
        </div>
      </div>
      <el-table :data="logs" stripe v-loading="loading" style="width:100%">
        <el-table-column :label="$t('myLogs.actorUser')" min-width="120">
          <template #default="{ row }">
            <template v-if="viewFilter === 'target' && row.user">
              <div style="display:flex;align-items:center;gap:6px">
                <el-avatar v-if="getSrc(row.user_id, row.user?.avatar)" :src="getSrc(row.user_id, row.user?.avatar)" :size="24" />
                <el-avatar v-else :size="24">{{ firstLetter(row.user?.username) }}</el-avatar>
                <span>{{ row.user.username }}</span>
              </div>
            </template>
            <span v-else style="color:#909399">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="action" :label="$t('common.action')" min-width="120" />
        <el-table-column prop="target_type" :label="$t('common.targetType')" min-width="100" />
        <el-table-column prop="ip_address" :label="$t('common.ipAddress')" width="130" />
        <el-table-column :label="$t('myLogs.targetUser')" min-width="120">
          <template #default="{ row }">
            <template v-if="row.target_user">
              <div style="display:flex;align-items:center;gap:6px">
                <el-avatar v-if="getSrc(row.target_user_id, row.target_user?.avatar)" :src="getSrc(row.target_user_id, row.target_user?.avatar)" :size="24" />
                <el-avatar v-else :size="24">{{ firstLetter(row.target_user?.username) }}</el-avatar>
                <span>{{ row.target_user.username }}</span>
              </div>
            </template>
            <span v-else style="color:#909399">-</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('common.detail')" min-width="200">
          <template #default="{ row }">
            <template v-if="row.detail && row.detail !== 'null'">
              <el-tag v-for="(val, key) in parseDetail(row.detail)" :key="key" size="small" style="margin:2px" type="info">
                {{ formatDetailKey(key) }}: {{ formatDetailValue(val) }}
              </el-tag>
            </template>
            <span v-else style="color:#909399">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="$t('common.time')" width="170" />
      </el-table>
      <el-empty v-if="!loading && logs.length === 0" :description="$t('myLogs.noLogs')" />
      <el-pagination v-if="total > pageSize" :current-page="page" :page-size="pageSize" :total="total"
        layout="total, sizes, prev, pager, next" :page-sizes="[20, 50, 100]"
        @current-change="handlePageChange" @size-change="handleSizeChange" style="margin-top:16px" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getMyLogs } from '../api/auth'
import { actionGroups, targetTypeOptions } from '../constants/operationLogs'
import { useAvatar } from '../composables/useAvatar'

const { t } = useI18n()
const { getSrc, firstLetter } = useAvatar()

const logs = ref([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const parseDetail = (json) => {
  try { return JSON.parse(json) } catch { return {} }
}

const formatDetailKey = (key) => {
  const labels = {
    username: t('common.username'),
    full_domain: t('common.domain'),
    host: t('myLogs.hostRecord'),
    type: t('myLogs.recordType'),
    value: t('myLogs.recordValue'),
    amount: t('myLogs.amount'),
    enabled: t('common.status'),
    ids: t('myLogs.recordId'),
    target_user_id: t('myLogs.targetUser'),
    invited_by: t('myLogs.inviter'),
    provider_name: t('myLogs.providerName'),
  }
  return labels[key] || key
}

const formatDetailValue = (val) => {
  if (Array.isArray(val)) return val.join(', ')
  if (typeof val === 'boolean') return val ? t('common.enabled') : t('common.disabled')
  return String(val)
}

const filters = reactive({ action: [], target_type: [], q: '', q_exclude: '', dateRange: null })
const actionExcludeMode = ref(false)
const targetTypeExcludeMode = ref(false)
const viewFilter = ref('all')

const loadLogs = async () => {
  loading.value = true
  try {
    const params = { page: page.value, page_size: pageSize.value }
    if (viewFilter.value !== 'all') params.view = viewFilter.value
    if (filters.action.length > 0) {
      if (actionExcludeMode.value) {
        params.action_exclude = filters.action.join(',')
      } else {
        params.action = filters.action.join(',')
      }
    }
    if (filters.target_type.length > 0) {
      if (targetTypeExcludeMode.value) {
        params.target_type_exclude = filters.target_type.join(',')
      } else {
        params.target_type = filters.target_type.join(',')
      }
    }
    if (filters.q) params.q = filters.q
    if (filters.q_exclude) params.q_exclude = filters.q_exclude
    if (filters.dateRange && filters.dateRange.length === 2) {
      params.start_time = filters.dateRange[0]
      params.end_time = filters.dateRange[1]
    }
    const res = await getMyLogs(params)
    logs.value = res.data.items || []
    total.value = res.data.total || 0
  } finally {
    loading.value = false
  }
}

const handleViewChange = () => {
  page.value = 1
  loadLogs()
}

const resetFilters = () => {
  filters.action = []
  filters.target_type = []
  filters.q = ''
  filters.q_exclude = ''
  filters.dateRange = null
  actionExcludeMode.value = false
  targetTypeExcludeMode.value = false
  viewFilter.value = 'all'
  page.value = 1
  loadLogs()
}

const handlePageChange = (p) => {
  page.value = p
  loadLogs()
}

const handleSizeChange = (s) => {
  pageSize.value = s
  page.value = 1
  loadLogs()
}

onMounted(loadLogs)
</script>

<style scoped>
.page-header {
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
.filter-group {
  display: flex;
  align-items: center;
  gap: 2px;
}
.view-filter {
  margin-bottom: 12px;
}
</style>
