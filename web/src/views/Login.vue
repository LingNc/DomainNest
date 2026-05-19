<!-- web/src/views/Login.vue -->
<template>
  <div class="login-container">
    <el-card class="login-card">
      <div class="login-header">
        <h2>DomainNest</h2>
        <p>域名分配与 DDNS 管理系统</p>
      </div>
      <el-form :model="form" @submit.prevent="handleLogin">
        <el-form-item>
          <el-input v-model="form.username" placeholder="用户名" prefix-icon="User" size="large" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" placeholder="密码" prefix-icon="Lock" show-password size="large" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" native-type="submit" size="large" style="width:100%">登 录</el-button>
        </el-form-item>
        <div class="links">
          <router-link to="/register">注册新账号</router-link>
          <router-link to="/forgot-password">忘记密码</router-link>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { login } from '../api/auth'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const form = ref({ username: '', password: '' })

const handleLogin = async () => {
  loading.value = true
  try {
    const res = await login(form.value)
    auth.setAuth(res.data.token, res.data.user)
    router.push('/dashboard')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #1d1e2c 0%, #2d3a4a 100%);
  padding: 16px;
}
.login-card {
  width: 100%;
  max-width: 420px;
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0,0,0,0.2);
}
.login-header {
  text-align: center;
  margin-bottom: 24px;
}
.login-header h2 {
  font-size: 26px;
  color: #1d1e2c;
  margin-bottom: 8px;
}
.login-header p {
  color: #909399;
  font-size: 14px;
}
.links {
  display: flex;
  justify-content: space-between;
}
.links a {
  color: #409eff;
  text-decoration: none;
  font-size: 14px;
}
@media (max-width: 768px) {
  .login-card {
    box-shadow: none;
    background: #fff;
  }
  .login-header h2 {
    font-size: 22px;
  }
}
</style>
