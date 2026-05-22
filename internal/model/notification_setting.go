package model

// NotificationSetting stores per-user mute preferences for notification categories.
type NotificationSetting struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID   uint64 `gorm:"index:idx_user_category;not null" json:"user_id"`
	Category string `gorm:"type:varchar(40);not null;index:idx_user_category" json:"category"`
	Muted    bool   `gorm:"default:false" json:"muted"`
}

func (NotificationSetting) TableName() string {
	return "notification_settings"
}
