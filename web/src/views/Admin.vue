<!-- web/src/views/Admin.vue -->
<template>
  <div>
    <div class="page-header">
      <h2>{{ $t('admin.title') }}</h2>
      <p class="subtitle">{{ $t('admin.subtitle') }}</p>
    </div>

    <el-card>
      <el-tabs v-model="activeTab">
        <!-- 用户管理 -->
        <el-tab-pane :label="$t('admin.userManagement')" name="users">
          <el-table :data="users" stripe v-loading="loading" style="width:100%">
            <el-table-column prop="id" :label="$t('admin.id')" width="60" />
            <el-table-column :label="$t('admin.avatar')" width="60">
              <template #default="{ row }">
                <el-avatar v-if="row.avatar" :src="row.avatar" :size="32" />
                <el-avatar v-else :size="32">{{ (row.username || '?')[0]?.toUpperCase() }}</el-avatar>
              </template>
            </el-table-column>
            <el-table-column prop="username" :label="$t('common.username')" min-width="100" />
            <el-table-column prop="email" :label="$t('admin.email')" min-width="140" show-overflow-tooltip />
            <el-table-column prop="role" :label="$t('admin.role')" width="90">
              <template #default="{ row }">
                <el-tag :type="row.is_super_admin ? 'warning' : row.role === 'admin' ? 'danger' : 'info'" size="small">
                  {{ row.is_super_admin ? $t('admin.superAdmin') : row.role === 'admin' ? $t('common.admin') : $t('admin.normalUser') }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="$t('common.status')" width="80">
              <template #default="{ row }">
                <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">
                  {{ row.status === 1 ? $t('admin.statusNormal') : $t('admin.statusDisabled') }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="invite_limit" :label="$t('admin.inviteLimit')" width="90" />
            <el-table-column prop="invite_count" :label="$t('admin.invitedCount')" width="80" />
            <el-table-column prop="created_at" :label="$t('admin.registerTime')" width="170" />
            <el-table-column :label="$t('common.action')" width="200" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="openEditUser(row)">{{ $t('common.edit') }}</el-button>
                <el-button link type="warning" size="small" :disabled="row.is_super_admin && row.id !== auth.user?.id" @click="openResetPwd(row)">{{ $t('admin.resetPassword') }}</el-button>
                <el-button v-if="row.status === 1" link type="danger" size="small" :disabled="row.is_super_admin" @click="handleDisable(row)">{{ $t('admin.disable') }}</el-button>
                <el-button v-else link type="success" size="small" @click="handleEnable(row)">{{ $t('admin.enable') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 域名管理 -->
        <el-tab-pane :label="$t('admin.domainManagement')" name="domains">
          <div class="domain-actions">
            <div style="display:flex;gap:8px;align-items:flex-end;flex-wrap:wrap">
              <el-select v-model="selectedProviderId" :placeholder="$t('admin.selectProvider')" style="width:200px" size="default">
                <el-option v-for="p in adminProviders" :key="p.id" :label="p.name" :value="p.id" />
              </el-select>
              <el-select v-model="selectedDomain" :placeholder="$t('admin.selectDomain')" style="width:240px" size="default" :loading="loadingAdminDomains" :disabled="!selectedProviderId">
                <el-option v-for="d in adminDomains" :key="d.domain_name" :label="getDomainLabel(d)" :value="d.domain_name" />
              </el-select>
              <el-button type="primary" size="small" :disabled="adminProviders.length === 0" @click="handleCreateRoot">{{ $t('admin.createRootDomain') }}</el-button>
            </div>
            <el-empty v-if="adminProviders.length === 0" :description="$t('admin.noProvidersHint')" :image-size="60" />
          </div>

          <el-table :data="allDomains" stripe v-loading="loading" style="width:100%">
            <el-table-column prop="id" :label="$t('admin.id')" width="60" />
            <el-table-column prop="full_domain" :label="$t('admin.fullDomain')" min-width="180" />
            <el-table-column :label="$t('admin.currentOwner')" min-width="120">
              <template #default="{ row }">
                <el-tag size="small">{{ row.owner?.username || row.owner_id }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" :label="$t('common.createdAt')" width="170" />
            <el-table-column :label="$t('common.action')" width="100" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="openAssign(row)">{{ $t('admin.assign') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 操作日志 -->
        <el-tab-pane :label="$t('admin.operationLogs')" name="logs">
          <div class="filter-bar">
            <el-select v-model="logFilters.user_id" :placeholder="$t('admin.user')" clearable filterable size="small" style="width:130px">
              <el-option v-for="u in users" :key="u.id" :label="`${u.nickname || u.username} (@${u.username})`" :value="u.id">
                <div style="display:flex;align-items:center;gap:6px">
                  <el-avatar v-if="u.avatar" :src="u.avatar" :size="20" />
                  <el-avatar v-else :size="20">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
                  <span>{{ u.nickname || u.username }}</span>
                </div>
              </el-option>
            </el-select>
            <el-input v-model="logFilters.q" :placeholder="$t('myLogs.searchPlaceholder')" clearable size="small" style="width:120px" />
            <el-input v-model="logFilters.q_exclude" :placeholder="$t('myLogs.excludePlaceholder')" clearable size="small" style="width:110px" />
            <div class="filter-group">
              <el-select v-model="logFilters.action" :placeholder="$t('myLogs.actionPlaceholder')" clearable filterable multiple collapse-tags collapse-tags-tooltip size="small" style="width:160px">
                <el-option-group v-for="group in actionGroupsI18n" :key="group.label" :label="group.label">
                  <el-option v-for="item in group.options" :key="item.value" :label="item.label" :value="item.value" />
                </el-option-group>
              </el-select>
              <el-tooltip :content="actionExcludeMode ? $t('myLogs.excludeMode') : $t('myLogs.includeMode')">
                <el-button size="small" :type="actionExcludeMode ? 'danger' : ''" @click="actionExcludeMode = !actionExcludeMode; loadLogs()" style="padding: 5px">
                  <el-icon :size="14"><component :is="actionExcludeMode ? 'CircleClose' : 'CircleCheck'" /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
            <div class="filter-group">
              <el-select v-model="logFilters.target_type" :placeholder="$t('myLogs.targetTypePlaceholder')" clearable multiple collapse-tags collapse-tags-tooltip size="small" style="width:160px">
                <el-option v-for="item in targetTypeOptionsI18n" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
              <el-tooltip :content="targetTypeExcludeMode ? $t('myLogs.excludeMode') : $t('myLogs.includeMode')">
                <el-button size="small" :type="targetTypeExcludeMode ? 'danger' : ''" @click="targetTypeExcludeMode = !targetTypeExcludeMode; loadLogs()" style="padding: 5px">
                  <el-icon :size="14"><component :is="targetTypeExcludeMode ? 'CircleClose' : 'CircleCheck'" /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
            <el-date-picker v-model="logFilters.dateRange" type="daterange" :range-separator="$t('myLogs.dateRangeSeparator')" :start-placeholder="$t('myLogs.startDatePlaceholder')" :end-placeholder="$t('myLogs.endDatePlaceholder')" size="small" style="width:220px" value-format="YYYY-MM-DD" />
            <div class="filter-actions">
              <el-button size="small" type="primary" @click="loadLogs">{{ $t('common.search') }}</el-button>
              <el-button size="small" @click="resetLogFilters">{{ $t('common.reset') }}</el-button>
            </div>
          </div>
          <el-table :data="logs" stripe v-loading="loading" style="width:100%">
            <el-table-column prop="id" :label="$t('admin.id')" width="60" />
            <el-table-column :label="$t('admin.user')" min-width="120">
              <template #default="{ row }">
                <div style="display:flex;align-items:center;gap:6px">
                  <el-avatar v-if="row.user?.avatar" :src="row.user.avatar" :size="24" />
                  <el-avatar v-else :size="24">{{ (row.user?.username || '?')[0]?.toUpperCase() }}</el-avatar>
                  <span>{{ row.user?.username || 'User #' + row.user_id }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="action" :label="$t('common.action')" min-width="120" />
            <el-table-column prop="target_type" :label="$t('admin.targetTypeLabel')" min-width="100" />
            <el-table-column prop="ip_address" :label="$t('common.ipAddress')" width="130" />
            <el-table-column :label="$t('common.detail')" min-width="200">
              <template #default="{ row }">
                <template v-if="row.detail && row.detail !== 'null'">
                  <el-tag v-for="(val, key) in parseDetail(row.detail)" :key="key" size="small" style="margin:2px" type="info">
                    {{ formatDetailKey(key) }}: {{ formatDetailValue(val) }}
                  </el-tag>
                </template>
                <span v-else style="color:#909399">-</span>
              </template>
            </el-table-column>
            <el-table-column prop="created_at" :label="$t('common.time')" width="170" />
          </el-table>
          <el-pagination v-if="logTotal > 0" :current-page="logPage" :page-size="logPageSize" :total="logTotal"
            layout="total, sizes, prev, pager, next" :page-sizes="[20, 50, 100]"
            @current-change="handleLogPageChange" @size-change="handleLogSizeChange" style="margin-top:16px" />
        </el-tab-pane>
        <el-tab-pane :label="$t('admin.systemSettings')" name="settings">
          <el-form :model="smtpForm" label-width="100px" style="max-width:560px">
            <el-divider content-position="left">{{ $t('admin.smtpConfig') }}</el-divider>
            <el-form-item :label="$t('admin.smtpHost')">
              <el-input v-model="smtpForm.host" :placeholder="$t('admin.smtpHostPlaceholder')" />
            </el-form-item>
            <el-form-item :label="$t('admin.smtpPort')">
              <el-input-number v-model="smtpForm.port" :min="1" :max="65535" />
            </el-form-item>
            <el-form-item :label="$t('admin.encryption')">
              <el-select v-model="smtpForm.tls_type" style="width:100%" @change="onTLSTypeChange">
                <el-option :label="$t('admin.tlsStarttls')" value="starttls" />
                <el-option :label="$t('admin.tlsImplicit')" value="implicit" />
                <el-option :label="$t('admin.tlsNone')" value="none" />
              </el-select>
            </el-form-item>
            <el-form-item :label="$t('admin.loginEmail')">
              <el-input v-model="smtpForm.username" :placeholder="$t('admin.loginEmailPlaceholder')" autocomplete="off" />
              <div class="el-form-item__tip" style="color:#909399;font-size:12px;margin-top:4px">{{ $t('admin.loginEmailTip') }}</div>
            </el-form-item>
            <el-form-item :label="$t('admin.authorizationCode')">
              <el-input v-model="smtpForm.password" type="password" show-password :placeholder="$t('admin.authCodePlaceholder')" autocomplete="new-password" />
            </el-form-item>
            <el-form-item :label="$t('admin.senderAddress')">
              <el-input v-model="smtpForm.from" :placeholder="$t('admin.senderAddressPlaceholder')" />
              <div class="el-form-item__tip" style="color:#909399;font-size:12px;margin-top:4px">{{ $t('admin.senderAddressTip') }}</div>
            </el-form-item>
            <el-form-item :label="$t('admin.senderName')">
              <el-input v-model="smtpForm.from_name" placeholder="DomainNest" />
            </el-form-item>
            <el-form-item :label="$t('admin.testRecipient')">
              <el-input v-model="testEmail" :placeholder="$t('admin.testRecipientPlaceholder')" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleSaveSMTP">{{ $t('admin.saveConfig') }}</el-button>
              <el-button @click="handleTestSMTP" :loading="testingSMTP" :disabled="!testEmail">{{ $t('admin.sendTestEmail') }}</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 分配域名对话框 -->
    <el-dialog v-model="showAssign" :title="$t('admin.assignDomain')" width="420px" destroy-on-close>
      <p class="assign-info">
        {{ $t('admin.assignTo', { domain: assignTarget?.full_domain }) }}
      </p>
      <el-select v-model="assignUserId" :placeholder="$t('admin.selectUser')" style="width:100%">
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
        <el-button @click="showAssign = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleAssign">{{ $t('admin.confirmAssign') }}</el-button>
      </template>
    </el-dialog>

    <!-- 编辑用户对话框 -->
    <el-dialog v-model="showEditUser" :title="$t('admin.editUser')" width="480px" destroy-on-close>
      <el-form :model="editForm" label-width="80px">
        <el-form-item :label="$t('common.username')">
          <el-input v-model="editForm.username" />
        </el-form-item>
        <el-form-item :label="$t('admin.role')">
          <el-select v-model="editForm.role" style="width:100%" :disabled="editTarget?.is_super_admin">
            <el-option :label="$t('admin.normalUser')" value="user" />
            <el-option :label="$t('common.admin')" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('common.status')">
          <el-select v-model="editForm.status" style="width:100%">
            <el-option :label="$t('admin.statusNormal')" :value="1" />
            <el-option :label="$t('admin.statusDisabled')" :value="0" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('admin.inviteLimit')">
          <el-input-number v-model="editForm.invite_limit" :min="0" :max="9999" />
          <div v-if="editInviteWarning" style="color:#e6a23c;font-size:12px;margin-top:4px">
            {{ editInviteWarning }}
          </div>
        </el-form-item>
        <el-form-item :label="$t('admin.nickname')">
          <el-input v-model="editForm.nickname" />
        </el-form-item>
        <el-form-item :label="$t('admin.email')">
          <el-input v-model="editForm.email" />
        </el-form-item>
        <el-form-item :label="$t('admin.phone')">
          <el-input v-model="editForm.phone" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditUser = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleEditUser">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- 重置密码对话框 -->
    <el-dialog v-model="showResetPwd" :title="$t('admin.resetPasswordTitle')" width="400px" destroy-on-close>
      <p style="margin-bottom:12px">{{ $t('admin.setNewPassword', { username: resetPwdTarget?.username }) }}</p>
      <el-input v-model="newPassword" type="password" :placeholder="$t('admin.newPasswordPlaceholder')" show-password />
      <template #footer>
        <el-button @click="showResetPwd = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="warning" @click="handleResetPwd">{{ $t('admin.confirmReset') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { listUsers, listLogs, listDomains, createRootDomain, assignDomain, updateUser, adminResetPassword, disableUser, getSettings, updateSettings, testSMTP, promoteToAdmin, demoteFromAdmin } from '../api/admin'
import { listProviders, listProviderDomains } from '../api/provider'
import { useAuthStore } from '../stores/auth'
import { ElMessage, ElMessageBox } from 'element-plus'
import { actionGroups, targetTypeOptions } from '../constants/operationLogs'

const auth = useAuthStore()
const { t } = useI18n()

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

const logFilters = reactive({ user_id: '', action: [], target_type: [], q: '', q_exclude: '', dateRange: null })
const actionExcludeMode = ref(false)
const targetTypeExcludeMode = ref(false)

const DETAIL_KEY_I18N = {
  username: 'common.username',
  full_domain: 'admin.detailKeyFullDomain',
  host: 'admin.detailKeyHost',
  type: 'admin.detailKeyType',
  value: 'admin.detailKeyValue',
  amount: 'admin.detailKeyAmount',
  enabled: 'admin.detailKeyStatus',
  ids: 'admin.detailKeyIds',
  target_user_id: 'admin.detailKeyTargetUser',
  invited_by: 'admin.detailKeyInvitedBy',
  provider_name: 'admin.detailKeyProvider',
}

const parseDetail = (json) => {
  try { return JSON.parse(json) } catch { return {} }
}

const formatDetailKey = (key) => {
  const i18nKey = DETAIL_KEY_I18N[key]
  return i18nKey ? t(i18nKey) : key
}

const formatDetailValue = (val) => {
  if (Array.isArray(val)) return val.join(', ')
  if (typeof val === 'boolean') return t(val ? 'common.enabled' : 'common.disabled')
  return String(val)
}

const getDomainLabel = (d) => d.domain_name + (d.record_count != null ? ` (${d.record_count} ${t('admin.recordCount')})` : '')

const GROUP_I18N_KEYS = {
  '域名': 'admin.actionGroupDomain',
  '记录': 'admin.actionGroupRecord',
  '权限': 'admin.actionGroupPermission',
  '用户': 'admin.actionGroupUser',
  '邀请': 'admin.actionGroupInvite',
  '好友': 'admin.actionGroupFriend',
  '提供商': 'admin.actionGroupProvider',
  '设置': 'admin.actionGroupSetting',
}

const ACTION_I18N_KEYS = {
  create_domain: 'admin.actionCreateDomain',
  transfer_domain: 'admin.actionTransferDomain',
  delete_domain: 'admin.actionDeleteDomain',
  create_root_domain: 'admin.actionCreateRootDomain',
  assign_domain: 'admin.actionAssignDomain',
  create_record: 'admin.actionCreateRecord',
  update_record: 'admin.actionUpdateRecord',
  delete_record: 'admin.actionDeleteRecord',
  toggle_record: 'admin.actionToggleRecord',
  batch_delete: 'admin.actionBatchDelete',
  batch_toggle: 'admin.actionBatchToggle',
  import: 'admin.actionImport',
  grant_permission: 'admin.actionGrantPermission',
  revoke_permission: 'admin.actionRevokePermission',
  revoke_request: 'admin.actionRevokeRequest',
  accept_return: 'admin.actionAcceptReturn',
  reject_return: 'admin.actionRejectReturn',
  assign_pending_records: 'admin.actionAssignPendingRecords',
  delete_pending_records: 'admin.actionDeletePendingRecords',
  register: 'admin.actionRegister',
  login: 'admin.actionLogin',
  update_profile: 'admin.actionUpdateProfile',
  change_password: 'admin.actionChangePassword',
  reset_token: 'admin.actionResetToken',
  upload_avatar: 'admin.actionUploadAvatar',
  delete_account: 'admin.actionDeleteAccount',
  update_user: 'admin.actionUpdateUser',
  admin_reset_password: 'admin.actionAdminResetPassword',
  disable_user: 'admin.actionDisableUser',
  promote_to_admin: 'admin.actionPromoteToAdmin',
  demote_from_admin: 'admin.actionDemoteFromAdmin',
  grant_invite: 'admin.actionGrantInvite',
  revoke_invite: 'admin.actionRevokeInvite',
  admin_grant: 'admin.actionAdminGrant',
  send_friend_request: 'admin.actionSendFriendRequest',
  accept_friend: 'admin.actionAcceptFriend',
  reject_friend: 'admin.actionRejectFriend',
  remove_friend: 'admin.actionRemoveFriend',
  create_provider: 'admin.actionCreateProvider',
  update_provider: 'admin.actionUpdateProvider',
  delete_provider: 'admin.actionDeleteProvider',
  claim_domain: 'admin.actionClaimDomain',
  update_settings: 'admin.actionUpdateSettings',
  smtp_test: 'admin.actionSmtpTest',
}

const TARGET_I18N_KEYS = {
  domain_node: 'admin.targetDomainNode',
  dns_record: 'admin.targetDnsRecord',
  user: 'admin.targetUser',
  setting: 'admin.targetSetting',
  provider: 'admin.targetProvider',
  friend: 'admin.targetFriend',
}

const actionGroupsI18n = computed(() => {
  return actionGroups.map(group => ({
    ...group,
    label: t(GROUP_I18N_KEYS[group.label] || group.label),
    options: group.options.map(opt => ({
      ...opt,
      label: t(ACTION_I18N_KEYS[opt.value] || opt.label)
    }))
  }))
})

const targetTypeOptionsI18n = computed(() => {
  return targetTypeOptions.map(opt => ({
    ...opt,
    label: t(TARGET_I18N_KEYS[opt.value] || opt.label)
  }))
})
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
    ElMessage.error(e.response?.data?.message || t('admin.fetchDomainsFailed'))
  } finally {
    loadingAdminDomains.value = false
  }
})

const loadLogs = async () => {
  const params = { page: logPage.value, page_size: logPageSize.value }
  if (logFilters.user_id) params.user_id = logFilters.user_id
  if (logFilters.action.length > 0) {
    if (actionExcludeMode.value) {
      params.action_exclude = logFilters.action.join(',')
    } else {
      params.action = logFilters.action.join(',')
    }
  }
  if (logFilters.target_type.length > 0) {
    if (targetTypeExcludeMode.value) {
      params.target_type_exclude = logFilters.target_type.join(',')
    } else {
      params.target_type = logFilters.target_type.join(',')
    }
  }
  if (logFilters.q) params.q = logFilters.q
  if (logFilters.q_exclude) params.q_exclude = logFilters.q_exclude
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
  logFilters.action = []
  logFilters.target_type = []
  logFilters.q = ''
  logFilters.q_exclude = ''
  logFilters.dateRange = null
  actionExcludeMode.value = false
  targetTypeExcludeMode.value = false
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
    ElMessage.warning(t('admin.selectProviderAndDomain'))
    return
  }
  await createRootDomain({ provider_id: selectedProviderId.value, domain_name: selectedDomain.value })
  ElMessage.success(t('admin.rootDomainCreated'))
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
    ElMessage.warning(t('admin.selectTargetUser'))
    return
  }
  await assignDomain(assignTarget.value.id, { user_id: assignUserId.value })
  ElMessage.success(t('admin.domainAssigned'))
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
        editInviteWarning.value = t('admin.inviteQuotaInsufficient', { available, additional })
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
  ElMessage.success(t('admin.userUpdated'))
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
    ElMessage.warning(t('admin.passwordMinLength'))
    return
  }
  await adminResetPassword(resetPwdTarget.value.id, { new_password: newPassword.value })
  ElMessage.success(t('admin.passwordReset'))
  showResetPwd.value = false
}

const handleDisable = async (row) => {
  await ElMessageBox.confirm(t('admin.confirmDisableUser', { username: row.username }), t('common.confirm'), { type: 'warning' })
  await disableUser(row.id)
  ElMessage.success(t('admin.userDisabled'))
  loadData()
}

const handleEnable = async (row) => {
  await updateUser(row.id, { status: 1 })
  ElMessage.success(t('admin.userEnabled'))
  loadData()
}

const handleSaveSMTP = async () => {
  await updateSettings('smtp', { ...smtpForm, test_email: testEmail.value })
  ElMessage.success(t('admin.smtpConfigSaved'))
}

const portMap = { starttls: 587, implicit: 465, none: 25 }
const onTLSTypeChange = (val) => {
  if (portMap[val]) smtpForm.port = portMap[val]
}

const handleTestSMTP = async () => {
  testingSMTP.value = true
  try {
    await testSMTP({ to: testEmail.value })
    ElMessage.success(t('admin.testEmailSent'))
  } catch (e) {
    ElMessage.error(t('admin.sendFailed') + (e.response?.data?.message || e.message))
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
.filter-group {
  display: flex;
  align-items: center;
  gap: 2px;
}
.filter-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
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
