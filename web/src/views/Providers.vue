<template>
  <div class="providers-page">
    <el-card>
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center">
          <span>DNS 提供商</span>
          <el-button type="primary" size="small" @click="openAddDialog">添加提供商</el-button>
        </div>
      </template>
      <el-table :data="providers" stripe v-loading="loading" style="width:100%">
        <el-table-column prop="name" label="名称" min-width="120" />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ row.provider_type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="AccessKey" width="140">
          <template #default="{ row }">
            <span style="font-family:monospace;font-size:12px">****{{ row.access_key_id?.slice(-4) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'danger'" size="small">
              {{ row.status === 'active' ? '正常' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button size="small" text type="primary" @click="viewDomains(row)">域名列表</el-button>
            <el-button size="small" text type="warning" @click="openEditDialog(row)">编辑</el-button>
            <el-button size="small" text type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Add Provider Dialog -->
    <el-dialog v-model="addDialogVisible" title="添加 DNS 提供商" width="480px">
      <el-form :model="addForm" label-width="100px">
        <el-form-item label="提供商类型">
          <el-select v-model="addForm.provider_type" style="width:100%" filterable>
            <el-option v-for="p in providerTypes" :key="p.value" :label="p.label" :value="p.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="addForm.name" placeholder="为这个提供商命名" />
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
        <el-button @click="addDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleAdd">添加</el-button>
      </template>
    </el-dialog>

    <!-- Edit Provider Dialog -->
    <el-dialog v-model="editDialogVisible" title="编辑提供商" width="480px">
      <el-form :model="editForm" label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="editForm.name" />
        </el-form-item>
        <el-form-item label="Endpoint">
          <el-input v-model="editForm.endpoint" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleEdit">保存</el-button>
      </template>
    </el-dialog>

    <!-- Domains Dialog -->
    <el-dialog v-model="domainsDialogVisible" :title="'域名列表 - ' + currentProvider?.name" width="600px">
      <el-table :data="domains" stripe v-loading="loadingDomains" style="width:100%">
        <el-table-column prop="domain_name" label="域名" min-width="200" />
        <el-table-column prop="record_count" label="记录数" width="100" />
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button size="small" type="primary" :loading="claiming[row.domain_name]" @click="handleClaim(row)">
              认领
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { listProviders, createProvider, updateProvider, deleteProvider, listProviderDomains, claimDomain } from '../api/provider'
import { ElMessage, ElMessageBox } from 'element-plus'

const providerTypes = [
  { value: 'aliyun', label: '阿里云 DNS' },
  { value: 'aliesa', label: '阿里云 ESA' },
  { value: 'tencentcloud', label: '腾讯云 DNSPod' },
  { value: 'cloudflare', label: 'Cloudflare' },
  { value: 'huaweicloud', label: '华为云 DNS' },
  { value: 'baiducloud', label: '百度云 DNS' },
  { value: 'trafficroute', label: '火山引擎 DNS' },
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
  { value: 'rainyun', label: '雨云' },
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
  aliyun: { id: 'AccessKey ID', secret: 'AccessKey Secret', ep: '可选，默认 alidns.aliyuncs.com' },
  aliesa: { id: 'AccessKey ID', secret: 'AccessKey Secret', ep: '可选' },
  tencentcloud: { id: 'SecretId', secret: 'SecretKey', ep: '可选，默认 dnspod.tencentcloudapi.com' },
  cloudflare: { id: 'API Token', secret: '(留空)', ep: '可选' },
  huaweicloud: { id: 'Access Key', secret: 'Secret Key', ep: '可选，默认 dns.myhuaweicloud.com' },
  baiducloud: { id: 'Access Key', secret: 'Secret Key', ep: '可选' },
  trafficroute: { id: 'Access Key', secret: 'Secret Key', ep: '可选' },
  godaddy: { id: 'API Key', secret: 'API Secret', ep: '可选，默认 api.godaddy.com' },
  namecheap: { id: 'API User', secret: 'API Key', ep: '可选，默认 api.namecheap.com' },
  namesilo: { id: 'API Key', secret: '(留空)', ep: '可选' },
  porkbun: { id: 'API Key', secret: 'Secret API Key', ep: '可选' },
  vercel: { id: 'API Token', secret: '(留空)', ep: '可选' },
  gcore: { id: 'API Key', secret: '(留空)', ep: '可选' },
  nsone: { id: 'API Key', secret: '(留空)', ep: '可选' },
  name_com: { id: '用户名', secret: 'API Token', ep: '可选，默认 api.name.com' },
  cloudns: { id: 'Auth ID', secret: 'Auth Password', ep: '可选' },
  dnsla: { id: '用户名', secret: '密码', ep: '可选' },
  dynadot: { id: 'API Key', secret: '(留空)', ep: '可选' },
  dynv6: { id: 'API Token', secret: '(留空)', ep: '可选' },
  spaceship: { id: 'API Key', secret: 'API Secret', ep: '可选' },
  edgeone: { id: 'SecretId', secret: 'SecretKey', ep: '可选' },
  rainyun: { id: 'API Key', secret: '(留空)', ep: '可选' },
  hipmdnsmgr: { id: 'API Token', secret: '(留空)', ep: '可选' },
  nowcn: { id: 'API Key', secret: 'API Secret', ep: '可选' },
  eranet: { id: 'API Key', secret: 'API Secret', ep: '可选' },
  tnethk: { id: 'API Key', secret: 'API Secret', ep: '可选' },
}

const credentialLabel = computed(() => ({
  id: credentialLabels[addForm.provider_type]?.id || 'AccessKey ID',
  secret: credentialLabels[addForm.provider_type]?.secret || 'AccessKey Secret',
}))
const credentialPlaceholder = computed(() => ({
  id: `输入 ${credentialLabel.value.id}`,
  secret: credentialLabels[addForm.provider_type]?.secret === '(留空)' ? '此提供商不需要' : `输入 ${credentialLabel.value.secret}`,
}))
const endpointPlaceholder = computed(() =>
  credentialLabels[addForm.provider_type]?.ep || '可选'
)

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
  const needsSecret = credentialLabels[addForm.provider_type]?.secret !== '(留空)'
  if (!addForm.name || !addForm.access_key_id || (needsSecret && !addForm.access_key_secret)) {
    ElMessage.warning('请填写必要字段')
    return
  }
  submitting.value = true
  try {
    await createProvider(addForm)
    ElMessage.success('提供商添加成功')
    addDialogVisible.value = false
    await loadProviders()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '添加失败')
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
    await updateProvider(editForm.id, { name: editForm.name, endpoint: editForm.endpoint })
    ElMessage.success('更新成功')
    editDialogVisible.value = false
    await loadProviders()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '更新失败')
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm('确定删除该提供商？需先解除所有关联域名。', '确认删除', { type: 'warning' })
    await deleteProvider(row.id)
    ElMessage.success('已删除')
    await loadProviders()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error(e.response?.data?.message || '删除失败')
  }
}

const viewDomains = async (row) => {
  currentProvider.value = row
  domains.value = []
  domainsDialogVisible.value = true
  loadingDomains.value = true
  try {
    const res = await listProviderDomains(row.id)
    domains.value = res.data || []
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '获取域名列表失败')
  } finally {
    loadingDomains.value = false
  }
}

const handleClaim = async (domain) => {
  claiming[domain.domain_name] = true
  try {
    await claimDomain(currentProvider.value.id, { domain_name: domain.domain_name })
    ElMessage.success(`域名 ${domain.domain_name} 认领成功`)
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '认领失败')
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
