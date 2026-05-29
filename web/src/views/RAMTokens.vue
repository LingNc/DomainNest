<!-- web/src/views/RAMTokens.vue -->
<template>
  <div>
    <div class="page-header">
      <div>
        <h2>{{ $t('sidebar.apiTokens') }}</h2>
        <p class="subtitle">{{ $t('ramTokens.subtitle') }}</p>
      </div>
      <el-button type="primary" size="small" @click="openCreate">
        <el-icon><component :is="'Plus'" /></el-icon>
        {{ $t('ramTokens.createToken') }}
      </el-button>
    </div>

    <el-tabs>
      <el-tab-pane :label="$t('ramTokens.managementTab')">
        <el-card>
          <div style="display: flex; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; align-items: center;">
            <el-input v-model="tokenSearch" :placeholder="$t('ramTokens.searchPlaceholder')" clearable style="width: 220px" />
            <el-select v-model="tokenStatusFilter" :placeholder="$t('ramTokens.statusFilter')" clearable style="width: 120px">
              <el-option :label="$t('common.enabled')" value="enabled" />
              <el-option :label="$t('common.disabled')" value="disabled" />
            </el-select>
          </div>
          <el-table :data="filteredTokens" stripe v-loading="loading" style="width:100%" @sort-change="handleSortChange">
            <el-table-column prop="id" :label="$t('ramTokens.id')" width="80" sortable="custom" />
            <el-table-column prop="name" :label="$t('ramTokens.name')" min-width="120" sortable="custom" />
            <el-table-column prop="access_key_id" :label="$t('ramTokens.accessKeyID')" min-width="180">
              <template #default="{ row }">
                <span v-if="row.access_key_id" style="font-family: monospace; font-size: 12px">{{ row.access_key_id }}</span>
                <span v-else style="color: #909399">-</span>
              </template>
            </el-table-column>
            <el-table-column :label="$t('ramTokens.tokenValue')" min-width="220">
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
            <el-table-column prop="created_at" :label="$t('common.createdAt')" width="170" sortable="custom" />
            <el-table-column :label="$t('ramTokens.action')" width="200" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="openEdit(row)">{{ $t('ramTokens.edit') }}</el-button>
                <el-button link type="warning" size="small" @click="handleReset(row.id)">{{ $t('ramTokens.reset') }}</el-button>
                <el-button link type="danger" size="small" @click="handleDelete(row.id)">{{ $t('ramTokens.delete') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!loading && filteredTokens.length === 0" :description="$t('ramTokens.noTokens')" />
        </el-card>
      </el-tab-pane>
      <el-tab-pane :label="$t('ramTokens.helpTab')">
        <!-- 使用帮助 -->
        <el-card style="margin-top: 20px">
          <template #header>
            <span>{{ $t('ramTokens.usageHelp') }}</span>
          </template>

          <el-table :data="providerEndpoints" stripe border size="small" style="width: 100%">
            <el-table-column prop="name" :label="$t('ramTokens.providerName')" min-width="120" />
            <el-table-column :label="$t('ramTokens.endpointUrl')" min-width="280">
              <template #default="{ row }">
                <code>{{ row.url }}</code>
                <el-button link type="primary" size="small" style="margin-left: 8px" @click="copyText(row.url)">{{ $t('common.copy') }}</el-button>
              </template>
            </el-table-column>
          </el-table>

          <el-divider />

          <h4>{{ $t('ramTokens.withHttpreq') }}</h4>
          <p>{{ $t('ramTokens.httpreqDesc') }}</p>
          <el-alert type="info" :closable="false" show-icon style="margin: 12px 0">
            {{ $t('ramTokens.httpreq1PanelV1Note') }}
          </el-alert>
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item :label="$t('ramTokens.endpointLabel')">
              <code>{{ httpreqEndpoint }}</code>
              <el-button link type="primary" size="small" style="margin-left: 8px" @click="copyText(httpreqEndpoint)">{{ $t('common.copy') }}</el-button>
            </el-descriptions-item>
          </el-descriptions>
          <el-button type="primary" size="small" style="margin-top: 12px" @click="download1PanelPatch">
            <el-icon><Download /></el-icon>
            {{ $t('ramTokens.download1PanelPatch') }}
          </el-button>

          <el-divider />

          <h4>{{ $t('ramTokens.withTechnitium') }}</h4>
          <p>{{ $t('ramTokens.technitiumDesc') }}</p>
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item :label="$t('ramTokens.baseUrlLabel')">
              <code>{{ technitiumEndpoint }}</code>
              <el-button link type="primary" size="small" style="margin-left: 8px" @click="copyText(technitiumEndpoint)">{{ $t('common.copy') }}</el-button>
            </el-descriptions-item>
            <el-descriptions-item :label="$t('ramTokens.tokenLabel')">
              <div v-if="firstEnabledToken" style="display: flex; align-items: center; gap: 8px;">
                <code>{{ firstEnabledToken.token }}</code>
                <el-button link type="primary" size="small" @click="copyToken(firstEnabledToken.token)">{{ $t('common.copy') }}</el-button>
              </div>
              <div v-else style="display: flex; align-items: center; gap: 8px;">
                <span style="color: #909399">{{ $t('ramTokens.noTokenForTechnitium') }}</span>
                <el-button link type="primary" size="small" @click="openCreate">{{ $t('ramTokens.createToken') }}</el-button>
              </div>
            </el-descriptions-item>
          </el-descriptions>

          <el-divider />

          <h4>{{ $t('ramTokens.withAcmeDNS') }}</h4>
          <p>{{ $t('ramTokens.acmeDNSDesc') }}</p>
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item :label="$t('ramTokens.apiBaseLabel')">
              <code>{{ acmednsEndpoint }}</code>
              <el-button link type="primary" size="small" style="margin-left: 8px" @click="copyText(acmednsEndpoint)">{{ $t('common.copy') }}</el-button>
            </el-descriptions-item>
            <el-descriptions-item :label="$t('ramTokens.baseUrlLabel')">
              <code>{{ acmednsEndpoint }}</code>
              <el-button link type="primary" size="small" style="margin-left: 8px" @click="copyText(acmednsEndpoint)">{{ $t('common.copy') }}</el-button>
            </el-descriptions-item>
          </el-descriptions>
          <el-alert type="info" :closable="false" show-icon style="margin-top: 12px">
            {{ $t('ramTokens.acmeDNSAuthHint') }}
          </el-alert>

          <el-divider />

          <h4>{{ $t('ramTokens.with1Panel') }}</h4>
          <el-alert type="info" :closable="false" show-icon style="margin-bottom: 12px">
            {{ $t('ramTokens.aliyunHostsHint') }}
          </el-alert>
          <ol style="line-height: 2; padding-left: 20px">
            <li>{{ $t('ramTokens.step1CreateToken') }}</li>
            <li>{{ $t('ramTokens.step2CopyCreds') }}</li>
            <li v-html="$t('ramTokens.step3HostsFile')"></li>
            <li>{{ $t('ramTokens.step4SelectAliyun') }}</li>
          </ol>
          <el-alert type="warning" :closable="false" show-icon>
            {{ $t('ramTokens.hostsWarning') }}
          </el-alert>

          <el-divider />

          <h4>{{ $t('ramTokens.withDdnsGo') }}</h4>
          <p>{{ $t('ramTokens.ddnsGoCallbackHint') }}</p>
          <el-button type="primary" size="small" @click="$router.push('/settings')">{{ $t('ramTokens.goToSettings') }}</el-button>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <!-- 创建 Token -->
    <el-dialog v-model="showCreate" :title="$t('ramTokens.createTitle')" width="520px" destroy-on-close>
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
    <el-dialog v-model="showEdit" :title="$t('ramTokens.editTitle')" width="520px" destroy-on-close>
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

    <!-- 创建成功，显示密钥 -->
    <el-dialog v-model="showCreatedSecret" :title="$t('ramTokens.credentialTitle')" width="560px" :close-on-click-modal="false">
      <el-alert type="warning" :closable="false" show-icon style="margin-bottom: 16px">
        {{ $t('ramTokens.secretShownOnce') }}
      </el-alert>
      <el-card shadow="never" style="background: #fafafa;">
        <template #header>
          <div style="display: flex; justify-content: space-between; align-items: center;">
            <span style="font-weight: 600;">{{ createdToken.name }}</span>
            <el-button type="primary" size="small" @click="copyAllCredentials">{{ $t('ramTokens.copyAll') }}</el-button>
          </div>
        </template>
        <div style="margin-bottom: 12px;">
          <div style="font-size: 12px; color: #909399; margin-bottom: 4px;">AccessKeyID</div>
          <div style="display: flex; align-items: center; gap: 8px;">
            <code style="flex: 1; word-break: break-all; font-size: 13px;">{{ createdToken.access_key_id }}</code>
            <el-button link type="primary" size="small" @click="copyText(createdToken.access_key_id)">{{ $t('common.copy') }}</el-button>
          </div>
        </div>
        <div>
          <div style="font-size: 12px; color: #909399; margin-bottom: 4px;">AccessKeySecret</div>
          <div style="display: flex; align-items: center; gap: 8px;">
            <code style="flex: 1; word-break: break-all; font-size: 13px;">{{ createdToken.access_key_secret }}</code>
            <el-button link type="primary" size="small" @click="copyText(createdToken.access_key_secret)">{{ $t('common.copy') }}</el-button>
          </div>
        </div>
        <el-divider style="margin: 12px 0;" />
        <div style="display: flex; justify-content: flex-end;">
          <el-button size="small" @click="downloadCSV">{{ $t('ramTokens.downloadCSV') }}</el-button>
        </div>
      </el-card>
      <template #footer>
        <el-button type="primary" @click="showCreatedSecret = false">{{ $t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { listRAMTokens, createRAMToken, updateRAMToken, resetRAMToken, deleteRAMToken } from '../api/ramToken'
import { useAuthStore } from '../stores/auth'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t } = useI18n()
const auth = useAuthStore()

const recordTypes = ['A', 'AAAA', 'CNAME', 'ALIAS', 'MX', 'TXT', 'CAA', 'NS', 'SRV']

const tokens = ref([])
const loading = ref(false)
const showCreate = ref(false)
const showEdit = ref(false)
const showCreatedSecret = ref(false)
const createdToken = ref({})
const editTargetId = ref(null)
const tokenSearch = ref('')
const tokenStatusFilter = ref('')
const sortProp = ref('')
const sortOrder = ref('')

const handleSortChange = ({ prop, order }) => {
  sortProp.value = prop || ''
  sortOrder.value = order || ''
}

const filteredTokens = computed(() => {
  let result = tokens.value
  const q = tokenSearch.value.toLowerCase()
  if (q) {
    result = result.filter(t =>
      (t.name || '').toLowerCase().includes(q) ||
      (t.access_key_id || '').toLowerCase().includes(q)
    )
  }
  if (tokenStatusFilter.value === 'enabled') {
    result = result.filter(t => t.enabled)
  } else if (tokenStatusFilter.value === 'disabled') {
    result = result.filter(t => !t.enabled)
  }
  if (sortProp.value && sortOrder.value) {
    const prop = sortProp.value
    const dir = sortOrder.value === 'ascending' ? 1 : -1
    result = [...result].sort((a, b) => {
      const va = a[prop] ?? ''
      const vb = b[prop] ?? ''
      if (va < vb) return -dir
      if (va > vb) return dir
      return 0
    })
  }
  return result
})

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

const copyText = (text) => {
  navigator.clipboard.writeText(text)
  ElMessage.success(t('common.copied'))
}

const copyAllCredentials = () => {
  const token = createdToken.value
  const text = `AccessKeyID:\n${token.access_key_id}\nAccessKeySecret:\n${token.access_key_secret}`
  navigator.clipboard.writeText(text)
  ElMessage.success(t('ramTokens.copiedAll'))
}

const download1PanelPatch = () => {
  const lang = localStorage.getItem('locale') || 'zh-CN'
  const link = document.createElement('a')
  link.href = `/api/v1/scripts/1panel-v1-patch.zip?lang=${lang}`
  link.download = '1panel-v1-httpreq-patch.zip'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

const downloadCSV = () => {
  const token = createdToken.value
  const escapeCsv = (val) => {
    const s = String(val || '')
    if (s.includes(',') || s.includes('"') || s.includes('\n')) return '"' + s.replace(/"/g, '""') + '"'
    return s
  }
  const headers = ['name', 'access_key_id', 'access_key_secret', 'endpoint', 'username', 'created_at']
  const values = [
    token.name || '',
    token.access_key_id || '',
    token.access_key_secret || '',
    window.location.origin + '/alidns',
    auth.user?.username || '',
    token.created_at || ''
  ]
  const csv = headers.join(',') + '\n' + values.map(escapeCsv).join(',') + '\n'
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = `credentials-${token.name || 'token'}.csv`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

const providerEndpoints = computed(() => {
  const base = window.location.origin
  return [
    { name: 'Aliyun DNS', url: base + '/alidns' },
    { name: 'Cloudflare', url: base + '/cf/v1' },
    { name: 'TencentCloud', url: base + '/tcdns' },
    { name: 'Technitium', url: base + '/technitium' },
    { name: 'httpreq', url: base + '/httpreq' },
    { name: 'AcmeDNS', url: base + '/acmedns' },
  ]
})

const technitiumEndpoint = computed(() => {
  return window.location.origin + '/technitium'
})

const firstEnabledToken = computed(() => {
  return tokens.value.find(t => t.enabled) || null
})

const acmednsEndpoint = computed(() => {
  return window.location.origin + '/acmedns'
})

const httpreqEndpoint = computed(() => {
  return window.location.origin + '/httpreq'
})

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

  const res = await createRAMToken(data)
  createdToken.value = res.data || res
  showCreatedSecret.value = true
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
  const res = await resetRAMToken(id)
  createdToken.value = res.data || res
  showCreatedSecret.value = true
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
