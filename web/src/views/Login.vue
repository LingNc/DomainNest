<!-- web/src/views/Login.vue -->
<template>
  <div class="login-container">
    <el-card class="login-card">
      <div class="login-header">
        <h2>DomainNest</h2>
        <p>{{ $t('login.subtitle') }}</p>
      </div>
      <el-form :model="form" @submit.prevent="handleLogin">
        <el-form-item>
          <el-input v-model="form.username" :placeholder="$t('common.username')" prefix-icon="User" size="large" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" :placeholder="$t('common.password')" prefix-icon="Lock" show-password size="large" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" native-type="submit" size="large" style="width:100%">{{ $t('login.loginBtn') }}</el-button>
        </el-form-item>
        <div class="links">
          <router-link to="/register">{{ $t('login.register') }}</router-link>
          <router-link to="/forgot-password">{{ $t('login.forgotPassword') }}</router-link>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { login } from '../api/auth'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const form = ref({ username: '', password: '' })

onMounted(() => {
  if (auth.isLoggedIn) {
    router.push('/dashboard')
  }
})

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
  background: #f5f7fa;
  padding: 16px;
}
.login-card {
  width: 100%;
  max-width: 420px;
  border-radius: 12px;
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
  .login-header h2 {
    font-size: 22px;
  }
}
</style>
