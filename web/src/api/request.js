import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import { useAvatarCache } from '../stores/avatarCache'
import i18n from '../i18n'
import { errorMessageMap } from '../i18n/errorMessages'

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

request.interceptors.request.use(config => {
  const auth = useAuthStore()
  if (auth.token) {
    config.headers.Authorization = `Bearer ${auth.token}`
  }
  return config
})

request.interceptors.response.use(
  response => {
    // Blob responses (file downloads) bypass the code check
    if (response.config.responseType === 'blob') {
      return response
    }
    const { data } = response
    if (data.code !== 0) {
      if (!response.config?.skipErrorToast) {
        const translatedKey = errorMessageMap.get(data.message)
        ElMessage.error(translatedKey ? i18n.global.t(translatedKey) : (data.message || i18n.global.t('errors.requestFailed')))
      }
      return Promise.reject(data)
    }
    useAvatarCache().extractFromResponse(data.data)
    return data
  },
  error => {
    if (error.config?.skipErrorToast) {
      return Promise.reject(error)
    }
    if (error.response?.status === 401) {
      const auth = useAuthStore()
      auth.clearAuth()
      // Don't redirect if already on a public page (login/register/forgot)
      const publicPaths = ['/login', '/register', '/forgot-password', '/reset-password']
      if (!publicPaths.some(p => window.location.pathname.startsWith(p))) {
        window.location.href = '/login'
      }
    }
    const errMsg = error.response?.data?.message
    const translatedKey = errMsg ? errorMessageMap.get(errMsg) : null
    ElMessage.error(translatedKey ? i18n.global.t(translatedKey) : (errMsg || i18n.global.t('errors.requestFailed')))
    return Promise.reject(error)
  }
)

export default request
