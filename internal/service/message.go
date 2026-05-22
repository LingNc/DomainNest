package service

import (
	"errors"
	"time"

	"domainnest/internal/model"

	"gorm.io/gorm"
)

type MessageService struct {
	db *gorm.DB
}

func NewMessageService(db *gorm.DB) *MessageService {
	return &MessageService{db: db}
}

// Conversation represents a conversation summary with another user.
type Conversation struct {
	User         model.User `json:"user"`
	LastMessage  string     `json:"last_message"`
	LastMsgTime  time.Time  `json:"last_msg_time"`
	UnreadCount  int64      `json:"unread_count"`
}

// SendMessage sends a message from sender to receiver.
func (s *MessageService) SendMessage(senderID, receiverID uint64, content string) (*model.Message, error) {
	if senderID == receiverID {
		return nil, errors.New("不能给自己发送消息")
	}

	if content == "" {
		return nil, errors.New("消息内容不能为空")
	}

	// Check receiver exists
	var receiver model.User
	if err := s.db.First(&receiver, receiverID).Error; err != nil {
		return nil, errors.New("接收者不存在")
	}

	msg := &model.Message{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    content,
		Type:       "user",
	}
	if err := s.db.Create(msg).Error; err != nil {
		return nil, err
	}

	// Reload with sender info
	s.db.Preload("Sender").First(msg, msg.ID)
	return msg, nil
}

// GetConversations returns all conversations for a user, sorted by last message time.
func (s *MessageService) GetConversations(userID uint64) ([]Conversation, error) {
	// Get all user IDs that have exchanged messages with this user
	type partnerRow struct {
		PartnerID uint64
	}

	var partnerIDs []uint64
	s.db.Raw(`
		SELECT DISTINCT
			CASE WHEN sender_id = ? THEN receiver_id ELSE sender_id END AS partner_id
		FROM messages
		WHERE (sender_id = ? OR receiver_id = ?) AND type = 'user'
	`, userID, userID, userID).Scan(&partnerIDs)

	if len(partnerIDs) == 0 {
		return []Conversation{}, nil
	}

	// Load partner users
	var users []model.User
	s.db.Where("id IN ?", partnerIDs).Find(&users)
	userMap := make(map[uint64]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	conversations := make([]Conversation, 0, len(partnerIDs))
	for _, pid := range partnerIDs {
		// Get last message
		var lastMsg model.Message
		s.db.Where("((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)) AND type = 'user'",
			userID, pid, pid, userID).
			Order("created_at DESC").First(&lastMsg)

		// Count unread messages (sent by partner to me, with read_at IS NULL)
		var unread int64
		s.db.Model(&model.Message{}).Where("sender_id = ? AND receiver_id = ? AND read_at IS NULL AND type = 'user'", pid, userID).Count(&unread)

		user := userMap[pid]
		// Clear sensitive fields
		user.Password = ""
		user.Token = ""

		conversations = append(conversations, Conversation{
			User:        user,
			LastMessage: lastMsg.Content,
			LastMsgTime: lastMsg.CreatedAt,
			UnreadCount: unread,
		})
	}

	// Sort by last message time descending
	for i := 0; i < len(conversations)-1; i++ {
		for j := i + 1; j < len(conversations); j++ {
			if conversations[j].LastMsgTime.After(conversations[i].LastMsgTime) {
				conversations[i], conversations[j] = conversations[j], conversations[i]
			}
		}
	}

	return conversations, nil
}

// GetMessages returns messages between two users with pagination.
func (s *MessageService) GetMessages(userID, otherID uint64, page, pageSize int) ([]model.Message, int64, error) {
	query := s.db.Model(&model.Message{}).Where(
		"(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
		userID, otherID, otherID, userID,
	)

	var total int64
	query.Count(&total)

	var messages []model.Message
	err := query.Preload("Sender").Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&messages).Error

	// Reverse so oldest first in page
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, total, err
}

// MarkAsRead marks all messages from otherUser to userID as read.
func (s *MessageService) MarkAsRead(userID, otherID uint64) error {
	now := time.Now()
	return s.db.Model(&model.Message{}).
		Where("sender_id = ? AND receiver_id = ? AND read_at IS NULL", otherID, userID).
		Update("read_at", now).Error
}

