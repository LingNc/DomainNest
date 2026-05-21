import { ElMessage, ElNotification } from 'element-plus'
import { useI18n } from 'vue-i18n'

export function useError() {
  const { t } = useI18n()

  const showError = (msg) => {
    if (msg.length > 100) {
      ElNotification({ type: 'error', title: t('common.error'), message: msg, duration: 0 })
    } else {
      ElMessage.error(msg)
    }
  }

  return { showError }
}
