package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/middleware"
	"domainnest/internal/service"

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

	middleware.LogOperation(h.db, userID, "send_friend_request", "user", &req.ReceiverID, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "friend request sent"})
}

// AcceptRequest accepts a pending friend request.
func (h *FriendHandler) AcceptRequest(c *gin.Context) {
	userID := c.GetUint64("user_id")
	requestID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid request id"})
		return
	}

	if err := h.friendService.AcceptRequest(requestID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "friend request accepted"})
}

// RejectRequest rejects a pending friend request.
func (h *FriendHandler) RejectRequest(c *gin.Context) {
	userID := c.GetUint64("user_id")
	requestID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid request id"})
		return
	}

	if err := h.friendService.RejectRequest(requestID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "friend request rejected"})
}

// RemoveFriend removes a friend.
func (h *FriendHandler) RemoveFriend(c *gin.Context) {
	userID := c.GetUint64("user_id")
	friendID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid friend id"})
		return
	}

	if err := h.friendService.RemoveFriend(userID, friendID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "friend removed"})
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
