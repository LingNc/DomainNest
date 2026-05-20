import { ref } from 'vue'
import { useAuthStore } from '../stores/auth'

let ws = null
let reconnectTimer = null
let reconnectDelay = 1000
const MAX_RECONNECT_DELAY = 30000
const listeners = new Map()

export const connected = ref(false)

function connect() {
  const auth = useAuthStore()
  if (!auth.token) return
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return

  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const url = `${protocol}://${window.location.host}/api/v1/ws?token=${auth.token}`

  try {
    ws = new WebSocket(url)
  } catch {
    scheduleReconnect()
    return
  }

  ws.onopen = () => {
    connected.value = true
    reconnectDelay = 1000
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      const cbs = listeners.get(msg.type)
      if (cbs) cbs.forEach(fn => fn(msg.payload))
      const allCbs = listeners.get('*')
      if (allCbs) allCbs.forEach(fn => fn(msg.payload, msg.type))
    } catch { /* ignore parse errors */ }
  }

  ws.onclose = (e) => {
    connected.value = false
    ws = null
    if (e.code !== 1000) scheduleReconnect()
  }

  ws.onerror = () => { /* onclose fires after onerror */ }
}

function scheduleReconnect() {
  if (reconnectTimer) return
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null
    reconnectDelay = Math.min(reconnectDelay * 2, MAX_RECONNECT_DELAY)
    connect()
  }, reconnectDelay)
}

export function disconnect() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  if (ws) {
    ws.close(1000, 'logout')
    ws = null
  }
  connected.value = false
  listeners.clear()
}

export function on(type, callback) {
  if (!listeners.has(type)) listeners.set(type, new Set())
  listeners.get(type).add(callback)
}

export function off(type, callback) {
  const cbs = listeners.get(type)
  if (cbs) cbs.delete(callback)
}

export function useWebSocket() {
  const auth = useAuthStore()
  if (auth.token && !ws) connect()
  return { connected, on, off, disconnect }
}
