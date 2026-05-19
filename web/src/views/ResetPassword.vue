<template>
  <div class="reset-container">
    <el-card class="reset-card">
      <div class="reset-header">
        <h2>DomainNest</h2>
        <p>{{ $t('resetPassword.title') }}</p>
      </div>
      <el-form v-if="!done" :model="form" @submit.prevent="handleSubmit">
        <el-form-item>
          <el-input v-model="form.token" :placeholder="$t('resetPassword.codePlaceholder')" prefix-icon="Key" size="large" maxlength="6" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.new_password" type="password" :placeholder="$t('resetPassword.newPasswordPlaceholder')" prefix-icon="Lock" show-password size="large" autocomplete="new-password" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="confirmPwd" type="password" :placeholder="$t('resetPassword.confirmPassword')" prefix-icon="Lock" show-password size="large" autocomplete="new-password" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" native-type="submit" size="large" style="width:100%">{{ $t('resetPassword.resetBtn') }}</el-button>
        </el-form-item>
        <div class="links">
          <router-link to="/login">{{ $t('common.backToLogin') }}</router-link>
        </div>
      </el-form>
      <div v-else class="success-msg">
        <el-icon :size="48" color="#67c23a"><component :is="'CircleCheck'" /></el-icon>
        <p>{{ $t('resetPassword.successMsg') }}</p>
        <el-button type="primary" @click="$router.push('/login')" style="margin-top:16px">{{ $t('resetPassword.goLogin') }}</el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { resetPassword } from '../api/auth'
import { ElMessage } from 'element-plus'

const router = useRouter()
const { t } = useI18n()
const loading = ref(false)
const done = ref(false)
const confirmPwd = ref('')
const form = ref({ token: '', new_password: '' })

const handleSubmit = async () => {
  if (!form.value.token || form.value.token.length !== 6) {
    ElMessage.warning(t('resetPassword.enterCode'))
    return
  }
  if (!form.value.new_password || form.value.new_password.length < 6) {
    ElMessage.warning(t('resetPassword.passwordMinLength'))
    return
  }
  if (form.value.new_password !== confirmPwd.value) {
    ElMessage.warning(t('resetPassword.passwordMismatch'))
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
