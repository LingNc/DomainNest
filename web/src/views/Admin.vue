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
            <el-table-column label="头像" width="60">
              <template #default="{ row }">
                <el-avatar v-if="row.avatar" :src="row.avatar" :size="32" />
                <el-avatar v-else :size="32">{{ (row.username || '?')[0]?.toUpperCase() }}</el-avatar>
              </template>
            </el-table-column>
            <el-table-column prop="username" label="用户名" min-width="100" />
            <el-table-column prop="email" label="邮箱" min-width="140" show-overflow-tooltip />
            <el-table-column prop="role" label="角色" width="90">
              <template #default="{ row }">
                <el-tag :type="row.is_super_admin ? 'warning' : row.role === 'admin' ? 'danger' : 'info'" size="small">
                  {{ row.is_super_admin ? '超级管理员' : row.role === 'admin' ? '管理员' : '普通用户' }}
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
                <el-button link type="warning" size="small" :disabled="row.is_super_admin && row.id !== auth.user?.id" @click="openResetPwd(row)">重置密码</el-button>
                <el-button v-if="row.status === 1" link type="danger" size="small" :disabled="row.is_super_admin" @click="handleDisable(row)">禁用</el-button>
                <el-button v-else link type="success" size="small" @click="handleEnable(row)">启用</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 域名管理 -->
        <el-tab-pane label="域名管理" name="domains">
          <div class="domain-actions">
            <div style="display:flex;gap:8px;align-items:flex-end;flex-wrap:wrap">
              <el-select v-model="selectedProviderId" placeholder="选择提供商" style="width:200px" size="default">
                <el-option v-for="p in adminProviders" :key="p.id" :label="p.name" :value="p.id" />
              </el-select>
              <el-select v-model="selectedDomain" placeholder="选择域名" style="width:240px" size="default" :loading="loadingAdminDomains" :disabled="!selectedProviderId">
                <el-option v-for="d in adminDomains" :key="d.domain_name" :label="d.domain_name + (d.record_count != null ? ` (${d.record_count} 条记录)` : '')" :value="d.domain_name" />
              </el-select>
              <el-button type="primary" @click="handleCreateRoot">创建根域名</el-button>
            </div>
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
          <div class="filter-bar">
            <el-select v-model="logFilters.user_id" placeholder="用户" clearable filterable size="small" style="width:160px">
              <el-option v-for="u in users" :key="u.id" :label="`${u.nickname || u.username} (@${u.username})`" :value="u.id">
                <div style="display:flex;align-items:center;gap:6px">
                  <el-avatar v-if="u.avatar" :src="u.avatar" :size="20" />
                  <el-avatar v-else :size="20">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
                  <span>{{ u.nickname || u.username }}</span>
                </div>
              </el-option>
            </el-select>
            <el-input v-model="logFilters.q" placeholder="关键词搜索" clearable size="small" style="width:160px" />
            <el-select v-model="logFilters.action" placeholder="操作类型" clearable filterable size="small" style="width:160px">
              <el-option-group v-for="group in actionGroups" :key="group.label" :label="group.label">
                <el-option v-for="item in group.options" :key="item.value" :label="item.label" :value="item.value" />
              </el-option-group>
            </el-select>
            <el-select v-model="logFilters.target_type" placeholder="目标类型" clearable multiple collapse-tags collapse-tags-tooltip size="small" style="width:200px">
              <el-option v-for="item in targetTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-date-picker v-model="logFilters.dateRange" type="daterange" range-separator="至" start-placeholder="开始日期" end-placeholder="结束日期" size="small" style="width:260px" value-format="YYYY-MM-DD" />
            <el-button size="small" type="primary" @click="loadLogs">搜索</el-button>
            <el-button size="small" @click="resetLogFilters">重置</el-button>
          </div>
          <el-table :data="logs" stripe v-loading="loading" style="width:100%">
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column label="用户" min-width="120">
              <template #default="{ row }">
                <div style="display:flex;align-items:center;gap:6px">
                  <el-avatar v-if="row.user?.avatar" :src="row.user.avatar" :size="24" />
                  <el-avatar v-else :size="24">{{ (row.user?.username || '?')[0]?.toUpperCase() }}</el-avatar>
                  <span>{{ row.user?.username || 'User #' + row.user_id }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="action" label="操作" min-width="120" />
            <el-table-column prop="target_type" label="目标类型" min-width="100" />
            <el-table-column prop="ip_address" label="IP 地址" width="130" />
            <el-table-column label="详情" min-width="200">
              <template #default="{ row }">
                <template v-if="row.detail && row.detail !== 'null'">
                  <el-tag v-for="(val, key) in parseDetail(row.detail)" :key="key" size="small" style="margin:2px" type="info">
                    {{ formatDetailKey(key) }}: {{ formatDetailValue(val) }}
                  </el-tag>
                </template>
                <span v-else style="color:#909399">-</span>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" label="时间" width="170" />
          </el-table>
          <el-pagination v-if="logTotal > 0" :current-page="logPage" :page-size="logPageSize" :total="logTotal"
            layout="total, sizes, prev, pager, next" :page-sizes="[20, 50, 100]"
            @current-change="handleLogPageChange" @size-change="handleLogSizeChange" style="margin-top:16px" />
        </el-tab-pane>
        <el-tab-pane label="系统设置" name="settings">
          <el-form :model="smtpForm" label-width="100px" style="max-width:560px">
            <el-divider content-position="left">SMTP 邮件配置</el-divider>
            <el-form-item label="SMTP 主机">
              <el-input v-model="smtpForm.host" placeholder="例如 smtp.gmail.com" />
            </el-form-item>
            <el-form-item label="端口">
              <el-input-number v-model="smtpForm.port" :min="1" :max="65535" />
            </el-form-item>
            <el-form-item label="加密方式">
              <el-select v-model="smtpForm.tls_type" style="width:100%" @change="onTLSTypeChange">
                <el-option label="STARTTLS (port 587)" value="starttls" />
                <el-option label="SSL/TLS (port 465)" value="implicit" />
                <el-option label="无加密 (port 25)" value="none" />
              </el-select>
            </el-form-item>
            <el-form-item label="登录邮箱">
              <el-input v-model="smtpForm.username" placeholder="例如 yourname@qq.com" autocomplete="off" />
              <div class="el-form-item__tip" style="color:#909399;font-size:12px;margin-top:4px">用于登录 SMTP 服务器</div>
            </el-form-item>
            <el-form-item label="授权码">
              <el-input v-model="smtpForm.password" type="password" show-password placeholder="邮箱授权码（非登录密码）" autocomplete="new-password" />
            </el-form-item>
            <el-form-item label="发件人地址">
              <el-input v-model="smtpForm.from" placeholder="noreply@example.com" />
              <div class="el-form-item__tip" style="color:#909399;font-size:12px;margin-top:4px">邮件中显示的发件人，可与登录邮箱相同</div>
            </el-form-item>
            <el-form-item label="发件人名称">
              <el-input v-model="smtpForm.from_name" placeholder="DomainNest" />
            </el-form-item>
            <el-form-item label="测试收件人">
              <el-input v-model="testEmail" placeholder="接收测试邮件的邮箱地址" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleSaveSMTP">保存配置</el-button>
              <el-button @click="handleTestSMTP" :loading="testingSMTP" :disabled="!testEmail">发送测试邮件</el-button>
            </el-form-item>
          </el-form>
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
          :label="`${u.nickname || u.username} (@${u.username})`"
          :value="u.id"
        >
          <div style="display:flex;align-items:center;gap:8px">
            <el-avatar v-if="u.avatar" :src="u.avatar" :size="24" />
            <el-avatar v-else :size="24">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
            <span>{{ u.nickname || u.username }}</span>
            <span style="color:#909399;font-size:12px">@{{ u.username }}</span>
          </div>
        </el-option>
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
          <el-input v-model="editForm.username" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="editForm.role" style="width:100%" :disabled="editTarget?.is_super_admin">
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
          <div v-if="editInviteWarning" style="color:#e6a23c;font-size:12px;margin-top:4px">
            {{ editInviteWarning }}
          </div>
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
import { ref, reactive, onMounted, watch } from 'vue'
import { listUsers, listLogs, listDomains, createRootDomain, assignDomain, updateUser, adminResetPassword, disableUser, getSettings, updateSettings, testSMTP, promoteToAdmin, demoteFromAdmin } from '../api/admin'
import { listProviders, listProviderDomains } from '../api/provider'
import { useAuthStore } from '../stores/auth'
import { ElMessage, ElMessageBox } from 'element-plus'
import { actionGroups, targetTypeOptions } from '../constants/operationLogs'

const auth = useAuthStore()

const activeTab = ref('users')
const users = ref([])
const logs = ref([])
const allDomains = ref([])
const loading = ref(false)

const adminProviders = ref([])
const adminDomains = ref([])
const selectedProviderId = ref(null)
const selectedDomain = ref('')
const loadingAdminDomains = ref(false)

const showAssign = ref(false)
const assignTarget = ref(null)
const assignUserId = ref(null)

const showEditUser = ref(false)
const editTarget = ref(null)
const editForm = reactive({ username: '', role: '', status: 1, invite_limit: 5, nickname: '', email: '', phone: '' })

const showResetPwd = ref(false)
const resetPwdTarget = ref(null)
const newPassword = ref('')

const smtpForm = reactive({ host: '', port: 587, username: '', password: '', from: '', from_name: 'DomainNest', tls_type: 'starttls' })
const testingSMTP = ref(false)
const testEmail = ref('')

const logFilters = reactive({ user_id: '', action: '', target_type: [], q: '', dateRange: null })

const keyMap = { username: '用户名', full_domain: '域名', host: '主机记录', type: '记录类型', value: '记录值', amount: '数量', enabled: '状态', ids: '记录ID', target_user_id: '目标用户', invited_by: '邀请人', provider_name: '提供商' }

const parseDetail = (json) => {
  try { return JSON.parse(json) } catch { return {} }
}

const formatDetailKey = (key) => keyMap[key] || key

const formatDetailValue = (val) => {
  if (Array.isArray(val)) return val.join(', ')
  if (typeof val === 'boolean') return val ? '启用' : '禁用'
  return String(val)
}
const logPage = ref(1)
const logPageSize = ref(20)
const logTotal = ref(0)

const editInviteWarning = ref('')

const loadData = async () => {
  loading.value = true
  try {
    const [usersRes, domainsRes] = await Promise.all([
      listUsers(),
      listDomains()
    ])
    users.value = usersRes.data
    allDomains.value = domainsRes.data
    await loadLogs()
  } finally {
    loading.value = false
  }
  loadSMTP()
  loadAdminProviders()
}

const loadAdminProviders = async () => {
  try {
    const res = await listProviders()
    adminProviders.value = res.data || []
  } catch { /* ignore */ }
}

watch(selectedProviderId, async (id) => {
  adminDomains.value = []
  selectedDomain.value = ''
  if (!id) return
  loadingAdminDomains.value = true
  try {
    const res = await listProviderDomains(id)
    adminDomains.value = res.data || []
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '获取域名列表失败')
  } finally {
    loadingAdminDomains.value = false
  }
})

