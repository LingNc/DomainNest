<!-- web/src/views/RAMTokens.vue -->
<template>
  <div>
    <div class="page-header">
      <div>
        <h2>API Tokens</h2>
        <p class="subtitle">管理 RAM Token，用于 DDNS 回调和 API 编程访问</p>
      </div>
      <el-button type="primary" size="small" @click="openCreate">
        <el-icon><component :is="'Plus'" /></el-icon>
        创建 Token
      </el-button>
    </div>

    <el-card>
      <el-table :data="tokens" stripe v-loading="loading" style="width:100%">
        <el-table-column prop="name" label="名称" min-width="120" />
        <el-table-column label="Token" min-width="220">
          <template #default="{ row }">
            <div class="token-cell">
              <code class="token-masked">{{ maskToken(row.token) }}</code>
              <el-button link type="primary" size="small" @click="copyToken(row.token)">复制</el-button>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="启用" width="70">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" size="small" @change="(val) => handleToggle(row.id, val)" />
          </template>
        </el-table-column>
        <el-table-column label="域名限制" min-width="120">
          <template #default="{ row }">
            <span v-if="row.allowed_domains">{{ row.allowed_domains }}</span>
            <span v-else style="color:#909399">全部</span>
          </template>
        </el-table-column>
        <el-table-column label="类型限制" min-width="120">
          <template #default="{ row }">
            <span v-if="row.allowed_types">{{ row.allowed_types }}</span>
            <span v-else style="color:#909399">全部</span>
          </template>
        </el-table-column>
        <el-table-column label="IP 限制" min-width="120">
          <template #default="{ row }">
            <span v-if="row.allowed_ips">{{ row.allowed_ips }}</span>
            <span v-else style="color:#909399">不限</span>
          </template>
        </el-table-column>
        <el-table-column prop="usage_count" label="使用次数" width="90" />
        <el-table-column prop="last_used_at" label="最后使用" width="170">
          <template #default="{ row }">{{ row.last_used_at || '-' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="openEdit(row)">编辑</el-button>
            <el-button link type="warning" size="small" @click="handleReset(row.id)">重置</el-button>
            <el-button link type="danger" size="small" @click="handleDelete(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && tokens.length === 0" description="暂无 API Token" />
    </el-card>

    <!-- 创建 Token -->
    <el-dialog v-model="showCreate" title="创建 API Token" width="520px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="例如 DDNS-家庭网络" />
        </el-form-item>
        <el-form-item label="域名限制">
          <el-input v-model="form.allowed_domains_text" type="textarea" :rows="2" placeholder="域名节点 ID，每行一个，留空=全部域名" />
        </el-form-item>
        <el-form-item label="类型限制">
          <el-checkbox-group v-model="form.allowed_types">
            <el-checkbox v-for="t in recordTypes" :key="t" :label="t">{{ t }}</el-checkbox>
          </el-checkbox-group>
          <div style="color:#909399;font-size:12px;margin-top:4px">不选 = 允许所有类型</div>
        </el-form-item>
        <el-form-item label="IP 限制">
          <el-input v-model="form.allowed_ips_text" type="textarea" :rows="2" placeholder="每行一个 CIDR，例如 192.168.1.0/24，留空=不限" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate = false">取消</el-button>
        <el-button type="primary" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>

    <!-- 编辑 Token -->
    <el-dialog v-model="showEdit" title="编辑 API Token" width="520px">
      <el-form :model="editForm" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="editForm.name" />
        </el-form-item>
        <el-form-item label="域名限制">
          <el-input v-model="editForm.allowed_domains_text" type="textarea" :rows="2" placeholder="域名节点 ID，每行一个，留空=全部域名" />
        </el-form-item>
        <el-form-item label="类型限制">
          <el-checkbox-group v-model="editForm.allowed_types">
            <el-checkbox v-for="t in recordTypes" :key="t" :label="t">{{ t }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="IP 限制">
          <el-input v-model="editForm.allowed_ips_text" type="textarea" :rows="2" placeholder="每行一个 CIDR" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEdit = false">取消</el-button>
        <el-button type="primary" @click="handleUpdate">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { listRAMTokens, createRAMToken, updateRAMToken, resetRAMToken, deleteRAMToken } from '../api/ramToken'
import { ElMessage, ElMessageBox } from 'element-plus'

const recordTypes = ['A', 'AAAA', 'CNAME', 'ALIAS', 'MX', 'TXT', 'CAA', 'NS', 'SRV']

const tokens = ref([])
const loading = ref(false)
const showCreate = ref(false)
const showEdit = ref(false)
const editTargetId = ref(null)

const form = ref({ name: '', allowed_domains_text: '', allowed_types: [], allowed_ips_text: '' })
const editForm = ref({ name: '', allowed_domains_text: '', allowed_types: [], allowed_ips_text: '' })

const maskToken = (t) => {
  if (!t || t.length < 10) return t
  return t.slice(0, 6) + '****' + t.slice(-4)
}

const copyToken = (t) => {
  navigator.clipboard.writeText(t)
  ElMessage.success('已复制到剪贴板')
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
  ElMessage.success('Token 创建成功')
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
  ElMessage.success('Token 已更新')
  showEdit.value = false
  loadTokens()
}

const handleToggle = async (id, enabled) => {
  await updateRAMToken(id, { enabled })
  ElMessage.success(enabled ? 'Token 已启用' : 'Token 已禁用')
}

const handleReset = async (id) => {
  await ElMessageBox.confirm('重置后旧 Token 将立即失效，确认继续？', '重置 Token', { type: 'warning' })
  await resetRAMToken(id)
  ElMessage.success('Token 已重置')
  loadTokens()
}

const handleDelete = async (id) => {
  await ElMessageBox.confirm('删除后使用此 Token 的所有服务将无法访问，确认删除？', '删除 Token', { type: 'error' })
  await deleteRAMToken(id)
  ElMessage.success('Token 已删除')
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
