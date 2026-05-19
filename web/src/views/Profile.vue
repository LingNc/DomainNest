<template>
  <div class="profile-page">
    <el-card>
      <template #header>
        <span>个人信息</span>
      </template>
      <el-form :model="form" label-width="80px" style="max-width:100%">
        <el-form-item label="头像">
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
            <span class="avatar-hint">点击头像上传，支持 JPG/PNG，自动裁剪为 128px</span>
          </div>
        </el-form-item>
        <el-form-item label="ID">
          <el-input :model-value="profile.id" disabled />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="form.username" placeholder="修改用户名" @input="onUsernameInput" />
          <div v-if="usernameStatus" :class="usernameStatus === 'available' ? 'username-ok' : 'username-err'">
            {{ usernameStatus === 'available' ? '用户名可用' : '用户名已被占用' }}
          </div>
        </el-form-item>
        <el-form-item label="角色">
          <el-tag :type="profile.is_super_admin ? 'warning' : profile.role === 'admin' ? 'danger' : 'info'">{{ profile.is_super_admin ? '超级管理员' : profile.role === 'admin' ? '管理员' : '普通用户' }}</el-tag>
        </el-form-item>
        <el-form-item label="昵称">
          <el-input v-model="form.nickname" placeholder="设置昵称" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="form.email" placeholder="设置邮箱" />
        </el-form-item>
        <el-form-item label="手机">
          <el-input v-model="form.phone" placeholder="设置手机号" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="handleSave">保存修改</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card style="margin-top:16px">
      <template #header>
        <span>邀请信息</span>
      </template>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="我的邀请码">
          <span class="code-text">{{ profile.invite_code }}</span>
          <el-button size="small" text type="primary" @click="copyCode" style="margin-left:8px">复制</el-button>
        </el-descriptions-item>
        <el-descriptions-item label="邀请上限">{{ profile.invite_limit }}</el-descriptions-item>
        <el-descriptions-item label="已分配">{{ profile.invite_count }}</el-descriptions-item>
        <el-descriptions-item label="可用额度">
          <el-tag :type="(profile.invite_limit - profile.invite_count) > 0 ? 'success' : 'danger'">
            {{ profile.invite_limit - profile.invite_count }}
          </el-tag>
        </el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card style="margin-top:16px">
      <template #header>
        <span>分配邀请额度</span>
      </template>
      <el-form :model="grantForm" label-width="80px" style="max-width:480px">
        <el-form-item label="目标用户">
          <el-select v-model="grantForm.target_user_id" placeholder="搜索用户" filterable remote :remote-method="searchUsersRemote" :loading="searchingUsers" style="width:100%">
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
        <el-form-item label="数量">
          <el-input-number v-model="grantForm.amount" :min="1" :max="profile.invite_limit - profile.invite_count" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="granting" @click="handleGrantInvite">分配</el-button>
          <span style="color:#909399;font-size:12px;margin-left:12px">从你的可用额度中分配给其他用户</span>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card style="margin-top:16px">
      <template #header>
        <span>收回邀请额度</span>
      </template>
      <el-form :model="revokeForm" label-width="80px" style="max-width:480px">
        <el-form-item label="目标用户">
          <el-select v-model="revokeForm.target_user_id" placeholder="搜索用户" filterable remote :remote-method="searchUsersRemote" :loading="searchingUsers" style="width:100%">
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
        <el-form-item label="数量">
          <el-input-number v-model="revokeForm.amount" :min="1" />
        </el-form-item>
        <el-form-item>
          <el-button type="danger" :loading="revoking" @click="handleRevokeInvite">收回</el-button>
          <span style="color:#909399;font-size:12px;margin-left:12px">收回对方未使用的邀请额度</span>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card style="margin-top:16px">
      <template #header>
        <span>邀请记录</span>
      </template>
      <el-table :data="inviteLogs" stripe v-loading="loadingInviteLogs" style="width:100%">
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="row.action === 'register' ? 'success' : row.action === 'grant' ? 'primary' : row.action === 'revoke' ? 'danger' : 'warning'" size="small">
              {{ {register: '注册', grant: '分配', admin_grant: '管理分配', revoke: '收回'}[row.action] || row.action }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="对象" min-width="120">
          <template #default="{ row }">
            <span v-if="row.inviter_id === profile.id">{{ row.invitee?.username || 'User #' + row.invitee_id }}</span>
            <span v-else>{{ row.inviter?.username || 'User #' + row.inviter_id }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="amount" label="数量" width="80" />
        <el-table-column prop="created_at" label="时间" width="170" />
      </el-table>
      <el-pagination v-if="inviteLogTotal > 10" :current-page="inviteLogPage" :page-size="10" :total="inviteLogTotal"
        layout="total, prev, pager, next" @current-change="handleInviteLogPageChange" style="margin-top:12px" />
    </el-card>

    <el-card style="margin-top:16px">
      <template #header>
        <span>修改密码</span>
      </template>
      <el-form :model="pwdForm" label-width="80px" style="max-width:480px">
        <el-form-item label="旧密码">
          <el-input v-model="pwdForm.old_password" type="password" show-password autocomplete="current-password" />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="pwdForm.new_password" type="password" show-password autocomplete="new-password" />
        </el-form-item>
        <el-form-item>
          <el-button type="warning" :loading="changingPwd" @click="handleChangePwd">修改密码</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card style="margin-top:16px">
      <template #header>
        <span>DDNS Token</span>
      </template>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="Token">
          <span class="code-text">{{ profile.ddns_token }}</span>
          <el-button size="small" text type="primary" @click="copyToken" style="margin-left:8px">复制</el-button>
        </el-descriptions-item>
      </el-descriptions>
      <el-button type="danger" size="small" style="margin-top:12px" @click="handleResetToken">重置 Token</el-button>
    </el-card>

    <el-card style="margin-top:16px">
      <template #header>
        <span>注销账号</span>
      </template>
      <p style="color:#f56c6c;font-size:13px;margin-bottom:12px">
        注销后，您的域名和权限将转移给邀请您的人，此操作不可撤销。
      </p>
      <el-button type="danger" :loading="deleting" @click="handleDeleteAccount">注销账号</el-button>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getProfile, updateProfile, changePassword, resetToken, checkUsername, uploadAvatar, grantInviteQuota, revokeInviteQuota, getInviteLogs, searchAllUsers, deleteAccount } from '../api/auth'
import { useAuthStore } from '../stores/auth'
import { ElMessage, ElMessageBox } from 'element-plus'

const auth = useAuthStore()
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
    ElMessage.warning('用户名已被占用')
    return
  }
  saving.value = true
  try {
    await updateProfile(form)
    ElMessage.success('保存成功')
    await loadProfile()
    // Update auth store with new profile data
    auth.setAuth(auth.token, { ...auth.user, username: form.username, nickname: form.nickname, avatar: form.avatar })
  } finally {
    saving.value = false
  }
}

