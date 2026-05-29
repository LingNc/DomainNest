import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import { useAvatarCache } from '../stores/avatarCache'
import i18n from '../i18n'

function snakeToCamel(str) {
  return str.toLowerCase().replace(/_([a-z])/g, (_, letter) => letter.toUpperCase())
}

function showErrorToast(data) {
  const errorCode = data?.error_code
  const params = data?.params || []

  if (errorCode) {
    const i18nKey = `errors.${snakeToCamel(errorCode)}`
    const message = i18n.global.t(i18nKey, params)
    // If i18n returns the key itself (missing translation), fall back
    if (message !== i18nKey) {
      ElMessage.error(message)
      return
    }
  }

  // Fallback: backend message or generic error
  ElMessage.error(data?.message || i18n.global.t('errors.requestFailed'))
}

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
        showErrorToast(data)
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
    showErrorToast(error.response?.data)
    return Promise.reject(error)
  }
)

export default request