// UnreadCount returns the total number of unread messages for a user.
func (s *MessageService) UnreadCount(userID uint64) (int64, error) {
	var count int64
	err := s.db.Model(&model.Message{}).Where("receiver_id = ? AND read_at IS NULL", userID).Count(&count).Error
	return count, err
}

// GetNotifications returns system notifications for a user with pagination.
func (s *MessageService) GetNotifications(userID uint64, page, pageSize int) ([]model.Message, int64, error) {
	query := s.db.Model(&model.Message{}).Where("receiver_id = ? AND type = ?", userID, "system")
	var total int64
	query.Count(&total)
	var messages []model.Message
	err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&messages).Error
	return messages, total, err
}

// GetNotificationUnreadCount returns unread system notification count.
func (s *MessageService) GetNotificationUnreadCount(userID uint64) (int64, error) {
	var count int64
	err := s.db.Model(&model.Message{}).Where("receiver_id = ? AND type = ? AND read_at IS NULL", userID, "system").Count(&count).Error
	return count, err
}

// MarkNotificationAsRead marks a specific notification as read.
func (s *MessageService) MarkNotificationAsRead(userID, notifID uint64) error {
	now := time.Now()
	return s.db.Model(&model.Message{}).Where("id = ? AND receiver_id = ? AND type = ?", notifID, userID, "system").Update("read_at", now).Error
}

// MarkAllNotificationsAsRead marks all notifications as read for a user.
func (s *MessageService) MarkAllNotificationsAsRead(userID uint64) error {
	now := time.Now()
	return s.db.Model(&model.Message{}).Where("receiver_id = ? AND type = ? AND read_at IS NULL", userID, "system").Update("read_at", now).Error
}

// SendSystemNotification creates a system notification (sender_id = 0).
func (s *MessageService) SendSystemNotification(receiverID uint64, title, content string, actionType string, actionData string) error {
	msg := &model.Message{
		SenderID:     0,
		ReceiverID:   receiverID,
		Content:      content,
		Type:         "system",
		Title:        title,
		ActionType:   actionType,
		ActionStatus: "",
		ActionData:   actionData,
	}
	return s.db.Create(msg).Error
}

// NotificationFilter holds optional filters for querying notifications.
type NotificationFilter struct {
	Category string
	IsRead   *bool // nil = don't filter, true = read only, false = unread only
}

// GetNotificationsFiltered returns system notifications with optional filters.
func (s *MessageService) GetNotificationsFiltered(userID uint64, page, pageSize int, filter NotificationFilter) ([]model.Message, int64, error) {
	query := s.db.Model(&model.Message{}).Where("receiver_id = ? AND type = ?", userID, "system")
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.IsRead != nil {
		if *filter.IsRead {
			query = query.Where("read_at IS NOT NULL")
		} else {
			query = query.Where("read_at IS NULL")
		}
	}
	var total int64
	query.Count(&total)
	var messages []model.Message
	err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&messages).Error
	return messages, total, err
}

// DeleteNotification deletes a specific notification belonging to a user.
func (s *MessageService) DeleteNotification(userID, notifID uint64) error {
	result := s.db.Where("id = ? AND receiver_id = ? AND type = ?", notifID, userID, "system").Delete(&model.Message{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("通知不存在")
	}
	return nil
}

// HandleNotificationAction processes an accept/reject action on a notification.
func (s *MessageService) HandleNotificationAction(userID, notifID uint64, action string) error {
	if action != "accepted" && action != "rejected" {
		return errors.New("无效的操作")
	}

	var msg model.Message
	if err := s.db.Where("id = ? AND receiver_id = ? AND type = ?", notifID, userID, "system").First(&msg).Error; err != nil {
		return errors.New("通知不存在")
	}
	if msg.ActionType == "" {
		return errors.New("该通知不支持操作")
	}
	if msg.ActionStatus != "" {
		return errors.New("该通知已处理")
	}

	return s.db.Model(&msg).Update("action_status", action).Error
}
