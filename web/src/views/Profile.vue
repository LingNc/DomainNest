<template>
  <div class="profile-page">
    <el-card>
      <template #header>
        <span>{{ $t('profile.title') }}</span>
      </template>
      <el-form :model="form" label-width="80px" style="max-width:100%">
        <el-form-item :label="$t('profile.avatar')">
          <div class="avatar-section">
            <el-upload
              class="avatar-uploader"
              :show-file-list="false"
              :before-upload="beforeAvatarUpload"
              :http-request="handleAvatarUpload"
              accept="image/jpeg,image/png"
            >
              <el-avatar v-if="form.avatar" :src="form.avatar" :size="64" />
              <el-avatar v-else :size="64" style="cursor:pointer">{{ (form.username || '?')[0]?.toUpperCase() }}</el-avatar>
            </el-upload>
            <span class="avatar-hint">{{ $t('profile.avatarHint') }}</span>
          </div>
        </el-form-item>
        <el-form-item :label="$t('profile.id')">
          <el-input :model-value="profile.id" disabled />
        </el-form-item>
        <el-form-item :label="$t('common.username')">
          <el-input v-model="form.username" :placeholder="$t('profile.usernamePlaceholder')" @input="onUsernameInput" />
          <div v-if="usernameStatus" :class="usernameStatus === 'available' ? 'username-ok' : 'username-err'">
            {{ usernameStatus === 'available' ? $t('profile.usernameAvailable') : $t('profile.usernameTaken') }}
          </div>
        </el-form-item>
        <el-form-item :label="$t('profile.role')">
          <el-tag :type="profile.is_super_admin ? 'warning' : profile.role === 'admin' ? 'danger' : 'info'">{{ roleLabel(profile) }}</el-tag>
        </el-form-item>
        <el-form-item :label="$t('profile.nickname')">
          <el-input v-model="form.nickname" :placeholder="$t('profile.nicknamePlaceholder')" />
        </el-form-item>
        <el-form-item :label="$t('profile.email')">
          <el-input v-model="form.email" :placeholder="$t('profile.emailPlaceholder')" />
        </el-form-item>
        <el-form-item :label="$t('profile.phone')">
          <el-input v-model="form.phone" :placeholder="$t('profile.phonePlaceholder')" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="handleSave">{{ $t('profile.save') }}</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card style="margin-top:16px">
      <template #header>
        <span>{{ $t('profile.inviteInfo') }}</span>
      </template>
      <el-descriptions :column="1" border>
        <el-descriptions-item :label="$t('profile.myInviteCode')">
          <span class="code-text">{{ profile.invite_code }}</span>
          <el-button size="small" text type="primary" @click="copyCode" style="margin-left:8px">{{ $t('common.copy') }}</el-button>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('profile.inviteLimit')">{{ profile.invite_limit }}</el-descriptions-item>
        <el-descriptions-item :label="$t('profile.assigned')">{{ profile.invite_count }}</el-descriptions-item>
        <el-descriptions-item :label="$t('profile.availableQuota')">
          <el-tag :type="(profile.invite_limit - profile.invite_count) > 0 ? 'success' : 'danger'">
            {{ profile.invite_limit - profile.invite_count }}
          </el-tag>
        </el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card style="margin-top:16px">
      <template #header>
        <span>{{ $t('profile.grantInvite') }}</span>
      </template>
      <el-form :model="grantForm" label-width="80px" style="max-width:480px">
        <el-form-item :label="$t('profile.targetUser')">
          <el-select v-model="grantForm.target_user_id" :placeholder="$t('profile.searchUser')" filterable remote :remote-method="searchUsersRemote" :loading="searchingUsers" style="width:100%">
            <el-option v-for="u in selectableUsers" :key="u.id" :label="`${u.nickname || u.username} (@${u.username})`" :value="u.id">
              <div style="display:flex;align-items:center;gap:8px">
                <el-avatar v-if="u.avatar" :src="u.avatar" :size="24" />
                <el-avatar v-else :size="24">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
                <span>{{ u.nickname || u.username }}</span>
                <span style="color:#909399;font-size:12px">@{{ u.username }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('profile.quantity')">
          <el-input-number v-model="grantForm.amount" :min="1" :max="auth.isSuperAdmin ? undefined : profile.invite_limit - profile.invite_count" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="granting" @click="handleGrantInvite">{{ $t('profile.grant') }}</el-button>
          <span style="color:#909399;font-size:12px;margin-left:12px">{{ $t('profile.grantHint') }}</span>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card style="margin-top:16px">
      <template #header>
        <span>{{ $t('profile.revokeInvite') }}</span>
      </template>
      <el-form :model="revokeForm" label-width="80px" style="max-width:480px">
        <el-form-item :label="$t('profile.targetUser')">
          <el-select v-model="revokeForm.target_user_id" :placeholder="$t('profile.searchUser')" filterable remote :remote-method="searchUsersRemote" :loading="searchingUsers" style="width:100%">
            <el-option v-for="u in selectableUsers" :key="u.id" :label="`${u.nickname || u.username} (@${u.username})`" :value="u.id">
              <div style="display:flex;align-items:center;gap:8px">
                <el-avatar v-if="u.avatar" :src="u.avatar" :size="24" />
                <el-avatar v-else :size="24">{{ (u.username || '?')[0]?.toUpperCase() }}</el-avatar>
                <span>{{ u.nickname || u.username }}</span>
                <span style="color:#909399;font-size:12px">@{{ u.username }}</span>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('profile.quantity')">
          <el-input-number v-model="revokeForm.amount" :min="1" />
        </el-form-item>
        <el-form-item>
          <el-button type="danger" :loading="revoking" @click="handleRevokeInvite">{{ $t('profile.revoke') }}</el-button>
          <span style="color:#909399;font-size:12px;margin-left:12px">{{ $t('profile.revokeHint') }}</span>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card style="margin-top:16px">
      <template #header>
        <span>{{ $t('profile.inviteLogs') }}</span>
      </template>
      <el-table :data="inviteLogs" stripe v-loading="loadingInviteLogs" style="width:100%">
        <el-table-column :label="$t('profile.type')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.action === 'register' ? 'success' : row.action === 'grant' ? 'primary' : row.action === 'revoke' ? 'danger' : 'warning'" size="small">
              {{ actionLabel(row.action) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="$t('profile.target')" min-width="120">
          <template #default="{ row }">
            <span v-if="row.inviter_id === profile.id">{{ row.invitee?.username || 'User #' + row.invitee_id }}</span>
            <span v-else>{{ row.inviter?.username || 'User #' + row.inviter_id }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="amount" :label="$t('profile.quantity')" width="80" />
        <el-table-column prop="created_at" :label="$t('profile.time')" width="170" />
      </el-table>
      <el-pagination v-if="inviteLogTotal > 10" :current-page="inviteLogPage" :page-size="10" :total="inviteLogTotal"
        layout="total, prev, pager, next" @current-change="handleInviteLogPageChange" style="margin-top:12px" />
    </el-card>

    <el-card style="margin-top:16px">
      <template #header>
        <span>{{ $t('profile.changePassword') }}</span>
      </template>
      <el-form :model="pwdForm" label-width="80px" style="max-width:480px">
        <el-form-item :label="$t('profile.oldPassword')">
          <el-input v-model="pwdForm.old_password" type="password" show-password autocomplete="current-password" />
        </el-form-item>
        <el-form-item :label="$t('profile.newPassword')">
          <el-input v-model="pwdForm.new_password" type="password" show-password autocomplete="new-password" />
        </el-form-item>
        <el-form-item>
          <el-button type="warning" :loading="changingPwd" @click="handleChangePwd">{{ $t('profile.changePassword') }}</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card style="margin-top:16px">
      <template #header>
        <span>{{ $t('profile.ddnsToken') }}</span>
      </template>
      <el-descriptions :column="1" border>
        <el-descriptions-item :label="$t('profile.token')">
          <span class="code-text">{{ profile.ddns_token }}</span>
          <el-button size="small" text type="primary" @click="copyToken" style="margin-left:8px">{{ $t('common.copy') }}</el-button>
        </el-descriptions-item>
      </el-descriptions>
      <el-button type="danger" size="small" style="margin-top:12px" @click="handleResetToken">{{ $t('profile.tokenReset') }}</el-button>
    </el-card>

    <el-card v-if="!auth.isSuperAdmin" style="margin-top:16px">
      <template #header>
        <span>{{ $t('profile.deleteAccount') }}</span>
      </template>
      <p style="color:#f56c6c;font-size:13px;margin-bottom:12px">
        {{ $t('profile.deleteWarning') }}
      </p>
      <el-button type="danger" :loading="deleting" @click="handleDeleteAccount">{{ $t('profile.deleteAccount') }}</el-button>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getProfile, updateProfile, changePassword, resetToken, checkUsername, uploadAvatar, grantInviteQuota, revokeInviteQuota, getInviteLogs, searchAllUsers, deleteAccount } from '../api/auth'
import { useAuthStore } from '../stores/auth'
import { ElMessage, ElMessageBox } from 'element-plus'

const { t } = useI18n()
const auth = useAuthStore()

const roleLabel = (p) => {
  if (p.is_super_admin) return t('profile.superAdmin')
  if (p.role === 'admin') return t('common.admin')
  return t('profile.user')
}

const actionLabel = (action) => {
  const map = { register: 'profile.register', grant: 'profile.grantAction', admin_grant: 'profile.adminGrant', revoke: 'profile.revokeAction' }
  return t(map[action]) || action
}
const router = useRouter()
const profile = ref({})
const saving = ref(false)
const changingPwd = ref(false)
const usernameStatus = ref('')
let checkTimer = null
const form = reactive({ username: '', nickname: '', email: '', phone: '', avatar: '' })
const pwdForm = reactive({ old_password: '', new_password: '' })

const grantForm = reactive({ target_user_id: null, amount: 1 })
const granting = ref(false)
const revokeForm = reactive({ target_user_id: null, amount: 1 })
const revoking = ref(false)
const selectableUsers = ref([])
const searchingUsers = ref(false)
const inviteLogs = ref([])
const loadingInviteLogs = ref(false)
const inviteLogPage = ref(1)
const inviteLogTotal = ref(0)
const deleting = ref(false)

const loadProfile = async () => {
  const res = await getProfile()
  profile.value = res.data
  form.username = res.data.username || ''
  form.nickname = res.data.nickname || ''
  form.email = res.data.email || ''
  form.phone = res.data.phone || ''
  form.avatar = res.data.avatar || ''
}

const loadInviteLogs = async () => {
  loadingInviteLogs.value = true
  try {
    const res = await getInviteLogs({ page: inviteLogPage.value, page_size: 10 })
    inviteLogs.value = res.data.items || []
    inviteLogTotal.value = res.data.total || 0
  } finally {
    loadingInviteLogs.value = false
  }
}

onMounted(() => {
  loadProfile()
  loadInviteLogs()
})

const onUsernameInput = () => {
  usernameStatus.value = ''
  clearTimeout(checkTimer)
  const name = form.username.trim()
  if (name.length < 3 || name === profile.value.username) return
  checkTimer = setTimeout(async () => {
    try {
      const res = await checkUsername(name)
      usernameStatus.value = res.data.available ? 'available' : 'taken'
    } catch { /* ignore */ }
  }, 400)
}

const handleSave = async () => {
  if (usernameStatus.value === 'taken') {
    ElMessage.warning(t('profile.usernameTaken'))
    return
  }
  saving.value = true
  try {
    await updateProfile(form)
    ElMessage.success(t('common.saved'))
    await loadProfile()
    // Update auth store with new profile data
    auth.setAuth(auth.token, { ...auth.user, username: form.username, nickname: form.nickname, avatar: form.avatar })
  } finally {
    saving.value = false
  }
}

const handleChangePwd = async () => {
  if (!pwdForm.old_password || !pwdForm.new_password) {
    ElMessage.warning(t('profile.pleaseFillAll'))
    return
  }
  if (pwdForm.new_password.length < 6) {
    ElMessage.warning(t('profile.passwordMinLength'))
    return
  }
  changingPwd.value = true
  try {
    await changePassword(pwdForm)
    ElMessage.success(t('profile.passwordChanged'))
    pwdForm.old_password = ''
    pwdForm.new_password = ''
  } finally {
    changingPwd.value = false
  }
}

const handleResetToken = async () => {
  try {
    const res = await resetToken()
    profile.value.ddns_token = res.data.token
    ElMessage.success(t('profile.tokenReset'))
  } catch {}
}

const copyCode = () => {
  navigator.clipboard.writeText(profile.value.invite_code)
  ElMessage.success(t('common.copied'))
}

const copyToken = () => {
  navigator.clipboard.writeText(profile.value.ddns_token)
  ElMessage.success(t('common.copied'))
}

const searchUsersRemote = async (query) => {
  if (!query) { selectableUsers.value = []; return }
  searchingUsers.value = true
  try {
    const res = await searchAllUsers(query)
    selectableUsers.value = res.data || []
  } catch { /* ignore */ }
  searchingUsers.value = false
}

const handleGrantInvite = async () => {
  if (!grantForm.target_user_id) {
    ElMessage.warning(t('profile.inputTargetUserId'))
    return
  }
  if (!auth.isSuperAdmin) {
    const available = profile.value.invite_limit - profile.value.invite_count
    if (grantForm.amount > available) {
      ElMessage.warning(t('profile.exceedQuota'))
      return
    }
  }
  granting.value = true
  try {
    await grantInviteQuota(grantForm)
    ElMessage.success(t('profile.grantSuccess'))
    grantForm.target_user_id = null
    grantForm.amount = 1
    await loadProfile()
    await loadInviteLogs()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || t('profile.grantFailed'))
  } finally {
    granting.value = false
  }
}

const handleRevokeInvite = async () => {
  if (!revokeForm.target_user_id) {
    ElMessage.warning(t('profile.selectTargetUser'))
    return
  }
  revoking.value = true
  try {
    await revokeInviteQuota(revokeForm)
    ElMessage.success(t('profile.revokeSuccess'))
    revokeForm.target_user_id = null
    revokeForm.amount = 1
    await loadProfile()
    await loadInviteLogs()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || t('profile.revokeFailed'))
  } finally {
    revoking.value = false
  }
}

const handleInviteLogPageChange = (p) => {
  inviteLogPage.value = p
  loadInviteLogs()
}

const beforeAvatarUpload = (file) => {
  const isImage = ['image/jpeg', 'image/png'].includes(file.type)
  const isLt2M = file.size / 1024 / 1024 < 2
  if (!isImage) ElMessage.error(t('profile.avatarFormatError'))
  if (!isLt2M) ElMessage.error(t('profile.avatarSizeError'))
  return isImage && isLt2M
}

const handleAvatarUpload = async ({ file }) => {
  const fd = new FormData()
  fd.append('file', file)
  try {
    const res = await uploadAvatar(fd)
    form.avatar = res.data.avatar
    auth.setAuth(auth.token, { ...auth.user, avatar: res.data.avatar })
    ElMessage.success(t('profile.avatarUploadSuccess'))
  } catch (e) {
    ElMessage.error(t('profile.uploadFailed') + ': ' + (e.response?.data?.message || e.message))
  }
}

const handleDeleteAccount = async () => {
  try {
    await ElMessageBox.confirm(t('profile.confirmDelete'), t('profile.confirmDeleteTitle'), {
      confirmButtonText: t('profile.confirmDeleteButton'),
      cancelButtonText: t('common.cancel'),
      type: 'warning',
    })
  } catch { return }
  deleting.value = true
  try {
    await deleteAccount()
    ElMessage.success(t('profile.accountDeleted'))
    auth.clearAuth()
    router.push('/login')
  } catch (e) {
    ElMessage.error(e.response?.data?.message || t('profile.deleteFailed'))
  } finally {
    deleting.value = false
  }
}
</script>

<style scoped>
.profile-page {
  max-width: 100%;
}
@media (max-width: 768px) {
  .profile-page {
    max-width: 100%;
  }
}
.code-text {
  font-family: monospace;
  font-size: 14px;
  color: #409eff;
}
.username-ok {
  color: #67c23a;
  font-size: 12px;
  margin-top: 4px;
}
.username-err {
  color: #f56c6c;
  font-size: 12px;
  margin-top: 4px;
}
.avatar-section {
  display: flex;
  align-items: center;
  gap: 12px;
}
.avatar-uploader :deep(.el-upload) {
  cursor: pointer;
}
.avatar-hint {
  color: #909399;
  font-size: 12px;
}
</style>
