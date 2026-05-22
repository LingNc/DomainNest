import request from './request'

// User notification endpoints
export const getNotifications = (params) => request.get('/notifications', { params })
export const markAsRead = (id) => request.put(`/notifications/${id}/read`)
export const markAllAsRead = () => request.put('/notifications/read-all')
export const getUnreadCount = () => request.get('/notifications/unread-count', { skipErrorToast: true })
export const deleteNotification = (id) => request.delete(`/notifications/${id}`)

// Admin notification endpoints
export const broadcastNotification = (data) => request.post('/admin/notifications/broadcast', data)
export const getNotificationStats = () => request.get('/admin/notifications/stats')
export const adminListNotifications = (params) => request.get('/admin/notifications', { params })
export const adminDeleteNotification = (id) => request.delete(`/admin/notifications/${id}`)
export const adminPurgeExpired = () => request.post('/admin/notifications/purge-expired')
