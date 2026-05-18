<!-- web/src/views/Settings.vue -->
<template>
  <div>
    <div class="page-header">
      <h2>系统设置</h2>
      <p class="subtitle">管理您的 DDNS Token 和账号设置</p>
    </div>

    <el-row :gutter="20">
      <el-col :span="14">
        <el-card>
          <template #header>DDNS Token</template>
          <p class="token-label">您的 DDNS Token：</p>
          <div class="token-row">
            <el-tag size="large" class="token-value">{{ token || '加载中...' }}</el-tag>
            <el-button type="warning" size="small" @click="handleResetToken">重置 Token</el-button>
          </div>
          <el-divider />
          <h4>ddns-go 配置说明</h4>
          <p class="hint">在 ddns-go 的 Webhook 设置中填入以下配置：</p>
          <el-input type="textarea" :rows="4" :value="ddnsConfig" readonly class="config-textarea" />
          <el-button type="primary" size="small" @click="copyConfig" style="margin-top:12px">
            复制配置
          </el-button>
        </el-card>
      </el-col>

      <el-col :span="10">
        <el-card>
          <template #header>修改密码</template>
          <el-form :model="passwordForm" @submit.prevent="handleChangePassword" label-width="80px">
            <el-form-item label="旧密码">
              <el-input v-model="passwordForm.old_password" type="password" show-password />
            </el-form-item>
            <el-form-item label="新密码">
              <el-input v-model="passwordForm.new_password" type="password" show-password />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" native-type="submit">修改密码</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getProfile, resetToken, changePassword } from '../api/auth'
import { ElMessage } from 'element-plus'

const token = ref('')
const passwordForm = ref({ old_password: '', new_password: '' })

const ddnsConfig = computed(() => {
  const baseUrl = window.location.origin
  return `URL:\n${baseUrl}/api/v1/ddns/update?token=${token.value}\n\nRequestBody:\n{"domain":"#{domain}","ip":"#{ip}","record_type":"#{recordType}","ttl":#{ttl}}`
})

const loadProfile = async () => {
  const res = await getProfile()
  token.value = res.data.ddns_token
}

const handleResetToken = async () => {
  const res = await resetToken()
  token.value = res.data.token
  ElMessage.success('Token 已重置')
}

const handleChangePassword = async () => {
  await changePassword(passwordForm.value)
  ElMessage.success('密码修改成功')
  passwordForm.value = { old_password: '', new_password: '' }
}

const copyConfig = () => {
  navigator.clipboard.writeText(ddnsConfig.value)
  ElMessage.success('已复制到剪贴板')
}

onMounted(loadProfile)
</script>

<style scoped>
.page-header {
  margin-bottom: 20px;
}
.page-header h2 {
  font-size: 22px;
  font-weight: 600;
  margin-bottom: 4px;
}
.subtitle {
  color: #909399;
  font-size: 14px;
}
.token-label {
  color: #606266;
  margin-bottom: 8px;
}
.token-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.token-value {
  font-family: monospace;
  font-size: 14px;
}
.hint {
  color: #909399;
  font-size: 13px;
  margin-bottom: 12px;
}
.config-textarea :deep(textarea) {
  font-family: monospace;
  font-size: 13px;
}
</style>