const loadLogs = async () => {
  const params = { page: logPage.value, page_size: logPageSize.value }
  if (logFilters.user_id) params.user_id = logFilters.user_id
  if (logFilters.action) params.action = logFilters.action
  if (logFilters.target_type.length > 0) params.target_type = logFilters.target_type.join(',')
  if (logFilters.q) params.q = logFilters.q
  if (logFilters.dateRange && logFilters.dateRange.length === 2) {
    params.start_time = logFilters.dateRange[0]
    params.end_time = logFilters.dateRange[1]
  }
  const logsRes = await listLogs(params)
  logs.value = logsRes.data.items
  logTotal.value = logsRes.data.total
}

const resetLogFilters = () => {
  logFilters.user_id = ''
  logFilters.action = ''
  logFilters.target_type = []
  logFilters.q = ''
  logFilters.dateRange = null
  logPage.value = 1
  loadLogs()
}

const handleLogPageChange = (p) => {
  logPage.value = p
  loadLogs()
}

const handleLogSizeChange = (s) => {
  logPageSize.value = s
  logPage.value = 1
  loadLogs()
}

const loadSMTP = async () => {
  try {
    const res = await getSettings('smtp')
    if (res.data) {
      Object.assign(smtpForm, res.data)
      if (res.data.test_email) {
        testEmail.value = res.data.test_email
      } else if (!testEmail.value && smtpForm.username) {
        testEmail.value = smtpForm.username
      }
    }
  } catch { /* no settings yet */ }
}

