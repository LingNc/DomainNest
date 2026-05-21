import request from './request'

export const sendMessage = (data) => request.post('/messages', data)
export const getConversations = () => request.get('/messages/conversations')
export const getMessages = (id, params) => request.get(`/messages/${id}`, { params })
export const markAsRead = (id) => request.post(`/messages/${id}/read`)
export const getUnreadCount = () => request.get('/messages/unread-count', { skipErrorToast: true })
export const getNotifications = (params) => request.get('/messages/notifications', { params })
export const getNotificationUnreadCount = () => request.get('/messages/notifications/unread-count', { skipErrorToast: true })
export const markNotificationAsRead = (id) => request.post(`/messages/notifications/${id}/read`)
export const markAllNotificationsAsRead = () => request.post('/messages/notifications/read-all')
export const handleNotificationAction = (id, action, config) => request.post(`/messages/notifications/${id}/action`, { action }, config)
