package notification

import "time"

// Priority levels for notifications.
type Priority int

const (
	PriorityInfo    Priority = 0
	PriorityWarning Priority = 1
	PriorityError   Priority = 2
)

// Notification is the value object used to create system messages.
type Notification struct {
	Category   string
	Title      string
	Content    string
	ActionType string
	ActionData string
	TargetType string
	TargetID   uint64
	Priority   Priority
	ExpiresAt  *time.Time
}
