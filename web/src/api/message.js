import request from './request'

export const sendMessage = (data) => request.post('/messages', data)
export const getConversations = () => request.get('/messages/conversations')
export const getMessages = (id, params) => request.get(`/messages/${id}`, { params })
export const markAsRead = (id) => request.post(`/messages/${id}/read`)
export const getUnreadCount = () => request.get('/messages/unread-count')
