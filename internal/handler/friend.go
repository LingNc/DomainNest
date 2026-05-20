package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/middleware"
	"domainnest/internal/model"
	"domainnest/internal/service"
	"domainnest/internal/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FriendHandler struct {
	friendService *service.FriendService
	db            *gorm.DB
}

func NewFriendHandler(friendService *service.FriendService, db *gorm.DB) *FriendHandler {
	return &FriendHandler{friendService: friendService, db: db}
}

// SendRequest sends a friend request to another user.
func (h *FriendHandler) SendRequest(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		ReceiverID uint64 `json:"receiver_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.friendService.SendRequest(userID, req.ReceiverID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// Push friend request to receiver via WebSocket
	var friendReq model.FriendRequest
	if err := h.db.Where("sender_id = ? AND receiver_id = ? AND status = ?", userID, req.ReceiverID, "pending").
		Order("id DESC").First(&friendReq).Error; err == nil {
		ws.BroadcastToUser(req.ReceiverID, ws.TypeFriendRequest, friendReq)
	}

	middleware.LogOperationUser(h.db, userID, req.ReceiverID, "send_friend_request", "user", &req.ReceiverID, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "好友请求已发送"})
}

// AcceptRequest accepts a pending friend request.
func (h *FriendHandler) AcceptRequest(c *gin.Context) {
	userID := c.GetUint64("user_id")
	requestID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的请求ID"})
		return
	}

	// Look up request to get senderID for broadcasting
	var friendReq model.FriendRequest
	if err := h.db.First(&friendReq, requestID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求不存在"})
		return
	}

	if err := h.friendService.AcceptRequest(requestID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// Notify sender that their request was accepted
	ws.BroadcastToUser(friendReq.SenderID, ws.TypeNewNotification, gin.H{"type": "friend_accepted", "from_user_id": userID})

	friendID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	middleware.LogOperationUser(h.db, userID, friendReq.SenderID, "accept_friend", "friend", &friendID, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "好友请求已接受"})
}

// RejectRequest rejects a pending friend request.
func (h *FriendHandler) RejectRequest(c *gin.Context) {
	userID := c.GetUint64("user_id")
	requestID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的请求ID"})
		return
	}

	// Look up request to get senderID for broadcasting
	var friendReq model.FriendRequest
	if err := h.db.First(&friendReq, requestID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求不存在"})
		return
	}

	if err := h.friendService.RejectRequest(requestID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// Notify sender that their request was rejected
	ws.BroadcastToUser(friendReq.SenderID, ws.TypeNewNotification, gin.H{"type": "friend_rejected", "from_user_id": userID})

	friendID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	middleware.LogOperationUser(h.db, userID, friendReq.SenderID, "reject_friend", "friend", &friendID, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "好友请求已拒绝"})
}

// RemoveFriend removes a friend.
func (h *FriendHandler) RemoveFriend(c *gin.Context) {
	userID := c.GetUint64("user_id")
	friendID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的好友ID"})
		return
	}

	if err := h.friendService.RemoveFriend(userID, friendID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// Look up the friendship to find the other user for logging
	var friendship model.Friendship
	var targetUserID uint64
	if h.db.Where("id = ?", friendID).First(&friendship).Error == nil {
		if friendship.UserID == userID {
			targetUserID = friendship.FriendID
		} else {
			targetUserID = friendship.UserID
		}
	}

	middleware.LogOperationUser(h.db, userID, targetUserID, "remove_friend", "friend", &friendID, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "好友已删除"})
}

// ListFriends returns all friends of the current user.
func (h *FriendHandler) ListFriends(c *gin.Context) {
	userID := c.GetUint64("user_id")

	friendships, err := h.friendService.ListFriends(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": friendships})
}

// ListPendingRequests returns pending friend requests received by the current user.
func (h *FriendHandler) ListPendingRequests(c *gin.Context) {
	userID := c.GetUint64("user_id")

	requests, err := h.friendService.ListPendingRequests(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": requests})
}

// ListSentRequests returns pending friend requests sent by the current user.
func (h *FriendHandler) ListSentRequests(c *gin.Context) {
	userID := c.GetUint64("user_id")

	requests, err := h.friendService.ListSentRequests(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": requests})
}

// SearchAllUsers searches all users (friends prioritized) by username or nickname.
func (h *FriendHandler) SearchAllUsers(c *gin.Context) {
	userID := c.GetUint64("user_id")
	keyword := c.Query("q")

	users, err := h.friendService.SearchAllUsers(userID, keyword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": users})
}

// SearchUsers searches for users by username or nickname.
func (h *FriendHandler) SearchUsers(c *gin.Context) {
	userID := c.GetUint64("user_id")
	keyword := c.Query("q")

	users, err := h.friendService.SearchUsers(userID, keyword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": users})
}
