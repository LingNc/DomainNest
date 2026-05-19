package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MessageHandler struct {
	messageService *service.MessageService
	friendService  *service.FriendService
	db             *gorm.DB
}

func NewMessageHandler(messageService *service.MessageService, friendService *service.FriendService, db *gorm.DB) *MessageHandler {
	return &MessageHandler{messageService: messageService, friendService: friendService, db: db}
}

// SendMessage sends a message to another user.
func (h *MessageHandler) SendMessage(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		ReceiverID uint64 `json:"receiver_id" binding:"required"`
		Content    string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	msg, err := h.messageService.SendMessage(userID, req.ReceiverID, req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": msg})
}

// GetConversations returns all conversations for the current user.
func (h *MessageHandler) GetConversations(c *gin.Context) {
	userID := c.GetUint64("user_id")

	conversations, err := h.messageService.GetConversations(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": conversations})
}

// GetMessages returns messages with a specific user.
func (h *MessageHandler) GetMessages(c *gin.Context) {
	userID := c.GetUint64("user_id")
	otherID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid user id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	messages, total, err := h.messageService.GetMessages(userID, otherID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items":     messages,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// MarkAsRead marks all messages from a specific user as read.
func (h *MessageHandler) MarkAsRead(c *gin.Context) {
	userID := c.GetUint64("user_id")
	otherID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid user id"})
		return
	}

	if err := h.messageService.MarkAsRead(userID, otherID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "messages marked as read"})
}

// UnreadCount returns the total unread message count.
func (h *MessageHandler) UnreadCount(c *gin.Context) {
	userID := c.GetUint64("user_id")

	count, err := h.messageService.UnreadCount(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"count": count}})
}