const handleCreateRoot = async () => {
  if (!selectedProviderId.value || !selectedDomain.value) {
    ElMessage.warning('请选择提供商和域名')
    return
  }
  await createRootDomain({ provider_id: selectedProviderId.value, domain_name: selectedDomain.value })
  ElMessage.success('根域名创建成功')
  selectedDomain.value = ''
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
  editForm.username = row.username || ''
  editForm.role = row.role
  editForm.status = row.status
  editForm.invite_limit = row.invite_limit || 5
  editForm.nickname = row.nickname || ''
  editForm.email = row.email || ''
  editForm.phone = row.phone || ''
  editInviteWarning.value = ''
  showEditUser.value = true
}

const checkInviteWarning = () => {
  const target = editTarget.value
  if (!target) return
  const currentAdmin = users.value.find(u => u.id === auth.user?.id)
  if (currentAdmin && !currentAdmin.is_super_admin) {
    const additional = editForm.invite_limit - (target.invite_limit || 0)
    if (additional > 0) {
      const available = (currentAdmin.invite_limit || 0) - (currentAdmin.invite_count || 0)
      if (additional > available) {
        editInviteWarning.value = `你的可用邀请额度不足 (可用: ${available}, 需要: ${additional})`
        return
      }
    }
  }
  editInviteWarning.value = ''
}

