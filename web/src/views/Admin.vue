<!-- web/src/views/Admin.vue -->
<template>
  <div>
    <div class="page-header">
      <h2>管理后台</h2>
      <p class="subtitle">用户管理、域名分配与操作日志</p>
    </div>

    <el-card>
      <el-tabs v-model="activeTab">
        <!-- 用户管理 -->
        <el-tab-pane label="用户管理" name="users">
          <el-table :data="users" stripe v-loading="loading" style="width:100%">
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="username" label="用户名" min-width="100" />
            <el-table-column prop="email" label="邮箱" min-width="140" show-overflow-tooltip />
            <el-table-column prop="role" label="角色" width="90">
              <template #default="{ row }">
                <el-tag :type="row.role === 'admin' ? 'danger' : 'info'" size="small">
                  {{ row.role === 'admin' ? '管理员' : '普通用户' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="80">
              <template #default="{ row }">
                <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">
                  {{ row.status === 1 ? '正常' : '禁用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="invite_limit" label="邀请上限" width="90" />
            <el-table-column prop="invite_count" label="已邀请" width="80" />
            <el-table-column prop="created_at" label="注册时间" width="170" />
            <el-table-column label="操作" width="200" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="openEditUser(row)">编辑</el-button>
                <el-button link type="warning" size="small" @click="openResetPwd(row)">重置密码</el-button>
                <el-button v-if="row.status === 1" link type="danger" size="small" @click="handleDisable(row)">禁用</el-button>
                <el-button v-else link type="success" size="small" @click="handleEnable(row)">启用</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 域名管理 -->
        <el-tab-pane label="域名管理" name="domains">
          <div class="domain-actions">
            <el-form :model="domainForm" @submit.prevent="handleCreateRoot" inline>
              <el-form-item label="主机名">
                <el-input v-model="domainForm.host" placeholder="例如 example" />
              </el-form-item>
              <el-form-item label="后缀">
                <el-input v-model="domainForm.domain_suffix" placeholder="例如 com" />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" native-type="submit">创建根域名</el-button>
              </el-form-item>
            </el-form>
          </div>

          <el-table :data="allDomains" stripe v-loading="loading" style="width:100%">
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="full_domain" label="完整域名" min-width="180" />
            <el-table-column label="当前所有者" min-width="120">
              <template #default="{ row }">
                <el-tag size="small">{{ row.owner?.username || row.owner_id }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="创建时间" width="170" />
            <el-table-column label="操作" width="100" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="openAssign(row)">分配</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 操作日志 -->
        <el-tab-pane label="操作日志" name="logs">
          <el-table :data="logs" stripe v-loading="loading" style="width:100%">
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="user_id" label="用户 ID" width="80" />
            <el-table-column prop="action" label="操作" min-width="120" />
            <el-table-column prop="target_type" label="目标类型" min-width="100" />
            <el-table-column prop="ip_address" label="IP 地址" width="130" />
            <el-table-column prop="created_at" label="时间" width="170" />
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 分配域名对话框 -->
    <el-dialog v-model="showAssign" title="分配域名" width="420px">
      <p class="assign-info">
        将 <strong>{{ assignTarget?.full_domain }}</strong> 分配给：
      </p>
      <el-select v-model="assignUserId" placeholder="选择用户" style="width:100%">
        <el-option
          v-for="u in users"
          :key="u.id"
          :label="`${u.username} (ID: ${u.id})`"
          :value="u.id"
        />
      </el-select>
      <template #footer>
        <el-button @click="showAssign = false">取消</el-button>
        <el-button type="primary" @click="handleAssign">确认分配</el-button>
      </template>
    </el-dialog>

    <!-- 编辑用户对话框 -->
    <el-dialog v-model="showEditUser" title="编辑用户" width="480px">
      <el-form :model="editForm" label-width="80px">
        <el-form-item label="用户名">
          <el-input :model-value="editTarget?.username" disabled />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="editForm.role" style="width:100%">
            <el-option label="普通用户" value="user" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="editForm.status" style="width:100%">
            <el-option label="正常" :value="1" />
            <el-option label="禁用" :value="0" />
          </el-select>
        </el-form-item>
        <el-form-item label="邀请上限">
          <el-input-number v-model="editForm.invite_limit" :min="0" :max="9999" />
        </el-form-item>
        <el-form-item label="昵称">
          <el-input v-model="editForm.nickname" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="editForm.email" />
        </el-form-item>
        <el-form-item label="手机">
          <el-input v-model="editForm.phone" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditUser = false">取消</el-button>
        <el-button type="primary" @click="handleEditUser">保存</el-button>
      </template>
    </el-dialog>

    <!-- 重置密码对话框 -->
    <el-dialog v-model="showResetPwd" title="重置密码" width="400px">
      <p style="margin-bottom:12px">为 <strong>{{ resetPwdTarget?.username }}</strong> 设置新密码：</p>
      <el-input v-model="newPassword" type="password" placeholder="新密码（至少6位）" show-password />
      <template #footer>
        <el-button @click="showResetPwd = false">取消</el-button>
        <el-button type="warning" @click="handleResetPwd">确认重置</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { listUsers, listLogs, listDomains, createRootDomain, assignDomain, updateUser, adminResetPassword, disableUser } from '../api/admin'
import { ElMessage, ElMessageBox } from 'element-plus'

const activeTab = ref('users')
const users = ref([])
const logs = ref([])
const allDomains = ref([])
const loading = ref(false)
const domainForm = ref({ host: '', domain_suffix: '' })

const showAssign = ref(false)
const assignTarget = ref(null)
const assignUserId = ref(null)

const showEditUser = ref(false)
const editTarget = ref(null)
const editForm = reactive({ role: '', status: 1, invite_limit: 5, nickname: '', email: '', phone: '' })

const showResetPwd = ref(false)
const resetPwdTarget = ref(null)
const newPassword = ref('')

const loadData = async () => {
  loading.value = true
  try {
    const [usersRes, logsRes, domainsRes] = await Promise.all([
      listUsers(),
      listLogs({ page: 1, page_size: 50 }),
      listDomains()
    ])
    users.value = usersRes.data
    logs.value = logsRes.data.items
    allDomains.value = domainsRes.data
  } finally {
    loading.value = false
  }
}

const handleCreateRoot = async () => {
  await createRootDomain(domainForm.value)
  ElMessage.success('根域名创建成功')
  domainForm.value = { host: '', domain_suffix: '' }
  loadData()
}

const openAssign = (row) => {
  assignTarget.value = row
  assignUserId.value = null
  showAssign.value = true
}

const handleAssign = async () => {
  if (!assignUserId.value) {
    ElMessage.warning('请选择目标用户')
    return
  }
  await assignDomain(assignTarget.value.id, { user_id: assignUserId.value })
  ElMessage.success('域名分配成功')
  showAssign.value = false
  loadData()
}

const openEditUser = (row) => {
  editTarget.value = row
  editForm.role = row.role
  editForm.status = row.status
  editForm.invite_limit = row.invite_limit || 5
  editForm.nickname = row.nickname || ''
  editForm.email = row.email || ''
  editForm.phone = row.phone || ''
  showEditUser.value = true
}

const handleEditUser = async () => {
  await updateUser(editTarget.value.id, editForm)
  ElMessage.success('用户信息已更新')
  showEditUser.value = false
  loadData()
}

const openResetPwd = (row) => {
  resetPwdTarget.value = row
  newPassword.value = ''
  showResetPwd.value = true
}

const handleResetPwd = async () => {
  if (!newPassword.value || newPassword.value.length < 6) {
    ElMessage.warning('密码至少6位')
    return
  }
  await adminResetPassword(resetPwdTarget.value.id, { new_password: newPassword.value })
  ElMessage.success('密码已重置')
  showResetPwd.value = false
}

const handleDisable = async (row) => {
  await ElMessageBox.confirm(`确定要禁用用户 ${row.username} 吗？`, '确认', { type: 'warning' })
  await disableUser(row.id)
  ElMessage.success('用户已禁用')
  loadData()
}

const handleEnable = async (row) => {
  await updateUser(row.id, { status: 1 })
  ElMessage.success('用户已启用')
  loadData()
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
.domain-actions {
  margin-bottom: 16px;
}
.assign-info {
  margin-bottom: 16px;
  color: #303133;
  font-size: 14px;
}

@media (max-width: 768px) {
  .domain-actions :deep(.el-form--inline .el-form-item) {
    display: block;
    margin-right: 0;
    margin-bottom: 12px;
  }
  .domain-actions :deep(.el-form--inline) {
    display: block;
  }
}
</style>
