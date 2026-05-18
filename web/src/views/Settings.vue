<template>
  <el-container>
    <el-header>
      <div class="header-content">
        <h2>DomainNest</h2>
        <el-button @click="$router.push('/dashboard')">Back</el-button>
      </div>
    </el-header>
    <el-main>
      <el-card>
        <template #header>DDNS Token</template>
        <p>Your DDNS Token: <el-tag>{{ token }}</el-tag></p>
        <el-button type="warning" @click="handleResetToken">Reset Token</el-button>

        <el-divider />

        <h4>ddns-go Configuration</h4>
        <p>Use these settings in your ddns-go Webhook configuration:</p>
        <el-input type="textarea" :rows="3" :value="ddnsConfig" readonly />
        <el-button type="primary" size="small" @click="copyConfig" style="margin-top:10px">Copy</el-button>
      </el-card>

      <el-card style="margin-top:20px">
        <template #header>Change Password</template>
        <el-form :model="passwordForm" @submit.prevent="handleChangePassword">
          <el-form-item label="Old Password">
            <el-input v-model="passwordForm.old_password" type="password" show-password />
          </el-form-item>
          <el-form-item label="New Password">
            <el-input v-model="passwordForm.new_password" type="password" show-password />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" native-type="submit">Change Password</el-button>
          </el-form-item>
        </el-form>
      </el-card>
    </el-main>
  </el-container>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getProfile, resetToken, changePassword } from '../api/auth'
import { ElMessage } from 'element-plus'

const token = ref('')
const passwordForm = ref({ old_password: '', new_password: '' })

const ddnsConfig = computed(() => {
  const baseUrl = window.location.origin
  return `URL: ${baseUrl}/api/v1/ddns/update?token=${token.value}\nRequestBody: {"domain":"#{domain}","ip":"#{ip}","record_type":"#{recordType}","ttl":#{ttl}}`
})

const loadProfile = async () => {
  const res = await getProfile()
  token.value = res.data.ddns_token
}

const handleResetToken = async () => {
  const res = await resetToken()
  token.value = res.data.token
  ElMessage.success('Token reset successfully')
}

const handleChangePassword = async () => {
  await changePassword(passwordForm.value)
  ElMessage.success('Password changed')
  passwordForm.value = { old_password: '', new_password: '' }
}

const copyConfig = () => {
  navigator.clipboard.writeText(ddnsConfig.value)
  ElMessage.success('Copied to clipboard')
}

onMounted(loadProfile)
</script>

<style scoped>
.el-header {
  background: #409eff;
  color: white;
  line-height: 60px;
}
.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
