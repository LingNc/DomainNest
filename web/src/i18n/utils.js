const LANG_KEY = 'lang'

export function getStoredLang() {
  const stored = localStorage.getItem(LANG_KEY)
  if (stored === 'zh-CN' || stored === 'en-US') return stored
  const navLang = (navigator.language || '').toLowerCase()
  return navLang.startsWith('zh') ? 'zh-CN' : 'en-US'
}

export function setLang(lang) {
  localStorage.setItem(LANG_KEY, lang)
}
