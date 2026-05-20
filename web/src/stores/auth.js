import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useAvatarCache } from './avatarCache'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref(JSON.parse(localStorage.getItem('user') || 'null'))

  const isAuthed = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const isSuperAdmin = computed(() => !!user.value?.is_super_admin)
  const username = computed(() => user.value?.username || '')
  const nickname = computed(() => user.value?.nickname || user.value?.username || '')
  const avatar = computed(() => user.value?.avatar || '')

  function setAuth(t, u) {
    token.value = t
    user.value = u
    localStorage.setItem('token', t)
    localStorage.setItem('user', JSON.stringify(u))
    if (u?.id && u?.avatar) {
      useAvatarCache().set(u.id, u.avatar)
    }
  }

  function clearAuth() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  return { token, user, isAuthed, isAdmin, isSuperAdmin, username, nickname, avatar, setAuth, clearAuth }
})
