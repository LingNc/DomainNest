import { useAvatarCache } from '../stores/avatarCache'

export function useAvatar() {
  const avatarCache = useAvatarCache()

  function getSrc(userId, directAvatar) {
    return directAvatar || avatarCache.get(userId) || ''
  }

  function firstLetter(username) {
    return (username || '?')[0]?.toUpperCase() || '?'
  }

  return { getSrc, firstLetter }
}
