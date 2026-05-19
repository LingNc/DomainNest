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
		return nil, errors.New("cannot send message to yourself")
	}

	if content == "" {
		return nil, errors.New("message content cannot be empty")
	}

	// Check receiver exists
	var receiver model.User
	if err := s.db.First(&receiver, receiverID).Error; err != nil {
		return nil, errors.New("receiver not found")
	}

	msg := &model.Message{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    content,
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
		WHERE sender_id = ? OR receiver_id = ?
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
		s.db.Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
			userID, pid, pid, userID).
			Order("created_at DESC").First(&lastMsg)

		// Count unread messages (sent by partner to me, with read_at IS NULL)
		var unread int64
		s.db.Model(&model.Message{}).Where("sender_id = ? AND receiver_id = ? AND read_at IS NULL", pid, userID).Count(&unread)

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
