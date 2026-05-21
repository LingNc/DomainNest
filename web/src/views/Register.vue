<!-- web/src/views/Register.vue -->
<template>
  <div class="register-container">
    <el-card class="register-card">
      <div class="register-header">
        <h2>DomainNest</h2>
        <p>{{ $t('register.title') }}</p>
      </div>
      <el-form :model="form" @submit.prevent="handleRegister">
        <el-form-item>
          <el-input v-model="form.username" :placeholder="$t('common.username')" prefix-icon="User" size="large" @input="onUsernameInput" :class="{ 'is-error': usernameError }" />
          <div v-if="usernameStatus" :class="usernameStatus === 'available' ? 'username-ok' : 'username-err'">
            {{ usernameStatus === 'available' ? $t('register.usernameAvailable') : $t('register.usernameTaken') }}
          </div>
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.email" :placeholder="$t('register.emailOptional')" prefix-icon="Message" size="large" />
        </el-form-item>
        <el-form-item v-if="form.email">
          <div class="verify-row">
            <el-input v-model="form.code" :placeholder="$t('register.codePlaceholder')" prefix-icon="Key" size="large" :disabled="emailVerified" />
            <el-button v-if="!emailVerified" type="primary" size="large" :loading="sendingCode" :disabled="countdown > 0" @click="handleSendCode">
              {{ countdown > 0 ? $t('register.resendCountdown', { seconds: countdown }) : (codeSent ? $t('register.resendCode') : $t('register.sendCode')) }}
            </el-button>
            <el-button v-else type="success" size="large" disabled>
              {{ $t('register.emailVerified') }}
            </el-button>
          </div>
        </el-form-item>
        <el-form-item v-if="form.email && codeSent && !emailVerified">
          <el-button type="success" :loading="verifying" style="width:100%" size="large" @click="handleVerifyEmail">
            {{ $t('register.verify') }}
          </el-button>
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" :placeholder="$t('common.password')" prefix-icon="Lock" show-password size="large" />
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.invite_code" :placeholder="$t('register.inviteCode')" prefix-icon="Link" size="large" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" native-type="submit" size="large" style="width:100%">{{ $t('register.registerBtn') }}</el-button>
        </el-form-item>
        <div class="links">
          <router-link to="/login">{{ $t('register.goLogin') }}</router-link>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { register, checkUsername, sendVerifyEmail, verifyEmail } from '../api/auth'
import { ElMessage } from 'element-plus'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const loading = ref(false)
const form = ref({ username: '', email: '', password: '', invite_code: '', code: '' })
const usernameStatus = ref('') // '' | 'available' | 'taken'
let checkTimer = null
const sendingCode = ref(false)
const verifying = ref(false)
const codeSent = ref(false)
const emailVerified = ref(false)
const countdown = ref(0)
let countdownTimer = null

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

const startCountdown = () => {
  countdown.value = 60
  countdownTimer = setInterval(() => {
    countdown.value--
    if (countdown.value <= 0) {
      clearInterval(countdownTimer)
    }
  }, 1000)
}

const handleSendCode = async () => {
  if (!form.value.email) {
    ElMessage.warning(t('register.emailRequired'))
    return
  }
  sendingCode.value = true
  try {
    await sendVerifyEmail({ email: form.value.email, purpose: 'register' }, { skipErrorToast: true })
    codeSent.value = true
    ElMessage.success(t('register.codeSent'))
    startCountdown()
  } catch (e) {
    ElMessage.error(e.response?.data?.message || e.message)
  } finally {
    sendingCode.value = false
  }
}

const handleVerifyEmail = async () => {
  if (!form.value.code) {
    ElMessage.warning(t('register.codeRequired'))
    return
  }
  verifying.value = true
  try {
    await verifyEmail({ email: form.value.email, code: form.value.code, purpose: 'register' }, { skipErrorToast: true })
    emailVerified.value = true
    ElMessage.success(t('register.verified'))
  } catch (e) {
    ElMessage.error(e.response?.data?.message || e.message)
  } finally {
    verifying.value = false
  }
}

const handleRegister = async () => {
  if (!form.value.invite_code) {
    ElMessage.warning(t('register.enterInviteCode'))
    return
  }
  if (usernameStatus.value === 'taken') {
    ElMessage.warning(t('register.usernameTaken'))
    return
  }
  loading.value = true
  try {
    await register(form.value)
    ElMessage.success(t('register.registerSuccess'))
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
.verify-row {
  display: flex;
  gap: 8px;
  width: 100%;
}
.verify-row .el-input {
  flex: 1;
}
.verify-row .el-button {
  flex-shrink: 0;
  min-width: 120px;
}
</style>
