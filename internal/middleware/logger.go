package middleware

import (
	"encoding/json"

	"domainnest/internal/model"

	"gorm.io/gorm"
)

func LogOperation(db *gorm.DB, userID uint64, action, targetType string, targetID *uint64, detail interface{}, ip string) {
	detailJSON, _ := json.Marshal(detail)
	db.Create(&model.OperationLog{
		UserID:     userID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     string(detailJSON),
		IPAddress:  ip,
	})
}
