<template>
  <div class="profile-page">
    <el-card>
      <template #header>
        <span>个人信息</span>
      </template>
      <el-form :model="form" label-width="80px" style="max-width:480px">
        <el-form-item label="用户名">
          <el-input :model-value="profile.username" disabled />
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
          <el-input v-model="pwdForm.old_password" type="password" show-password />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="pwdForm.new_password" type="password" show-password />
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
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { getProfile, updateProfile, changePassword, resetToken } from '../api/auth'
import { ElMessage } from 'element-plus'

const profile = ref({})
const saving = ref(false)
const changingPwd = ref(false)
const form = reactive({ nickname: '', email: '', phone: '' })
const pwdForm = reactive({ old_password: '', new_password: '' })

const loadProfile = async () => {
  const res = await getProfile()
  profile.value = res.data
  form.nickname = res.data.nickname || ''
  form.email = res.data.email || ''
  form.phone = res.data.phone || ''
}

onMounted(loadProfile)

const handleSave = async () => {
  saving.value = true
  try {
    await updateProfile(form)
    ElMessage.success('保存成功')
    await loadProfile()
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
</style>
