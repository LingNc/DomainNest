package service

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"domainnest/internal/errs"
	"domainnest/internal/model"

	"gorm.io/gorm"
)

type InviteCodeService struct {
	db *gorm.DB
}

func NewInviteCodeService(db *gorm.DB) *InviteCodeService {
	return &InviteCodeService{db: db}
}

// GenerateCodes creates count unique invite codes for the given user.
// Consumes 1 quota per code (invite_count++). All users including SuperAdmin are subject to quota check.
func (s *InviteCodeService) GenerateCodes(creatorID uint64, count int) ([]model.InviteCode, error) {
	if count < 1 || count > 100 {
		count = 10
	}

	var user model.User
	if err := s.db.First(&user, creatorID).Error; err != nil {
		return nil, errs.New(errs.UserNotFound, "用户不存在")
	}

	available := user.InviteLimit - user.InviteCount
	if available < count {
		return nil, errs.New(errs.NoRevocableInviteQuotaGlobal, "邀请额度不足")
	}

	codes := make([]model.InviteCode, 0, count)
	for i := 0; i < count; i++ {
		code := generateSingleUseCode()
		codes = append(codes, model.InviteCode{Code: code, CreatorID: creatorID})
	}

	tx := s.db.Begin()

	if err := tx.Create(&codes).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Model(&user).UpdateColumn("invite_count", gorm.Expr("invite_count + ?", count)).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return codes, nil
}

// ListUserCodes lists invite codes created by a specific user.
func (s *InviteCodeService) ListUserCodes(userID uint64, page, pageSize int) ([]model.InviteCode, int64, error) {
	var total int64
	s.db.Model(&model.InviteCode{}).Where("creator_id = ?", userID).Count(&total)

	var codes []model.InviteCode
	s.db.Where("creator_id = ?", userID).
		Preload("UsedByUser", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "username", "nickname")
		}).
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&codes)

	return codes, total, nil
}

// ListCodes lists all invite codes (admin overview).
func (s *InviteCodeService) ListCodes(page, pageSize int) ([]model.InviteCode, int64, error) {
	var total int64
	s.db.Model(&model.InviteCode{}).Count(&total)

	var codes []model.InviteCode
	s.db.Preload("Creator", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "username")
	}).
		Preload("UsedByUser", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "username")
	}).
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&codes)

	return codes, total, nil
}

// DeleteCode deletes an unused invite code owned by the given user, restoring 1 quota.
func (s *InviteCodeService) DeleteCode(id, userID uint64) error {
	var code model.InviteCode
	if err := s.db.Where("id = ? AND creator_id = ?", id, userID).First(&code).Error; err != nil {
		return err
	}
	if code.UsedBy != nil {
		return errs.New(errs.CannotDeleteUsedInviteCode, "已使用的邀请码不能删除")
	}

	var creator model.User
	if err := s.db.First(&creator, code.CreatorID).Error; err != nil {
		return err
	}

	tx := s.db.Begin()

	if err := tx.Delete(&code).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&creator).UpdateColumn("invite_count", gorm.Expr("GREATEST(invite_count - 1, 0)")).Error; err != nil {
			tx.Rollback()
			return err
		}

	return tx.Commit().Error
}

// BatchDeleteCodes deletes multiple unused invite codes and restores quota.
// Only codes owned by the given userID are affected.
func (s *InviteCodeService) BatchDeleteCodes(ids []uint64, userID uint64) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return 0, errs.New(errs.UserNotFound, "用户不存在")
	}

	// Find unused codes owned by this user among the requested IDs
	var codes []model.InviteCode
	if err := s.db.Where("id IN ? AND creator_id = ? AND used_by IS NULL", ids, userID).Find(&codes).Error; err != nil {
		return 0, err
	}

	if len(codes) == 0 {
		return 0, nil
	}

	tx := s.db.Begin()

	if err := tx.Delete(&codes).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Model(&user).UpdateColumn("invite_count", gorm.Expr("GREATEST(invite_count - ?, 0)", len(codes))).Error; err != nil {
			tx.Rollback()
			return 0, err
		}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	return len(codes), nil
}

// ConsumeCode finds an unused code and marks it as used. Returns the creator ID.
// The db parameter should be the calling transaction (tx) to ensure atomicity
// and avoid FK lock-waits across connections.
func (s *InviteCodeService) ConsumeCode(db *gorm.DB, codeStr string, userID uint64) (uint64, error) {
	var code model.InviteCode
	err := db.Where("code = ? AND used_by IS NULL", codeStr).First(&code).Error
	if err != nil {
		return 0, errs.New(errs.InviteCodeInvalid, "邀请码无效或已使用")
	}
	now := time.Now()
	updates := map[string]interface{}{"used_by": userID, "used_at": now}
	if err := db.Model(&code).Updates(updates).Error; err != nil {
		return 0, err
	}
	return code.CreatorID, nil
}

func generateSingleUseCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b) // 8 chars
}