const handleChangePwd = async () => {
  if (!pwdForm.old_password || !pwdForm.new_password) {
    ElMessage.warning('请填写完整')
    return
  }
  if (pwdForm.new_password.length < 6) {
    ElMessage.warning('新密码至少6位')
    return
  }
  changingPwd.value = true
  try {
    await changePassword(pwdForm)
    ElMessage.success('密码修改成功')
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
    ElMessage.success('Token 已重置')
  } catch {}
}

const copyCode = () => {
  navigator.clipboard.writeText(profile.value.invite_code)
  ElMessage.success('已复制')
}

const copyToken = () => {
  navigator.clipboard.writeText(profile.value.ddns_token)
  ElMessage.success('已复制')
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
    ElMessage.warning('请输入目标用户 ID')
    return
  }
  const available = profile.value.invite_limit - profile.value.invite_count
  if (grantForm.amount > available) {
    ElMessage.warning('超出可用额度')
    return
  }
  granting.value = true
  try {
    await grantInviteQuota(grantForm)
    ElMessage.success('邀请额度分配成功')
    grantForm.target_user_id = null
    grantForm.amount = 1
    await loadProfile()
    await loadInviteLogs()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '分配失败')
  } finally {
    granting.value = false
  }
}

const handleRevokeInvite = async () => {
  if (!revokeForm.target_user_id) {
    ElMessage.warning('请选择目标用户')
    return
  }
  revoking.value = true
  try {
    await revokeInviteQuota(revokeForm)
    ElMessage.success('邀请额度已收回')
    revokeForm.target_user_id = null
    revokeForm.amount = 1
    await loadProfile()
    await loadInviteLogs()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '收回失败')
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
  if (!isImage) ElMessage.error('头像只能是 JPG 或 PNG 格式')
  if (!isLt2M) ElMessage.error('头像大小不能超过 2MB')
  return isImage && isLt2M
}

const handleAvatarUpload = async ({ file }) => {
  const fd = new FormData()
  fd.append('file', file)
  try {
    const res = await uploadAvatar(fd)
    form.avatar = res.data.avatar
    auth.setAuth(auth.token, { ...auth.user, avatar: res.data.avatar })
    ElMessage.success('头像上传成功')
  } catch (e) {
    ElMessage.error('上传失败: ' + (e.response?.data?.message || e.message))
  }
}

const handleDeleteAccount = async () => {
  try {
    await ElMessageBox.confirm('确定要注销账号吗？您的所有域名和权限将转移给邀请您的人，此操作不可撤销。', '注销确认', {
      confirmButtonText: '确定注销',
      cancelButtonText: '取消',
      type: 'warning',
    })
  } catch { return }
  deleting.value = true
  try {
    await deleteAccount()
    ElMessage.success('账号已注销')
    auth.clearAuth()
    router.push('/login')
  } catch (e) {
    ElMessage.error(e.response?.data?.message || '注销失败')
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
