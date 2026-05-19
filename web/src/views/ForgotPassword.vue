<template>
  <div class="forgot-container">
    <el-card class="forgot-card">
      <div class="forgot-header">
        <h2>DomainNest</h2>
        <p>找回密码</p>
      </div>
      <el-form v-if="!sent" :model="form" @submit.prevent="handleSubmit">
        <el-form-item>
          <el-input v-model="form.email" placeholder="请输入注册邮箱" prefix-icon="Message" size="large" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" native-type="submit" size="large" style="width:100%">发送验证码</el-button>
        </el-form-item>
        <div class="links">
          <router-link to="/login">返回登录</router-link>
        </div>
      </el-form>
      <div v-else class="success-msg">
        <el-icon :size="48" color="#67c23a"><component :is="'CircleCheck'" /></el-icon>
        <p>如果该邮箱已注册，我们已向 <b>{{ form.email }}</b> 发送了验证码。</p>
        <p class="hint">请检查收件箱（及垃圾邮件），验证码 30 分钟内有效。</p>
        <el-button type="primary" @click="$router.push('/reset-password')" style="margin-top:16px">去重置密码</el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { forgotPassword } from '../api/auth'
import { ElMessage } from 'element-plus'

const loading = ref(false)
const sent = ref(false)
const form = ref({ email: '' })

const handleSubmit = async () => {
  if (!form.value.email) {
    ElMessage.warning('请输入邮箱地址')
    return
  }
  loading.value = true
  try {
    await forgotPassword({ email: form.value.email })
    sent.value = true
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.forgot-container {
  display: flex;
  justify-content: center;
  align-items: flex-start;
  padding: 40px 16px;
  background: #f5f7fa;
  min-height: 100%;
}
.forgot-card {
  width: 100%;
  max-width: 420px;
  border-radius: 12px;
}
.forgot-header {
  text-align: center;
  margin-bottom: 24px;
}
.forgot-header h2 {
  font-size: 26px;
  color: #1d1e2c;
  margin-bottom: 8px;
}
.forgot-header p {
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
  line-height: 1.6;
}
.hint {
  color: #909399 !important;
  font-size: 13px !important;
}
</style>
