<template>
  <div class="providers-page">
    <el-card>
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center">
          <span>{{ $t('providers.title') }}</span>
          <el-button type="primary" size="small" @click="openAddDialog">{{ $t('providers.addProvider') }}</el-button>
        </div>
      </template>
      <el-table :data="providers" stripe v-loading="loading" style="width:100%">
        <el-table-column prop="name" :label="$t('providers.name')" min-width="120" />
        <el-table-column :label="$t('providers.type')" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ row.provider_type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="AccessKey" width="140">
          <template #default="{ row }">
            <span style="font-family:monospace;font-size:12px">****{{ row.access_key_id?.slice(-4) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('providers.status')" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small">
              {{ row.status === 'active' ? $t('providers.active') : $t('common.disabled') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" :label="$t('common.createdAt')" width="170" />
        <el-table-column :label="$t('common.actions')" width="220" fixed="right">
          <template #default="{ row }">
            <el-button size="small" text type="primary" @click="viewDomains(row)">{{ $t('providers.domainList') }}</el-button>
            <el-button size="small" text type="warning" @click="openEditDialog(row)">{{ $t('common.edit') }}</el-button>
            <el-button size="small" text type="danger" @click="handleDelete(row)">{{ $t('common.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Add Provider Dialog -->
    <el-dialog v-model="addDialogVisible" :title="$t('providers.addDialogTitle')" width="480px" destroy-on-close>
      <el-form :model="addForm" label-width="100px">
        <el-form-item :label="$t('providers.providerType')">
          <el-select v-model="addForm.provider_type" style="width:100%" filterable>
            <el-option v-for="p in providerTypeOptions" :key="p.value" :label="p.label" :value="p.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('providers.name')">
          <el-input v-model="addForm.name" :placeholder="$t('providers.namePlaceholder')" />
        </el-form-item>
        <el-form-item :label="credentialLabel.id">
          <el-input v-model="addForm.access_key_id" :placeholder="credentialPlaceholder.id" />
        </el-form-item>
        <el-form-item :label="credentialLabel.secret">
          <el-input v-model="addForm.access_key_secret" type="password" show-password :placeholder="credentialPlaceholder.secret" />
        </el-form-item>
        <el-form-item label="Endpoint">
          <el-input v-model="addForm.endpoint" :placeholder="endpointPlaceholder" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleAdd">{{ $t('providers.add') }}</el-button>
      </template>
    </el-dialog>

    <!-- Edit Provider Dialog -->
    <el-dialog v-model="editDialogVisible" :title="$t('providers.editDialogTitle')" width="480px" destroy-on-close>
      <el-form :model="editForm" label-width="100px">
        <el-form-item :label="$t('providers.name')">
          <el-input v-model="editForm.name" />
        </el-form-item>
        <el-form-item label="Endpoint">
          <el-input v-model="editForm.endpoint" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleEdit">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- Domains Dialog -->
    <el-dialog v-model="domainsDialogVisible" :title="$t('providers.domainList') + ' - ' + currentProvider?.name" width="600px" destroy-on-close>
      <el-table :data="domains" stripe v-loading="loadingDomains" style="width:100%">
        <el-table-column prop="domain_name" :label="$t('common.domain')" min-width="200" />
        <el-table-column prop="record_count" :label="$t('providers.recordCount')" width="100" />
        <el-table-column :label="$t('common.actions')" width="120">
          <template #default="{ row }">
            <el-button size="small" type="primary" :loading="claiming[row.domain_name]" @click="handleClaim(row)">
              {{ $t('providers.claim') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { listProviders, createProvider, updateProvider, deleteProvider, listProviderDomains, claimDomain } from '../api/provider'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t } = useI18n()

const showError = (msg) => {
  if (msg.length > 100) {
    ElMessageBox.alert(msg, t('common.error'), { type: 'error', confirmButtonText: t('common.confirm') })
  } else {
    ElMessage.error(msg)
  }
}

const providerTypeLabelKeys = {
  aliyun: 'providers.providerAliyun',
  aliesa: 'providers.providerAliesa',
  tencentcloud: 'providers.providerTencentcloud',
  huaweicloud: 'providers.providerHuaweicloud',
  baiducloud: 'providers.providerBaiducloud',
  trafficroute: 'providers.providerTrafficroute',
  rainyun: 'providers.providerRainyun',
}

const providerTypeOptions = computed(() =>
  providerTypes.map(p => ({
    ...p,
    label: providerTypeLabelKeys[p.value] ? t(providerTypeLabelKeys[p.value]) : p.label,
  }))
)

const providerTypes = [
  { value: 'aliyun', label: 'Alibaba Cloud DNS' },
  { value: 'aliesa', label: 'Alibaba Cloud ESA' },
  { value: 'tencentcloud', label: 'Tencent Cloud DNSPod' },
  { value: 'cloudflare', label: 'Cloudflare' },
  { value: 'huaweicloud', label: 'Huawei Cloud DNS' },
  { value: 'baiducloud', label: 'Baidu Cloud DNS' },
  { value: 'trafficroute', label: 'Volcengine DNS' },
  { value: 'godaddy', label: 'GoDaddy' },
  { value: 'namecheap', label: 'Namecheap' },
  { value: 'namesilo', label: 'NameSilo' },
  { value: 'porkbun', label: 'Porkbun' },
  { value: 'vercel', label: 'Vercel' },
  { value: 'gcore', label: 'Gcore' },
  { value: 'nsone', label: 'NS1' },
  { value: 'name_com', label: 'Name.com' },
  { value: 'cloudns', label: 'ClouDNS' },
  { value: 'dnsla', label: 'DNSLA' },
  { value: 'dynadot', label: 'Dynadot' },
  { value: 'dynv6', label: 'dynv6' },
  { value: 'spaceship', label: 'Spaceship' },
  { value: 'edgeone', label: 'EdgeOne' },
  { value: 'rainyun', label: 'Rain Yun' },
  { value: 'hipmdnsmgr', label: 'HiPM DNS' },
  { value: 'nowcn', label: 'Nowcn' },
  { value: 'eranet', label: 'Eranet' },
  { value: 'tnethk', label: 'Tnethk' },
]

const providers = ref([])
const loading = ref(false)
const submitting = ref(false)

const addDialogVisible = ref(false)
const addForm = reactive({ provider_type: 'aliyun', name: '', access_key_id: '', access_key_secret: '', endpoint: '' })

const editDialogVisible = ref(false)
const editForm = reactive({ id: null, name: '', endpoint: '' })

const domainsDialogVisible = ref(false)
const currentProvider = ref(null)
const domains = ref([])
const loadingDomains = ref(false)
const claiming = reactive({})

const credentialLabels = {
  aliyun: { id: 'AccessKey ID', secret: 'AccessKey Secret' },
  aliesa: { id: 'AccessKey ID', secret: 'AccessKey Secret' },
  tencentcloud: { id: 'SecretId', secret: 'SecretKey' },
  cloudflare: { id: 'API Token', secret: '__OPTIONAL__' },
  huaweicloud: { id: 'Access Key', secret: 'Secret Key' },
  baiducloud: { id: 'Access Key', secret: 'Secret Key' },
  trafficroute: { id: 'Access Key', secret: 'Secret Key' },
  godaddy: { id: 'API Key', secret: 'API Secret' },
  namecheap: { id: 'API User', secret: 'API Key' },
  namesilo: { id: 'API Key', secret: '__OPTIONAL__' },
  porkbun: { id: 'API Key', secret: 'Secret API Key' },
  vercel: { id: 'API Token', secret: '__OPTIONAL__' },
  gcore: { id: 'API Key', secret: '__OPTIONAL__' },
  nsone: { id: 'API Key', secret: '__OPTIONAL__' },
  name_com: { id: '___USERNAME___', secret: 'API Token' },
  cloudns: { id: 'Auth ID', secret: 'Auth Password' },
  dnsla: { id: '___USERNAME___', secret: '___PASSWORD___' },
  dynadot: { id: 'API Key', secret: '__OPTIONAL__' },
  dynv6: { id: 'API Token', secret: '__OPTIONAL__' },
  spaceship: { id: 'API Key', secret: 'API Secret' },
  edgeone: { id: 'SecretId', secret: 'SecretKey' },
  rainyun: { id: 'API Key', secret: '__OPTIONAL__' },
  hipmdnsmgr: { id: 'API Token', secret: '__OPTIONAL__' },
  nowcn: { id: 'API Key', secret: 'API Secret' },
  eranet: { id: 'API Key', secret: 'API Secret' },
  tnethk: { id: 'API Key', secret: 'API Secret' },
}

const endpointDefaults = {
  aliyun: 'alidns.aliyuncs.com',
  tencentcloud: 'dnspod.tencentcloudapi.com',
  huaweicloud: 'dns.myhuaweicloud.com',
  godaddy: 'api.godaddy.com',
  namecheap: 'api.namecheap.com',
  name_com: 'api.name.com',
}

function resolveLabel(val, fallback) {
  if (val === '___USERNAME___') return t('providers.username')
  if (val === '___PASSWORD___') return t('providers.password')
  return val || fallback
}

const credentialLabel = computed(() => ({
  id: resolveLabel(credentialLabels[addForm.provider_type]?.id, 'AccessKey ID'),
  secret: resolveLabel(credentialLabels[addForm.provider_type]?.secret, 'AccessKey Secret'),
}))
const credentialPlaceholder = computed(() => ({
  id: t('providers.input', { label: credentialLabel.value.id }),
  secret: credentialLabels[addForm.provider_type]?.secret === '__OPTIONAL__' ? t('providers.notRequired') : t('providers.input', { label: credentialLabel.value.secret }),
}))
const endpointPlaceholder = computed(() => {
  const host = endpointDefaults[addForm.provider_type]
  return host ? t('providers.optionalDefault', { host }) : t('providers.optional')
})

const loadProviders = async () => {
  loading.value = true
  try {
    const res = await listProviders()
    providers.value = res.data || []
  } finally {
    loading.value = false
  }
}

onMounted(loadProviders)

const openAddDialog = () => {
  addForm.provider_type = 'aliyun'
  addForm.name = ''
  addForm.access_key_id = ''
  addForm.access_key_secret = ''
  addForm.endpoint = ''
  addDialogVisible.value = true
}

const handleAdd = async () => {
  const needsSecret = credentialLabels[addForm.provider_type]?.secret !== '__OPTIONAL__'
  if (!addForm.name || !addForm.access_key_id || (needsSecret && !addForm.access_key_secret)) {
    ElMessage.warning(t('providers.pleaseFillFields'))
    return
  }
  submitting.value = true
  try {
    await createProvider(addForm, { skipErrorToast: true })
    ElMessage.success(t('providers.addSuccess'))
    addDialogVisible.value = false
    await loadProviders()
  } catch (e) {
    showError(e.response?.data?.message || t('providers.addFailed'))
  } finally {
    submitting.value = false
  }
}

const openEditDialog = (row) => {
  editForm.id = row.id
  editForm.name = row.name
  editForm.endpoint = row.endpoint || ''
  editDialogVisible.value = true
}

const handleEdit = async () => {
  submitting.value = true
  try {
    await updateProvider(editForm.id, { name: editForm.name, endpoint: editForm.endpoint }, { skipErrorToast: true })
    ElMessage.success(t('providers.updateSuccess'))
    editDialogVisible.value = false
    await loadProviders()
  } catch (e) {
    showError(e.response?.data?.message || t('providers.updateFailed'))
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(t('providers.confirmDeleteText'), t('common.confirmDelete'), { type: 'warning' })
    await deleteProvider(row.id, { skipErrorToast: true })
    ElMessage.success(t('providers.deleted'))
    await loadProviders()
  } catch (e) {
    if (e !== 'cancel') showError(e.response?.data?.message || t('providers.deleteFailed'))
  }
}

const viewDomains = async (row) => {
  currentProvider.value = row
  domains.value = []
  domainsDialogVisible.value = true
  loadingDomains.value = true
  try {
    const res = await listProviderDomains(row.id, { skipErrorToast: true })
    domains.value = res.data || []
  } catch (e) {
    showError(e.response?.data?.message || t('providers.fetchDomainsFailed'))
  } finally {
    loadingDomains.value = false
  }
}

const handleClaim = async (domain) => {
  claiming[domain.domain_name] = true
  try {
    await claimDomain(currentProvider.value.id, { domain_name: domain.domain_name }, { skipErrorToast: true })
    ElMessage.success(t('providers.claimSuccess', { domain: domain.domain_name }))
  } catch (e) {
    showError(e.response?.data?.message || t('providers.claimFailed'))
  } finally {
    claiming[domain.domain_name] = false
  }
}
</script>

<style scoped>
.providers-page {
  max-width: 100%;
}
@media (max-width: 768px) {
  .providers-page {
    max-width: 100%;
  }
}
</style>
