import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'

const STYLE_ID = 'error-box-style'
const CUSTOM_CLASS = 'error-box-alert'

function ensureStyle() {
  if (document.getElementById(STYLE_ID)) return
  const style = document.createElement('style')
  style.id = STYLE_ID
  style.textContent = `
    .${CUSTOM_CLASS} { background: #fef0f0 !important; }
    .${CUSTOM_CLASS} .el-message-box__header { background: #fef0f0 !important; }
    .${CUSTOM_CLASS} .el-message-box__title { color: #f56c6c !important; }
    .${CUSTOM_CLASS} .el-message-box__content { color: #c45656; }
  `
  document.head.appendChild(style)
}

export function useError() {
  const { t } = useI18n()

  const showError = (msg) => {
    if (msg.length > 100) {
      ensureStyle()
      ElMessageBox.alert(msg, t('common.error'), {
        type: 'error',
        confirmButtonText: t('common.confirm'),
        customClass: CUSTOM_CLASS,
      })
    } else {
      ElMessage.error(msg)
    }
  }

  return { showError }
}
