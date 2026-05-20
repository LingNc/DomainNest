<!-- web/src/views/MyPermissions.vue -->
<template>
  <div>
    <div class="page-header">
      <h2>{{ $t('permissions.title') }}</h2>
      <p class="subtitle">{{ $t('permissions.subtitle') }}</p>
    </div>

    <el-card>
      <el-table :data="permissions" stripe v-loading="loading" style="width:100%">
        <el-table-column prop="domain_node.full_domain" :label="$t('common.domain')" min-width="200">
          <template #default="{ row }">
            <el-button link type="primary" @click="$router.push(`/domains/${row.domain_node_id}`)">
              {{ row.domain_node?.full_domain || 'Domain #' + row.domain_node_id }}
            </el-button>
          </template>
        </el-table-column>
        <el-table-column prop="permission_level" :label="$t('domainDetail.permLevel')" width="120">
          <template #default="{ row }">
            <el-tag :type="permTagType(row.permission_level)" size="small">
              {{ permLabel(row.permission_level) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="host_prefix" :label="$t('domainDetail.hostPrefix')" min-width="120">
          <template #default="{ row }">
            <span v-if="row.host_prefix">{{ row.host_prefix }}</span>
            <span v-else style="color:#909399">{{ $t('common.unlimited') }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="max_depth" :label="$t('domainDetail.maxDepth')" width="100">
          <template #default="{ row }">
            <span v-if="row.max_depth">{{ row.max_depth }}</span>
            <span v-else style="color:#909399">{{ $t('common.unlimited') }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="allowed_types" :label="$t('domainDetail.recordTypes')" min-width="160">
          <template #default="{ row }">
            <span v-if="row.allowed_types">{{ row.allowed_types }}</span>
            <span v-else style="color:#909399">{{ $t('permissions.all') }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="allowed_ips" :label="$t('domainDetail.ipRestriction')" min-width="160">
          <template #default="{ row }">
            <span v-if="row.allowed_ips">{{ row.allowed_ips }}</span>
            <span v-else style="color:#909399">{{ $t('common.unlimited') }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('permissions.grantor')" min-width="120">
          <template #default="{ row }">
            <div style="display:flex;align-items:center;gap:6px">
              <el-avatar v-if="row.creator?.avatar" :src="row.creator.avatar" :size="24" />
              <el-avatar v-else :size="24">{{ (row.creator?.username || '?')[0]?.toUpperCase() }}</el-avatar>
              <span>{{ row.creator?.username || '—' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="$t('permissions.grantTime')" width="170" />
      </el-table>
      <el-empty v-if="!loading && permissions.length === 0" :description="$t('permissions.noPermissions')" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getMyPermissions } from '../api/permission'

const { t } = useI18n()

const permissions = ref([])
const loading = ref(false)

const permTagType = (level) => ({ read: 'info', write: 'success', admin: 'warning' }[level] || 'info')
const permLabel = (level) => ({ read: t('common.read'), write: t('common.write'), admin: t('common.admin') }[level] || level)

const loadData = async () => {
  loading.value = true
  try {
    const res = await getMyPermissions()
    permissions.value = res.data || []
  } catch {
    // silently ignore — shows empty state
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
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
</style>
