<!-- web/src/views/Register.vue -->
<template>
  <div class="register-container">
    <el-card class="register-card">
      <div class="register-header">
        <h2>DomainNest</h2>
        <p>注册新账号</p>
      </div>
      <el-form :model="form" @submit.prevent="handleRegister">
        <el-form-item>
          <el-input v-model="form.username" placeholder="用户名" prefix-icon="User" size="large" @input="onUsernameInput" :class="{ 'is-error': usernameError }" />
          <div v-if="usernameStatus" :class="usernameStatus === 'available' ? 'username-ok' : 'username-err'">
            {{ usernameStatus === 'available' ? '用户名可用' : '用户名已被占用' }}
          </div>
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.email" placeholder="邮箱（选填）" prefix-icon="Message" size="large" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" placeholder="密码" prefix-icon="Lock" show-password size="large" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.invite_code" placeholder="邀请码" prefix-icon="Link" size="large" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" native-type="submit" size="large" style="width:100%">注 册</el-button>
        </el-form-item>
        <div class="links">
          <router-link to="/login">已有账号？去登录</router-link>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { register, checkUsername } from '../api/auth'
import { ElMessage } from 'element-plus'

const router = useRouter()
const route = useRoute()
const loading = ref(false)
const form = ref({ username: '', email: '', password: '', invite_code: '' })
const usernameStatus = ref('') // '' | 'available' | 'taken'
let checkTimer = null

onMounted(() => {
  if (route.query.code) {
    form.value.invite_code = route.query.code
  }
})

const onUsernameInput = () => {
  usernameStatus.value = ''
  clearTimeout(checkTimer)
  const name = form.value.username.trim()
  if (name.length < 3) return
  checkTimer = setTimeout(async () => {
    try {
      const res = await checkUsername(name)
      usernameStatus.value = res.data.available ? 'available' : 'taken'
    } catch { /* ignore */ }
  }, 400)
}

const handleRegister = async () => {
  if (!form.value.invite_code) {
    ElMessage.warning('请输入邀请码')
    return
  }
  if (usernameStatus.value === 'taken') {
    ElMessage.warning('用户名已被占用')
    return
  }
  loading.value = true
  try {
    await register(form.value)
    ElMessage.success('注册成功，请登录')
    router.push('/login')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.register-container {
  display: flex;
  justify-content: center;
  align-items: flex-start;
  padding: 40px 16px;
  background: #f5f7fa;
  min-height: 100%;
}
.register-card {
  width: 100%;
  max-width: 420px;
  border-radius: 12px;
}
.register-header {
  text-align: center;
  margin-bottom: 24px;
}
.register-header h2 {
  font-size: 26px;
  color: #1d1e2c;
  margin-bottom: 8px;
}
.register-header p {
  color: #909399;
  font-size: 14px;
}
.links {
  text-align: center;
}
.links a {
  color: #409eff;
  text-decoration: none;
  font-size: 14px;
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
</style>
