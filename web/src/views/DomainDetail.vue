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
      <el-col :span="17">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>DNS 记录</span>
              <el-button type="primary" size="small" @click="showAddRecord = true">
                <el-icon><component :is="'Plus'" /></el-icon>
                添加记录
              </el-button>
            </div>
          </template>
          <el-table :data="records" stripe v-loading="loading">
            <el-table-column prop="host" label="主机记录" width="120" />
            <el-table-column prop="record_type" label="类型" width="80" />
            <el-table-column prop="value" label="记录值" show-overflow-tooltip />
            <el-table-column prop="ttl" label="TTL" width="80" />
            <el-table-column prop="sync_status" label="同步状态" width="100">
              <template #default="{ row }">
                <el-tag :type="statusType(row.sync_status)" size="small">{{ statusLabel(row.sync_status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="150" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="editRecord(row)">编辑</el-button>
                <el-button link type="danger" size="small" @click="handleDeleteRecord(row.id)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!loading && records.length === 0" description="暂无 DNS 记录" />
        </el-card>
      </el-col>

      <el-col :span="7">
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
      </el-col>
    </el-row>

    <!-- 添加记录 -->
    <el-dialog v-model="showAddRecord" title="添加 DNS 记录" width="480px">
      <el-form :model="recordForm" label-width="80px">
        <el-form-item label="主机记录">
          <el-input v-model="recordForm.host" placeholder="@ 表示根域名" />
        </el-form-item>
        <el-form-item label="记录类型">
          <el-select v-model="recordForm.record_type" style="width:100%">
            <el-option label="A - IPv4 地址" value="A" />
            <el-option label="AAAA - IPv6 地址" value="AAAA" />
            <el-option label="CNAME - 别名" value="CNAME" />
            <el-option label="TXT - 文本记录" value="TXT" />
            <el-option label="MX - 邮件" value="MX" />
          </el-select>
        </el-form-item>
        <el-form-item label="记录值">
          <el-input v-model="recordForm.value" placeholder="例如 192.168.1.1" />
        </el-form-item>
        <el-form-item label="TTL">
          <el-input-number v-model="recordForm.ttl" :min="60" :max="86400" />
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
        <el-form-item label="记录值">
          <el-input v-model="editForm.value" />
        </el-form-item>
        <el-form-item label="TTL">
          <el-input-number v-model="editForm.ttl" :min="60" :max="86400" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditRecord = false">取消</el-button>
        <el-button type="primary" @click="handleUpdateRecord">保存</el-button>
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
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getDomain, createDomain, transferDomain, deleteDomain } from '../api/domain'
import { getRecords, createRecord, updateRecord, deleteRecord } from '../api/record'
import { ElMessage, ElMessageBox } from 'element-plus'

const route = useRoute()
const router = useRouter()
const domainId = route.params.id

const domain = ref(null)
const records = ref([])
const loading = ref(false)
const showAddRecord = ref(false)
const showEditRecord = ref(false)
const showCreateChild = ref(false)
const showTransfer = ref(false)

const recordForm = ref({ host: '@', record_type: 'A', value: '', ttl: 600 })
const editForm = ref({ id: null, host: '', record_type: '', value: '', ttl: 600 })
const childForm = ref({ host: '' })
const transferForm = ref({ target_user_id: 1 })

const statusType = (s) => s === 'synced' ? 'success' : s === 'failed' ? 'danger' : 'warning'
const statusLabel = (s) => s === 'synced' ? '已同步' : s === 'failed' ? '同步失败' : '等待同步'

const loadData = async () => {
  loading.value = true
  try {
    const [domainRes, recordsRes] = await Promise.all([
      getDomain(domainId),
      getRecords(domainId)
    ])
    domain.value = domainRes.data
    records.value = recordsRes.data
  } finally {
    loading.value = false
  }
}

const handleAddRecord = async () => {
  await createRecord(domainId, recordForm.value)
  ElMessage.success('记录添加成功')
  showAddRecord.value = false
  recordForm.value = { host: '@', record_type: 'A', value: '', ttl: 600 }
  loadData()
}

const editRecord = (row) => {
  editForm.value = { id: row.id, host: row.host, record_type: row.record_type, value: row.value, ttl: row.ttl }
  showEditRecord.value = true
}

const handleUpdateRecord = async () => {
  await updateRecord(editForm.value.id, { value: editForm.value.value, ttl: editForm.value.ttl })
  ElMessage.success('记录更新成功')
  showEditRecord.value = false
  loadData()
}

const handleDeleteRecord = async (id) => {
  await ElMessageBox.confirm('确定删除此记录？', '提示', { type: 'warning' })
  await deleteRecord(id)
  ElMessage.success('记录已删除')
  loadData()
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
</style>
