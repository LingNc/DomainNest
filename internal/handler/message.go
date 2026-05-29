package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/errs"
	"domainnest/internal/service"
	"domainnest/internal/ws"

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
		errs.JSONError(c, err)
		return
	}

	msg, err := h.messageService.SendMessage(userID, req.ReceiverID, req.Content)
	if err != nil {
		errs.JSONError(c, err)
		return
	}

	// Push new message to receiver via WebSocket
	ws.BroadcastToUser(req.ReceiverID, ws.TypeNewMessage, msg)
	if count, err := h.messageService.UnreadCount(req.ReceiverID); err == nil {
		ws.BroadcastToUser(req.ReceiverID, ws.TypeUnreadUpdate, gin.H{"count": count})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": msg})
}

// GetConversations returns all conversations for the current user.
func (h *MessageHandler) GetConversations(c *gin.Context) {
	userID := c.GetUint64("user_id")

	conversations, err := h.messageService.GetConversations(userID)
	if err != nil {
		errs.JSONError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": conversations})
}

// GetMessages returns messages with a specific user.
func (h *MessageHandler) GetMessages(c *gin.Context) {
	userID := c.GetUint64("user_id")
	otherID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidUserID)
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
		errs.JSONError(c, err)
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
		errs.JSONErrorCode(c, errs.InvalidUserID)
		return
	}

	if err := h.messageService.MarkAsRead(userID, otherID); err != nil {
		errs.JSONError(c, err)
		return
	}

	// Broadcast updated unread count to the current user
	if count, err := h.messageService.UnreadCount(userID); err == nil {
		ws.BroadcastToUser(userID, ws.TypeUnreadUpdate, gin.H{"count": count})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "消息已标记为已读"})
}

// UnreadCount returns the total unread message count.
func (h *MessageHandler) UnreadCount(c *gin.Context) {
	userID := c.GetUint64("user_id")

	count, err := h.messageService.UnreadCount(userID)
	if err != nil {
		errs.JSONError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"count": count}})
}

// GetNotifications returns system notifications for the current user.
func (h *MessageHandler) GetNotifications(c *gin.Context) {
	userID := c.GetUint64("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	notifications, total, err := h.messageService.GetNotifications(userID, page, pageSize)
	if err != nil {
		errs.JSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": notifications, "total": total, "page": page, "page_size": pageSize}})
}

// NotificationUnreadCount returns unread notification count.
func (h *MessageHandler) NotificationUnreadCount(c *gin.Context) {
	userID := c.GetUint64("user_id")
	count, err := h.messageService.GetNotificationUnreadCount(userID)
	if err != nil {
		errs.JSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"count": count}})
}

// MarkNotificationAsRead marks a notification as read.
func (h *MessageHandler) MarkNotificationAsRead(c *gin.Context) {
	userID := c.GetUint64("user_id")
	notifID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidNotificationID)
		return
	}
	if err := h.messageService.MarkNotificationAsRead(userID, notifID); err != nil {
		errs.JSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已标记为已读"})
}

// MarkAllNotificationsAsRead marks all notifications as read.
func (h *MessageHandler) MarkAllNotificationsAsRead(c *gin.Context) {
	userID := c.GetUint64("user_id")
	if err := h.messageService.MarkAllNotificationsAsRead(userID); err != nil {
		errs.JSONError(c, err)
		return
	}

	// Broadcast updated unread count to the current user
	if count, err := h.messageService.UnreadCount(userID); err == nil {
		ws.BroadcastToUser(userID, ws.TypeUnreadUpdate, gin.H{"count": count})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已全部标记为已读"})
}

// HandleNotificationAction processes an accept/reject action on a notification.
func (h *MessageHandler) HandleNotificationAction(c *gin.Context) {
	userID := c.GetUint64("user_id")
	notifID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		errs.JSONErrorCode(c, errs.InvalidNotificationID)
		return
	}
	var req struct {
		Action string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		errs.JSONError(c, err)
		return
	}
	if err := h.messageService.HandleNotificationAction(userID, notifID, req.Action); err != nil {
		errs.JSONError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "操作成功"})
}
