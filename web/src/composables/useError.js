import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'

export function useError() {
  const { t } = useI18n()

  const showError = (msg) => {
    if (msg.length > 100) {
      ElMessageBox.alert(msg, t('common.error'), { type: 'error', confirmButtonText: t('common.confirm') })
    } else {
      ElMessage.error(msg)
    }
  }

  return { showError }
}
