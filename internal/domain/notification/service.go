package notification

import (
	"domainnest/internal/model"
	"domainnest/internal/ws"

	"gorm.io/gorm"
)

// Service handles creating and managing system notifications.
type Service struct {
	db *gorm.DB
}

// NewService creates a new notification service.
func NewService(db *gorm.DB, hub *ws.Hub) *Service {
	return &Service{db: db}
}

// Send creates a system notification for a single user and broadcasts via WebSocket.
func (s *Service) Send(receiverID uint64, n Notification) error {
	msg := &model.Message{
		SenderID:   nil, // system
		ReceiverID: receiverID,
		Type:       "system",
		Category:   n.Category,
		Title:      n.Title,
		Content:    n.Content,
		ActionType: n.ActionType,
		ActionData: n.ActionData,
		TargetType: n.TargetType,
		TargetID:   n.TargetID,
		Priority:   int(n.Priority),
		ExpiresAt:  n.ExpiresAt,
	}
	if err := s.db.Create(msg).Error; err != nil {
		return err
	}
	// Broadcast via WebSocket
	ws.BroadcastToUser(receiverID, ws.TypeNewNotification, msg)
	return nil
}

// SendToMultiple sends the same notification to multiple users.
func (s *Service) SendToMultiple(receiverIDs []uint64, n Notification) error {
	for _, id := range receiverIDs {
		if err := s.Send(id, n); err != nil {
			return err
		}
	}
	return nil
}

// PurgeExpired deletes all expired notifications. Returns the number of rows deleted.
func (s *Service) PurgeExpired() int64 {
	result := s.db.Where("expires_at IS NOT NULL AND expires_at < NOW()").Delete(&model.Message{})
	return result.RowsAffected
}
