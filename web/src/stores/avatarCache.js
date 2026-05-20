import { reactive } from 'vue'

const cache = reactive({})

export function useAvatarCache() {
  function set(id, avatar) {
    if (id && avatar) cache[id] = avatar
  }

  function get(id) {
    return cache[id] || ''
  }

  function remove(id) {
    delete cache[id]
  }

  function extractFromUser(user) {
    if (user && user.id && user.avatar) {
      cache[user.id] = user.avatar
    }
  }

  function extractFromResponse(data) {
    if (!data) return
    if (Array.isArray(data)) {
      data.forEach(item => extractFromResponse(item))
      return
    }
    if (typeof data === 'object') {
      // Cache if this object has id + avatar
      if (data.id && data.avatar) {
        cache[data.id] = data.avatar
      }
      // Recurse into known user-bearing fields
      const userFields = ['user', 'target_user', 'owner', 'sender', 'receiver', 'creator', 'friend', 'inviter', 'invitee', 'partner']
      for (const field of userFields) {
        if (data[field]) extractFromResponse(data[field])
      }
      // Recurse into items/data arrays
      if (data.items) extractFromResponse(data.items)
      if (data.data) extractFromResponse(data.data)
    }
  }

  function clear() {
    Object.keys(cache).forEach(k => delete cache[k])
  }

  return { cache, set, get, remove, extractFromUser, extractFromResponse, clear }
}
