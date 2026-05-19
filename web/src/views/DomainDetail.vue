<!-- web/src/views/DomainDetail.vue -->
<template>
  <div>
    <div class="page-header">
      <div>
        <el-button text @click="$router.push('/dashboard')">
          <el-icon><component :is="'ArrowLeft'" /></el-icon>
          返回
        </el-button>
        <h2>{{ domain?.full_domain || '加载中...' }}</h2>
      </div>
    </div>

    <el-row :gutter="20">
      <el-col :xs="24" :lg="17">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>DNS 记录</span>
              <div class="header-actions">
                <el-button size="small" @click="handleExport('json')">导出 JSON</el-button>
                <el-button size="small" @click="handleExport('csv')">导出 CSV</el-button>
                <el-button size="small" @click="showImport = true">导入</el-button>
                <el-button type="primary" size="small" @click="openAddRecord">
                  <el-icon><component :is="'Plus'" /></el-icon>
                  添加记录
                </el-button>
              </div>
            </div>
          </template>

          <!-- 搜索筛选栏 -->
          <div class="filter-bar">
            <el-input v-model="filters.host" placeholder="主机记录" clearable size="small" style="width:140px" @clear="loadRecords" @keyup.enter="loadRecords" />
            <el-select v-model="filters.record_type" placeholder="记录类型" clearable size="small" style="width:120px" @change="loadRecords">
              <el-option v-for="t in recordTypes" :key="t.value" :label="t.label" :value="t.value" />
            </el-select>
            <el-input v-model="filters.value" placeholder="记录值" clearable size="small" style="width:160px" @clear="loadRecords" @keyup.enter="loadRecords" />
            <el-select v-model="filters.enabled" placeholder="启用状态" clearable size="small" style="width:100px" @change="loadRecords">
              <el-option label="启用" :value="true" />
              <el-option label="禁用" :value="false" />
            </el-select>
            <el-select v-model="filters.sync_status" placeholder="同步状态" clearable size="small" style="width:110px" @change="loadRecords">
              <el-option label="已同步" value="synced" />
              <el-option label="等待同步" value="pending" />
              <el-option label="同步失败" value="failed" />
              <el-option label="已禁用" value="disabled" />
            </el-select>
            <el-button size="small" type="primary" @click="loadRecords">搜索</el-button>
          </div>

          <!-- 批量操作栏 -->
          <div class="batch-bar" v-if="selectedIds.length > 0">
            <span>已选 {{ selectedIds.length }} 项</span>
            <el-button size="small" type="success" @click="handleBatchToggle(true)">批量启用</el-button>
            <el-button size="small" type="warning" @click="handleBatchToggle(false)">批量禁用</el-button>
            <el-button size="small" type="danger" @click="handleBatchDelete">批量删除</el-button>
          </div>

          <el-table :data="records" stripe v-loading="loading" @selection-change="handleSelectionChange" row-key="id">
            <el-table-column type="selection" width="40" />
            <el-table-column prop="host" label="主机记录" width="120" />
            <el-table-column prop="record_type" label="类型" width="80" />
            <el-table-column prop="value" label="记录值" show-overflow-tooltip />
            <el-table-column prop="priority" label="优先级" width="70">
              <template #default="{ row }">{{ row.priority ?? '-' }}</template>
            </el-table-column>
            <el-table-column prop="ttl" label="TTL" width="70" />
            <el-table-column prop="line" label="线路" width="80">
              <template #default="{ row }">{{ row.line || 'default' }}</template>
            </el-table-column>
            <el-table-column label="启用" width="70">
              <template #default="{ row }">
                <el-switch v-model="row.enabled" size="small" @change="(val) => handleToggle(row.id, val)" />
              </template>
            </el-table-column>
            <el-table-column prop="sync_status" label="同步状态" width="90">
              <template #default="{ row }">
                <el-tag :type="statusType(row.sync_status)" size="small">{{ statusLabel(row.sync_status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" min-width="150" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="editRecord(row)">编辑</el-button>
                <el-button v-if="row.sync_status === 'failed' && auth.isAdmin" link type="warning" size="small" @click="handleRetrySync(row.id)">重试</el-button>
                <el-button link type="danger" size="small" @click="handleDeleteRecord(row.id)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!loading && records.length === 0" description="暂无 DNS 记录" />

          <!-- 分页 -->
          <div class="pagination-bar" v-if="total > 0">
            <el-pagination
              v-model:current-page="pagination.page"
              v-model:page-size="pagination.pageSize"
              :total="total"
              :page-sizes="[10, 20, 50, 100]"
              layout="total, sizes, prev, pager, next"
              @current-change="loadRecords"
              @size-change="loadRecords"
            />
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :lg="7" class="right-col">
        <el-card>
          <template #header>操作</template>
          <el-button type="primary" @click="showCreateChild = true" style="width:100%;margin-bottom:12px">
            创建子域名
          </el-button>
          <el-button type="warning" @click="showTransfer = true" style="width:100%;margin-bottom:12px">
            转让域名
          </el-button>
          <el-button type="danger" @click="handleDeleteDomain" style="width:100%">
            删除域名
          </el-button>
        </el-card>

        <el-card style="margin-top:16px" v-if="domain">
          <template #header>域名信息</template>
          <el-descriptions :column="1" size="small">
            <el-descriptions-item label="完整域名">{{ domain.full_domain }}</el-descriptions-item>
            <el-descriptions-item label="域名 ID">{{ domain.id }}</el-descriptions-item>
            <el-descriptions-item label="创建时间">{{ domain.created_at }}</el-descriptions-item>
          </el-descriptions>
        </el-card>

        <el-card style="margin-top:16px">
          <template #header>
            <div class="card-header">
              <span>权限管理</span>
              <el-button size="small" type="primary" @click="showGrantPerm = true">授权</el-button>
            </div>
          </template>
          <div v-if="permissions.length === 0" style="color:#909399;font-size:13px">暂无委派权限</div>
          <div v-for="p in permissions" :key="p.id" class="perm-item">
            <div class="perm-info">
              <span class="perm-user">{{ p.user?.username || 'User #' + p.user_id }}</span>
              <el-tag :type="permTagType(p.permission_level)" size="small">{{ permLabel(p.permission_level) }}</el-tag>
            </div>
            <div class="perm-detail" v-if="p.allowed_types || p.allowed_ips">
              <span v-if="p.allowed_types" class="perm-restrict">类型: {{ p.allowed_types }}</span>
              <span v-if="p.allowed_ips" class="perm-restrict">IP: {{ p.allowed_ips }}</span>
            </div>
            <el-button link type="danger" size="small" @click="handleRevokePerm(p.user_id)">撤销</el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 授权对话框 -->
    <el-dialog v-model="showGrantPerm" title="授权用户" width="480px">
      <el-form :model="grantForm" label-width="80px">
        <el-form-item label="用户 ID">
          <el-input-number v-model="grantForm.target_user_id" :min="1" style="width:100%" />
        </el-form-item>
        <el-form-item label="权限级别">
          <el-select v-model="grantForm.level" style="width:100%">
            <el-option label="只读 (read)" value="read" />
            <el-option label="读写 (write)" value="write" />
            <el-option label="管理员 (admin)" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item label="记录类型">
          <el-checkbox-group v-model="grantForm.allowed_types">
            <el-checkbox v-for="t in recordTypes" :key="t.value" :label="t.value">{{ t.value }}</el-checkbox>
          </el-checkbox-group>
          <div style="color:#909399;font-size:12px;margin-top:4px">不选 = 允许所有类型</div>
        </el-form-item>
        <el-form-item label="IP 限制">
          <el-input v-model="grantForm.allowed_ips_text" type="textarea" :rows="2" placeholder="每行一个 CIDR，例如 192.168.1.0/24，留空=不限" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showGrantPerm = false">取消</el-button>
        <el-button type="primary" @click="handleGrantPerm">确认授权</el-button>
      </template>
    </el-dialog>

    <!-- 添加记录 -->
    <el-dialog v-model="showAddRecord" title="添加 DNS 记录" width="520px">
      <el-form :model="recordForm" label-width="80px">
        <el-form-item label="主机记录">
          <el-input v-model="recordForm.host" placeholder="@ 表示根域名" />
        </el-form-item>
        <el-form-item label="记录类型">
          <el-select v-model="recordForm.record_type" style="width:100%" @change="onTypeChange">
            <el-option v-for="t in recordTypes" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="recordForm.record_type === 'MX'" label="优先级">
          <el-input-number v-model="recordForm.priority" :min="0" :max="65535" />
        </el-form-item>
        <el-form-item v-if="recordForm.record_type === 'SRV'" label="SRV 值">
          <div style="display:flex;gap:8px;width:100%">
            <el-input-number v-model="srvForm.priority" :min="0" :max="65535" placeholder="优先级" style="flex:1" />
            <el-input-number v-model="srvForm.weight" :min="0" :max="65535" placeholder="权重" style="flex:1" />
            <el-input-number v-model="srvForm.port" :min="0" :max="65535" placeholder="端口" style="flex:1" />
          </div>
        </el-form-item>
        <el-form-item v-if="recordForm.record_type === 'SRV'" label="目标地址">
          <el-input v-model="srvForm.target" placeholder="target.example.com" />
        </el-form-item>
        <el-form-item v-if="recordForm.record_type === 'CAA'" label="CAA 标志">
          <el-input-number v-model="caaForm.flag" :min="0" :max="255" />
        </el-form-item>
        <el-form-item v-if="recordForm.record_type === 'CAA'" label="CAA 标签">
          <el-select v-model="caaForm.tag" style="width:100%">
            <el-option label="issue - 授权 CA 颁发" value="issue" />
            <el-option label="issuewild - 通配符授权" value="issuewild" />
            <el-option label="iodef - 通知 URL" value="iodef" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="recordForm.record_type === 'CAA'" label="CAA 值">
          <el-input v-model="caaForm.value" placeholder="例如 letsencrypt.org" />
        </el-form-item>
        <el-form-item v-if="!['SRV', 'CAA'].includes(recordForm.record_type)" label="记录值">
          <el-input v-model="recordForm.value" :placeholder="valuePlaceholder" />
        </el-form-item>
        <el-form-item label="TTL">
          <el-select v-model="recordForm.ttl" style="width:100%">
            <el-option label="1 分钟 (60s)" :value="60" />
            <el-option label="5 分钟 (300s)" :value="300" />
            <el-option label="10 分钟 (600s)" :value="600" />
            <el-option label="30 分钟 (1800s)" :value="1800" />
            <el-option label="1 小时 (3600s)" :value="3600" />
            <el-option label="12 小时 (43200s)" :value="43200" />
            <el-option label="1 天 (86400s)" :value="86400" />
          </el-select>
        </el-form-item>
        <el-form-item label="线路">
          <el-select v-model="recordForm.line" style="width:100%">
            <el-option label="默认" value="default" />
            <el-option label="中国电信" value="telecom" />
            <el-option label="中国联通" value="unicom" />
            <el-option label="中国移动" value="mobile" />
            <el-option label="境外" value="overseas" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddRecord = false">取消</el-button>
        <el-button type="primary" @click="handleAddRecord">添加</el-button>
      </template>
    </el-dialog>

    <!-- 编辑记录 -->
    <el-dialog v-model="showEditRecord" title="编辑 DNS 记录" width="480px">
      <el-form :model="editForm" label-width="80px">
        <el-form-item label="主机记录">
          <el-input v-model="editForm.host" disabled />
        </el-form-item>
        <el-form-item label="记录类型">
          <el-input v-model="editForm.record_type" disabled />
        </el-form-item>
        <el-form-item v-if="editForm.record_type === 'MX'" label="优先级">
          <el-input-number v-model="editForm.priority" :min="0" :max="65535" />
        </el-form-item>
        <el-form-item label="记录值">
          <el-input v-model="editForm.value" />
        </el-form-item>
        <el-form-item label="TTL">
          <el-select v-model="editForm.ttl" style="width:100%">
            <el-option label="1 分钟 (60s)" :value="60" />
            <el-option label="5 分钟 (300s)" :value="300" />
            <el-option label="10 分钟 (600s)" :value="600" />
            <el-option label="30 分钟 (1800s)" :value="1800" />
            <el-option label="1 小时 (3600s)" :value="3600" />
            <el-option label="12 小时 (43200s)" :value="43200" />
            <el-option label="1 天 (86400s)" :value="86400" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditRecord = false">取消</el-button>
        <el-button type="primary" @click="handleUpdateRecord">保存</el-button>
      </template>
    </el-dialog>

    <!-- 导入记录 -->
    <el-dialog v-model="showImport" title="导入 DNS 记录" width="480px">
      <el-tabs v-model="importTab">
        <el-tab-pane label="JSON 导入" name="json">
          <el-input v-model="importJson" type="textarea" :rows="10" placeholder='[{"host":"@","record_type":"A","value":"1.2.3.4","ttl":600}]' />
        </el-tab-pane>
        <el-tab-pane label="CSV 导入" name="csv">
          <el-upload
            :auto-upload="false"
            :limit="1"
            accept=".csv"
            :on-change="handleCsvFile"
            :file-list="csvFileList"
          >
            <el-button size="small">选择 CSV 文件</el-button>
            <template #tip>
              <div class="el-upload__tip">格式: host,record_type,value,ttl,priority,line,enabled</div>
            </template>
          </el-upload>
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <el-button @click="showImport = false">取消</el-button>
        <el-button type="primary" @click="handleImport">导入</el-button>
      </template>
    </el-dialog>

    <!-- 创建子域名 -->
    <el-dialog v-model="showCreateChild" title="创建子域名" width="400px">
      <el-form :model="childForm" label-width="80px">
        <el-form-item label="子域名">
          <el-input v-model="childForm.host" placeholder="输入子域名名称" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateChild = false">取消</el-button>
        <el-button type="primary" @click="handleCreateChild">创建</el-button>
      </template>
    </el-dialog>

    <!-- 转让域名 -->
    <el-dialog v-model="showTransfer" title="转让域名" width="400px">
      <el-alert type="warning" :closable="false" style="margin-bottom:16px">
        转让将把该域名及其所有子域名移交给目标用户，此操作不可撤销。
      </el-alert>
      <el-form :model="transferForm" label-width="80px">
        <el-form-item label="目标用户">
          <el-input-number v-model="transferForm.target_user_id" :min="1" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showTransfer = false">取消</el-button>
        <el-button type="warning" @click="handleTransfer">确认转让</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getDomain, createDomain, transferDomain, deleteDomain } from '../api/domain'
import { getRecords, createRecord, updateRecord, deleteRecord, toggleRecord, batchDeleteRecords, batchToggleRecords, exportRecords, importRecords } from '../api/record'
import { retrySync } from '../api/admin'
import { getPermissions, grantPermission, revokePermission } from '../api/permission'
import { useAuthStore } from '../stores/auth'
import { ElMessage, ElMessageBox } from 'element-plus'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const domainId = route.params.id

const recordTypes = [
  { label: 'A - IPv4', value: 'A' },
  { label: 'AAAA - IPv6', value: 'AAAA' },
  { label: 'CNAME - 别名', value: 'CNAME' },
  { label: 'ALIAS - 根域名别名', value: 'ALIAS' },
  { label: 'MX - 邮件', value: 'MX' },
  { label: 'TXT - 文本', value: 'TXT' },
  { label: 'CAA - 证书授权', value: 'CAA' },
  { label: 'NS - 域名服务器', value: 'NS' },
  { label: 'SRV - 服务记录', value: 'SRV' },
]

const domain = ref(null)
const records = ref([])
const total = ref(0)
const loading = ref(false)
const selectedIds = ref([])

const showAddRecord = ref(false)
const showEditRecord = ref(false)
const showCreateChild = ref(false)
const showTransfer = ref(false)
const showImport = ref(false)
const importTab = ref('json')
const importJson = ref('')
const csvFile = ref(null)
const csvFileList = ref([])

const permissions = ref([])
const showGrantPerm = ref(false)
const grantForm = ref({ target_user_id: null, level: 'read', allowed_types: [], allowed_ips_text: '' })

const filters = reactive({ host: '', record_type: '', value: '', enabled: undefined, sync_status: '' })
const pagination = reactive({ page: 1, pageSize: 20 })

const recordForm = ref({ host: '@', record_type: 'A', value: '', ttl: 600, priority: null, line: 'default' })
const editForm = ref({ id: null, host: '', record_type: '', value: '', ttl: 600, priority: null })
const srvForm = ref({ priority: 0, weight: 0, port: 0, target: '' })
const caaForm = ref({ flag: 0, tag: 'issue', value: '' })
const childForm = ref({ host: '' })
const transferForm = ref({ target_user_id: 1 })

const statusType = (s) => s === 'synced' ? 'success' : s === 'failed' ? 'danger' : s === 'disabled' ? 'info' : 'warning'
const statusLabel = (s) => ({ synced: '已同步', failed: '同步失败', pending: '等待同步', disabled: '已禁用' }[s] || s)

const valuePlaceholder = computed(() => {
  const m = { A: '例如 192.168.1.1', AAAA: '例如 2001:db8::1', CNAME: '例如 example.com', ALIAS: '例如 example.com', MX: '例如 mail.example.com', TXT: '例如 v=spf1 include:example.com ~all', NS: '例如 ns1.example.com' }
  return m[recordForm.value.record_type] || '输入记录值'
})

const loadRecords = async () => {
  loading.value = true
  try {
    const params = { page: pagination.page, page_size: pagination.pageSize }
    if (filters.host) params.host = filters.host
    if (filters.record_type) params.record_type = filters.record_type
    if (filters.value) params.value = filters.value
    if (filters.enabled !== undefined && filters.enabled !== '') params.enabled = filters.enabled
    if (filters.sync_status) params.sync_status = filters.sync_status
    const res = await getRecords(domainId, params)
    records.value = res.data.items
    total.value = res.data.total
  } finally {
    loading.value = false
  }
}

const loadDomain = async () => {
  const res = await getDomain(domainId)
  domain.value = res.data
}

const loadData = async () => {
  await Promise.all([loadDomain(), loadRecords(), loadPermissions()])
}

const onTypeChange = () => {
  recordForm.value.value = ''
  recordForm.value.priority = null
  srvForm.value = { priority: 0, weight: 0, port: 0, target: '' }
  caaForm.value = { flag: 0, tag: 'issue', value: '' }
}

const buildRecordValue = () => {
  const type = recordForm.value.record_type
  if (type === 'SRV') {
    return `${srvForm.value.priority} ${srvForm.value.weight} ${srvForm.value.port} ${srvForm.value.target}`
  }
  if (type === 'CAA') {
    return `${caaForm.value.flag} ${caaForm.value.tag} ${caaForm.value.value}`
  }
  return recordForm.value.value
}

const openAddRecord = () => {
  recordForm.value = { host: '@', record_type: 'A', value: '', ttl: 600, priority: null, line: 'default' }
  srvForm.value = { priority: 0, weight: 0, port: 0, target: '' }
  caaForm.value = { flag: 0, tag: 'issue', value: '' }
  showAddRecord.value = true
}

const handleAddRecord = async () => {
  const data = {
    host: recordForm.value.host,
    record_type: recordForm.value.record_type,
    value: buildRecordValue(),
    ttl: recordForm.value.ttl,
    line: recordForm.value.line,
  }
  if (recordForm.value.record_type === 'MX' && recordForm.value.priority != null) {
    data.priority = recordForm.value.priority
  }
  await createRecord(domainId, data)
  ElMessage.success('记录添加成功')
  showAddRecord.value = false
  loadRecords()
}

const editRecord = (row) => {
  editForm.value = { id: row.id, host: row.host, record_type: row.record_type, value: row.value, ttl: row.ttl, priority: row.priority }
  showEditRecord.value = true
}

const handleUpdateRecord = async () => {
  const data = { value: editForm.value.value, ttl: editForm.value.ttl }
  if (editForm.value.record_type === 'MX' && editForm.value.priority != null) {
    data.priority = editForm.value.priority
  }
  await updateRecord(editForm.value.id, data)
  ElMessage.success('记录更新成功')
  showEditRecord.value = false
  loadRecords()
}

const handleToggle = async (id, enabled) => {
  await toggleRecord(id, { enabled })
  ElMessage.success(enabled ? '记录已启用' : '记录已禁用')
  loadRecords()
}

const handleDeleteRecord = async (id) => {
  await ElMessageBox.confirm('确定删除此记录？', '提示', { type: 'warning' })
  await deleteRecord(id)
  ElMessage.success('记录已删除')
  loadRecords()
}

const handleSelectionChange = (rows) => {
  selectedIds.value = rows.map(r => r.id)
}

const handleBatchDelete = async () => {
  await ElMessageBox.confirm(`确定删除选中的 ${selectedIds.value.length} 条记录？`, '批量删除', { type: 'warning' })
  await batchDeleteRecords(selectedIds.value)
  ElMessage.success('批量删除成功')
  loadRecords()
}

const handleBatchToggle = async (enabled) => {
  await batchToggleRecords(selectedIds.value, enabled)
  ElMessage.success(enabled ? '批量启用成功' : '批量禁用成功')
  loadRecords()
}

const handleExport = async (format) => {
  const res = await exportRecords(domainId, format)
  if (format === 'csv') {
    const url = URL.createObjectURL(new Blob([res.data]))
    const a = document.createElement('a')
    a.href = url
    a.download = 'records.csv'
    a.click()
    URL.revokeObjectURL(url)
  } else {
    const blob = new Blob([JSON.stringify(res.data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'records.json'
    a.click()
    URL.revokeObjectURL(url)
  }
}

const handleCsvFile = (file) => {
  csvFile.value = file.raw
}

const handleImport = async () => {
  if (importTab.value === 'json') {
    let data
    try {
      data = JSON.parse(importJson.value)
    } catch {
      ElMessage.error('JSON 格式错误')
      return
    }
    const res = await importRecords(domainId, data, 'json')
    const r = res.data
    ElMessage.success(`导入完成: 创建 ${r.created} 条, 跳过 ${r.skipped} 条`)
    if (r.errors?.length) console.warn('Import errors:', r.errors)
  } else {
    if (!csvFile.value) {
      ElMessage.error('请选择 CSV 文件')
      return
    }
    const res = await importRecords(domainId, csvFile.value, 'csv')
    const r = res.data
    ElMessage.success(`导入完成: 创建 ${r.created} 条, 跳过 ${r.skipped} 条`)
    if (r.errors?.length) console.warn('Import errors:', r.errors)
  }
  showImport.value = false
  importJson.value = ''
  csvFile.value = null
  csvFileList.value = []
  loadRecords()
}

const handleRetrySync = async (id) => {
  await retrySync(id)
  ElMessage.success('已重新同步')
  loadRecords()
}

const handleCreateChild = async () => {
  await createDomain({ parent_id: parseInt(domainId), host: childForm.value.host })
  ElMessage.success('子域名创建成功')
  showCreateChild.value = false
  router.push('/dashboard')
}

const handleTransfer = async () => {
  await ElMessageBox.confirm('确定转让此域名及所有子域名？此操作不可撤销。', '确认转让', { type: 'warning' })
  await transferDomain(domainId, transferForm.value)
  ElMessage.success('域名已转让')
  showTransfer.value = false
  router.push('/dashboard')
}

const handleDeleteDomain = async () => {
  await ElMessageBox.confirm('确定删除此域名？仅当无子域名和记录时可删除。', '确认删除', { type: 'error' })
  await deleteDomain(domainId)
  ElMessage.success('域名已删除')
  router.push('/dashboard')
}

const loadPermissions = async () => {
  try {
    const res = await getPermissions(domainId)
    permissions.value = res.data || []
  } catch { /* no permission to view */ }
}

const permTagType = (level) => ({ read: 'info', write: 'success', admin: 'warning' }[level] || 'info')
const permLabel = (level) => ({ read: '只读', write: '读写', admin: '管理员' }[level] || level)

const handleGrantPerm = async () => {
  const data = {
    target_user_id: grantForm.value.target_user_id,
    level: grantForm.value.level,
  }
  if (grantForm.value.allowed_types.length > 0) {
    data.allowed_types = grantForm.value.allowed_types
  }
  if (grantForm.value.allowed_ips_text.trim()) {
    data.allowed_ips = grantForm.value.allowed_ips_text.trim().split('\n').map(s => s.trim()).filter(Boolean)
  }
  await grantPermission(domainId, data)
  ElMessage.success('授权成功')
  showGrantPerm.value = false
  grantForm.value = { target_user_id: null, level: 'read', allowed_types: [], allowed_ips_text: '' }
  loadPermissions()
}

const handleRevokePerm = async (userId) => {
  await ElMessageBox.confirm('确定撤销该用户的权限？', '撤销权限', { type: 'warning' })
  await revokePermission(domainId, userId)
  ElMessage.success('权限已撤销')
  loadPermissions()
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
  margin-top: 4px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.header-actions {
  display: flex;
  gap: 8px;
}
.filter-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
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
.pagination-bar {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
@media (max-width: 1200px) {
  .right-col {
    margin-top: 16px;
  }
}
.perm-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid #f0f0f0;
  flex-wrap: wrap;
  gap: 4px;
}
.perm-item:last-child {
  border-bottom: none;
}
.perm-info {
  display: flex;
  align-items: center;
  gap: 8px;
}
.perm-user {
  font-weight: 500;
  font-size: 13px;
}
.perm-detail {
  width: 100%;
  display: flex;
  gap: 8px;
  font-size: 12px;
  color: #909399;
}
.perm-restrict {
  background: #f5f5f5;
  padding: 1px 6px;
  border-radius: 3px;
}
</style>
