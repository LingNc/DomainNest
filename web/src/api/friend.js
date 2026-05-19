import request from './request'

export const listFriends = () => request.get('/friends')
export const sendFriendRequest = (data) => request.post('/friends', data)
export const removeFriend = (id) => request.delete(`/friends/${id}`)
export const listPendingRequests = () => request.get('/friends/requests/pending')
export const listSentRequests = () => request.get('/friends/requests/sent')
export const acceptRequest = (id) => request.post(`/friends/requests/${id}/accept`)
export const rejectRequest = (id) => request.post(`/friends/requests/${id}/reject`)
export const searchUsers = (q) => request.get('/friends/search', { params: { q } })