const handleEditUser = async () => {
  const target = editTarget.value
  // Role changes must go through dedicated promote/demote endpoints (super_admin guarded)
  if (editForm.role !== target.role) {
    if (editForm.role === 'admin') {
      await promoteToAdmin(target.id)
    } else {
      await demoteFromAdmin(target.id)
    }
  }
  // Other fields via updateUser (excluding role)
  const { role, ...rest } = editForm
  await updateUser(target.id, rest)
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

const handleSaveSMTP = async () => {
  await updateSettings('smtp', { ...smtpForm, test_email: testEmail.value })
  ElMessage.success('SMTP 配置已保存')
}

const portMap = { starttls: 587, implicit: 465, none: 25 }
const onTLSTypeChange = (val) => {
  if (portMap[val]) smtpForm.port = portMap[val]
}

const handleTestSMTP = async () => {
  testingSMTP.value = true
  try {
    await testSMTP({ to: testEmail.value })
    ElMessage.success('测试邮件已发送，请检查邮箱')
  } catch (e) {
    ElMessage.error('发送失败: ' + (e.response?.data?.message || e.message))
  } finally {
    testingSMTP.value = false
  }
}

watch(() => editForm.invite_limit, checkInviteWarning)

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
.filter-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
  align-items: center;
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
  :deep(.el-table) {
    font-size: 12px;
  }
  :deep(.el-table .cell) {
    padding: 0 6px;
  }
  /* Hide less critical columns on mobile */
  :deep(.el-table__body-wrapper) {
    overflow-x: auto;
  }
  :deep(.el-tabs__item) {
    padding: 0 10px;
    font-size: 13px;
  }
}
</style>
