package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"domainnest/internal/model"

	"gorm.io/gorm"
)

type InviteCodeService struct {
	db *gorm.DB
}

func NewInviteCodeService(db *gorm.DB) *InviteCodeService {
	return &InviteCodeService{db: db}
}

func (s *InviteCodeService) GenerateCodes(creatorID uint64, count int) ([]model.InviteCode, error) {
	if count < 1 || count > 100 {
		count = 10
	}
	codes := make([]model.InviteCode, 0, count)
	for i := 0; i < count; i++ {
		code := generateSingleUseCode()
		ic := model.InviteCode{Code: code, CreatorID: creatorID}
		if err := s.db.Create(&ic).Error; err != nil {
			return nil, err
		}
		codes = append(codes, ic)
	}
	return codes, nil
}

func (s *InviteCodeService) ListCodes(page, pageSize int) ([]model.InviteCode, int64, error) {
	var total int64
	s.db.Model(&model.InviteCode{}).Count(&total)
	var codes []model.InviteCode
	s.db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&codes)
	return codes, total, nil
}

func (s *InviteCodeService) DeleteCode(id uint64) error {
	var code model.InviteCode
	if err := s.db.First(&code, id).Error; err != nil {
		return err
	}
	if code.UsedBy != nil {
		return errors.New("已使用的邀请码不能删除")
	}
	return s.db.Delete(&code).Error
}

// ConsumeCode finds an unused code and marks it as used. Returns the creator ID.
func (s *InviteCodeService) ConsumeCode(codeStr string, userID uint64) (uint64, error) {
	var code model.InviteCode
	err := s.db.Where("code = ? AND used_by IS NULL", codeStr).First(&code).Error
	if err != nil {
		return 0, errors.New("邀请码无效或已使用")
	}
	now := time.Now()
	updates := map[string]interface{}{"used_by": userID, "used_at": now}
	if err := s.db.Model(&code).Updates(updates).Error; err != nil {
		return 0, err
	}
	return code.CreatorID, nil
}

func generateSingleUseCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b) // 8 chars
}
