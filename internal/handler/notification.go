package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

// NotificationHandler handles user-facing notification endpoints.
type NotificationHandler struct {
	messageService *service.MessageService
}

func NewNotificationHandler(messageService *service.MessageService) *NotificationHandler {
	return &NotificationHandler{messageService: messageService}
}

// List returns the current user's notifications (paginated, filterable by category, read/unread).
func (h *NotificationHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	category := c.Query("category")
	filter := service.NotificationFilter{Category: category}
	if v := c.Query("is_read"); v == "true" {
		t := true
		filter.IsRead = &t
	} else if v == "false" {
		f := false
		filter.IsRead = &f
	}

	notifications, total, err := h.messageService.GetNotificationsFiltered(userID, page, pageSize, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items":     notifications,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// MarkAsRead marks a single notification as read.
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID := c.GetUint64("user_id")
	notifID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的通知ID"})
		return
	}
	if err := h.messageService.MarkNotificationAsRead(userID, notifID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已标记为已读"})
}

// MarkAllAsRead marks all of the current user's notifications as read.
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := c.GetUint64("user_id")
	if err := h.messageService.MarkAllNotificationsAsRead(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已全部标记为已读"})
}

// UnreadCount returns the current user's unread notification count.
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userID := c.GetUint64("user_id")
	count, err := h.messageService.GetNotificationUnreadCount(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"count": count}})
}

// Delete removes a specific notification belonging to the current user.
func (h *NotificationHandler) Delete(c *gin.Context) {
	userID := c.GetUint64("user_id")
	notifID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的通知ID"})
		return
	}
	if err := h.messageService.DeleteNotification(userID, notifID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "通知已删除"})
}
