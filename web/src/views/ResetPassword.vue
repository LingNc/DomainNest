<template>
  <div class="reset-container">
    <el-card class="reset-card">
      <div class="reset-header">
        <h2>DomainNest</h2>
        <p>重置密码</p>
      </div>
      <el-form v-if="!done" :model="form" @submit.prevent="handleSubmit">
        <el-form-item>
          <el-input v-model="form.token" placeholder="请输入 6 位验证码" prefix-icon="Key" size="large" maxlength="6" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.new_password" type="password" placeholder="新密码（至少6位）" prefix-icon="Lock" show-password size="large" autocomplete="new-password" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="confirmPwd" type="password" placeholder="确认新密码" prefix-icon="Lock" show-password size="large" autocomplete="new-password" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" native-type="submit" size="large" style="width:100%">重置密码</el-button>
        </el-form-item>
        <div class="links">
          <router-link to="/login">返回登录</router-link>
        </div>
      </el-form>
      <div v-else class="success-msg">
        <el-icon :size="48" color="#67c23a"><component :is="'CircleCheck'" /></el-icon>
        <p>密码重置成功！</p>
        <el-button type="primary" @click="$router.push('/login')" style="margin-top:16px">去登录</el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { resetPassword } from '../api/auth'
import { ElMessage } from 'element-plus'

const router = useRouter()
const loading = ref(false)
const done = ref(false)
const confirmPwd = ref('')
const form = ref({ token: '', new_password: '' })

const handleSubmit = async () => {
  if (!form.value.token || form.value.token.length !== 6) {
    ElMessage.warning('请输入 6 位验证码')
    return
  }
  if (!form.value.new_password || form.value.new_password.length < 6) {
    ElMessage.warning('密码至少6位')
    return
  }
  if (form.value.new_password !== confirmPwd.value) {
    ElMessage.warning('两次密码不一致')
    return
  }
  loading.value = true
  try {
    await resetPassword({ token: form.value.token, new_password: form.value.new_password })
    done.value = true
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.reset-container {
  display: flex;
  justify-content: center;
  align-items: flex-start;
  padding: 40px 16px;
  background: #f5f7fa;
  min-height: 100%;
}
.reset-card {
  width: 100%;
  max-width: 420px;
  border-radius: 12px;
}
.reset-header {
  text-align: center;
  margin-bottom: 24px;
}
.reset-header h2 {
  font-size: 26px;
  color: #1d1e2c;
  margin-bottom: 8px;
}
.reset-header p {
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
.success-msg {
  text-align: center;
  padding: 16px 0;
}
.success-msg p {
  margin-top: 12px;
  color: #303133;
  font-size: 14px;
}
</style>
