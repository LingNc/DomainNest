<template>
  <div class="profile-page">
    <el-card>
      <template #header>
        <span>个人信息</span>
      </template>
      <el-form :model="form" label-width="80px" style="max-width:480px">
        <el-form-item label="头像">
          <div class="avatar-section">
            <el-avatar v-if="form.avatar" :src="form.avatar" :size="64" />
            <el-avatar v-else :size="64">{{ (form.username || '?')[0]?.toUpperCase() }}</el-avatar>
            <el-input v-model="form.avatar" placeholder="输入头像图片 URL" style="margin-left:12px;flex:1" />
          </div>
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="form.username" placeholder="修改用户名" @input="onUsernameInput" />
          <div v-if="usernameStatus" :class="usernameStatus === 'available' ? 'username-ok' : 'username-err'">
            {{ usernameStatus === 'available' ? '用户名可用' : '用户名已被占用' }}
          </div>
        </el-form-item>
        <el-form-item label="角色">
          <el-tag :type="profile.role === 'admin' ? 'danger' : 'info'">{{ profile.role === 'admin' ? '管理员' : '普通用户' }}</el-tag>
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
        <el-descriptions-item label="已邀请">{{ profile.invite_count }}</el-descriptions-item>
      </el-descriptions>
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
          <router-link to="/forgot-password" style="margin-left:16px;font-size:13px">忘记密码？通过邮箱重置</router-link>
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
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { getProfile, updateProfile, changePassword, resetToken, checkUsername } from '../api/auth'
import { useAuthStore } from '../stores/auth'
import { ElMessage } from 'element-plus'

const auth = useAuthStore()
const profile = ref({})
const saving = ref(false)
const changingPwd = ref(false)
const usernameStatus = ref('')
let checkTimer = null
const form = reactive({ username: '', nickname: '', email: '', phone: '', avatar: '' })
const pwdForm = reactive({ old_password: '', new_password: '' })

const loadProfile = async () => {
  const res = await getProfile()
  profile.value = res.data
  form.username = res.data.username || ''
  form.nickname = res.data.nickname || ''
  form.email = res.data.email || ''
  form.phone = res.data.phone || ''
  form.avatar = res.data.avatar || ''
}

onMounted(loadProfile)

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
</script>

<style scoped>
.profile-page {
  max-width: 640px;
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
}
</style>
