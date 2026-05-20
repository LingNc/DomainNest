<!-- web/src/views/RAMTokens.vue -->
<template>
  <div>
    <div class="page-header">
      <div>
        <h2>API Tokens</h2>
        <p class="subtitle">{{ $t('ramTokens.subtitle') }}</p>
      </div>
      <el-button type="primary" size="small" @click="openCreate">
        <el-icon><component :is="'Plus'" /></el-icon>
        {{ $t('ramTokens.createToken') }}
      </el-button>
    </div>

    <el-card>
      <el-table :data="tokens" stripe v-loading="loading" style="width:100%">
        <el-table-column prop="name" :label="$t('ramTokens.name')" min-width="120" />
        <el-table-column label="Token" min-width="220">
          <template #default="{ row }">
            <div class="token-cell">
              <code class="token-masked">{{ maskToken(row.token) }}</code>
              <el-button link type="primary" size="small" @click="copyToken(row.token)">{{ $t('common.copy') }}</el-button>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="$t('ramTokens.enabled')" width="70">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" size="small" @change="(val) => handleToggle(row.id, val)" />
          </template>
        </el-table-column>
        <el-table-column :label="$t('ramTokens.domainRestriction')" min-width="120">
          <template #default="{ row }">
            <span v-if="row.allowed_domains">{{ row.allowed_domains }}</span>
            <span v-else style="color:#909399">{{ $t('ramTokens.all') }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('ramTokens.typeRestriction')" min-width="120">
          <template #default="{ row }">
            <span v-if="row.allowed_types">{{ row.allowed_types }}</span>
            <span v-else style="color:#909399">{{ $t('ramTokens.all') }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('ramTokens.ipRestriction')" min-width="120">
          <template #default="{ row }">
            <span v-if="row.allowed_ips">{{ row.allowed_ips }}</span>
            <span v-else style="color:#909399">{{ $t('ramTokens.unlimited') }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="usage_count" :label="$t('ramTokens.usageCount')" width="90" />
        <el-table-column prop="last_used_at" :label="$t('ramTokens.lastUsed')" width="170">
          <template #default="{ row }">{{ row.last_used_at || '-' }}</template>
        </el-table-column>
        <el-table-column :label="$t('ramTokens.action')" width="200" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="openEdit(row)">{{ $t('ramTokens.edit') }}</el-button>
            <el-button link type="warning" size="small" @click="handleReset(row.id)">{{ $t('ramTokens.reset') }}</el-button>
            <el-button link type="danger" size="small" @click="handleDelete(row.id)">{{ $t('ramTokens.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && tokens.length === 0" :description="$t('ramTokens.noTokens')" />
    </el-card>

    <!-- 创建 Token -->
    <el-dialog v-model="showCreate" :title="$t('ramTokens.createTitle')" width="520px">
      <el-form :model="form" label-width="80px">
        <el-form-item :label="$t('ramTokens.name')">
          <el-input v-model="form.name" :placeholder="$t('ramTokens.namePlaceholder')" />
        </el-form-item>
        <el-form-item :label="$t('ramTokens.domainRestriction')">
          <el-input v-model="form.allowed_domains_text" type="textarea" :rows="2" :placeholder="$t('ramTokens.domainPlaceholder')" />
        </el-form-item>
        <el-form-item :label="$t('ramTokens.typeRestriction')">
          <el-checkbox-group v-model="form.allowed_types">
            <el-checkbox v-for="t in recordTypes" :key="t" :label="t">{{ t }}</el-checkbox>
          </el-checkbox-group>
          <div style="color:#909399;font-size:12px;margin-top:4px">{{ $t('ramTokens.typeHint') }}</div>
        </el-form-item>
        <el-form-item :label="$t('ramTokens.ipRestriction')">
          <el-input v-model="form.allowed_ips_text" type="textarea" :rows="2" :placeholder="$t('ramTokens.ipPlaceholder')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate = false">{{ $t('ramTokens.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreate">{{ $t('ramTokens.create') }}</el-button>
      </template>
    </el-dialog>

    <!-- 编辑 Token -->
    <el-dialog v-model="showEdit" :title="$t('ramTokens.editTitle')" width="520px">
      <el-form :model="editForm" label-width="80px">
        <el-form-item :label="$t('ramTokens.name')">
          <el-input v-model="editForm.name" />
        </el-form-item>
        <el-form-item :label="$t('ramTokens.domainRestriction')">
          <el-input v-model="editForm.allowed_domains_text" type="textarea" :rows="2" :placeholder="$t('ramTokens.domainPlaceholder')" />
        </el-form-item>
        <el-form-item :label="$t('ramTokens.typeRestriction')">
          <el-checkbox-group v-model="editForm.allowed_types">
            <el-checkbox v-for="t in recordTypes" :key="t" :label="t">{{ t }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item :label="$t('ramTokens.ipRestriction')">
          <el-input v-model="editForm.allowed_ips_text" type="textarea" :rows="2" :placeholder="$t('ramTokens.ipPlaceholder')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEdit = false">{{ $t('ramTokens.cancel') }}</el-button>
        <el-button type="primary" @click="handleUpdate">{{ $t('ramTokens.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { listRAMTokens, createRAMToken, updateRAMToken, resetRAMToken, deleteRAMToken } from '../api/ramToken'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t } = useI18n()

const recordTypes = ['A', 'AAAA', 'CNAME', 'ALIAS', 'MX', 'TXT', 'CAA', 'NS', 'SRV']

const tokens = ref([])
const loading = ref(false)
const showCreate = ref(false)
const showEdit = ref(false)
const editTargetId = ref(null)

const form = ref({ name: '', allowed_domains_text: '', allowed_types: [], allowed_ips_text: '' })
const editForm = ref({ name: '', allowed_domains_text: '', allowed_types: [], allowed_ips_text: '' })

const maskToken = (token) => {
  if (!token || token.length < 10) return token
  return token.slice(0, 6) + '****' + token.slice(-4)
}

const copyToken = (token) => {
  navigator.clipboard.writeText(token)
  ElMessage.success(t('ramTokens.copied'))
}

const loadTokens = async () => {
  loading.value = true
  try {
    const res = await listRAMTokens()
    tokens.value = res.data || []
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  form.value = { name: '', allowed_domains_text: '', allowed_types: [], allowed_ips_text: '' }
  showCreate.value = true
}

const parseDomainsText = (text) => {
  if (!text.trim()) return []
  return text.trim().split('\n').map(s => parseInt(s.trim())).filter(n => !isNaN(n))
}

const parseIPsText = (text) => {
  if (!text.trim()) return []
  return text.trim().split('\n').map(s => s.trim()).filter(Boolean)
}

const handleCreate = async () => {
  const data = { name: form.value.name }
  const domains = parseDomainsText(form.value.allowed_domains_text)
  if (domains.length > 0) data.allowed_domains = domains
  if (form.value.allowed_types.length > 0) data.allowed_types = form.value.allowed_types
  const ips = parseIPsText(form.value.allowed_ips_text)
  if (ips.length > 0) data.allowed_ips = ips

  await createRAMToken(data)
  ElMessage.success(t('ramTokens.createSuccess'))
  showCreate.value = false
  loadTokens()
}

const openEdit = (row) => {
  editTargetId.value = row.id
  editForm.value = {
    name: row.name,
    allowed_domains_text: row.allowed_domains || '',
    allowed_types: row.allowed_types ? JSON.parse(row.allowed_types) : [],
    allowed_ips_text: row.allowed_ips || '',
  }
  showEdit.value = true
}

const handleUpdate = async () => {
  const data = { name: editForm.value.name }
  const domains = parseDomainsText(editForm.value.allowed_domains_text)
  data.allowed_domains = domains.length > 0 ? domains : []
  data.allowed_types = editForm.value.allowed_types.length > 0 ? editForm.value.allowed_types : []
  const ips = parseIPsText(editForm.value.allowed_ips_text)
  data.allowed_ips = ips.length > 0 ? ips : []

  await updateRAMToken(editTargetId.value, data)
  ElMessage.success(t('ramTokens.updateSuccess'))
  showEdit.value = false
  loadTokens()
}

const handleToggle = async (id, enabled) => {
  await updateRAMToken(id, { enabled })
  ElMessage.success(enabled ? t('ramTokens.enabledSuccess') : t('ramTokens.disabledSuccess'))
}

const handleReset = async (id) => {
  await ElMessageBox.confirm(t('ramTokens.resetConfirm'), t('ramTokens.resetTitle'), { type: 'warning' })
  await resetRAMToken(id)
  ElMessage.success(t('ramTokens.resetSuccess'))
  loadTokens()
}

const handleDelete = async (id) => {
  await ElMessageBox.confirm(t('ramTokens.deleteConfirm'), t('ramTokens.deleteTitle'), { type: 'error' })
  await deleteRAMToken(id)
  ElMessage.success(t('ramTokens.deleteSuccess'))
  loadTokens()
}

onMounted(loadTokens)
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
.token-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}
.token-masked {
  font-family: monospace;
  font-size: 13px;
  background: #f5f5f5;
  padding: 2px 6px;
  border-radius: 3px;
}
</style>
