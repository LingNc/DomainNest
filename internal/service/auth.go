package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"domainnest/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db                *gorm.DB
	inviteCodeService *InviteCodeService
}

func NewAuthService(db *gorm.DB, inviteCodeService *InviteCodeService) *AuthService {
	return &AuthService{db: db, inviteCodeService: inviteCodeService}
}

func (s *AuthService) Register(username, password, email, inviteCode string) (*model.User, error) {
	var existing model.User
	if err := s.db.Where("username = ?", username).First(&existing).Error; err == nil {
		return nil, errors.New("用户名已存在")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:    username,
		Password:    string(hashedPassword),
		Email:       email,
		Role:        "user",
		Token:       token,
		InviteLimit: 0,
	}

	tx := s.db.Begin()
	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Validate and consume single-use invite code
	inviterID, err := s.inviteCodeService.ConsumeCode(inviteCode, user.ID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	invitedBy := inviterID
	user.InvitedBy = &invitedBy

	if err := tx.Model(user).Update("invited_by", invitedBy).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Create InviteLog for registration
	if err := tx.Create(&model.InviteLog{
		InviterID: inviterID,
		InviteeID: user.ID,
		Action:    "register",
		Amount:    1,
	}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(username, password string) (*model.User, error) {
	var user model.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, errors.New("账号或密码错误")
	}

	if user.Status == 0 {
		return nil, errors.New("账号已被禁用")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("账号或密码错误")
	}

	return &user, nil
}

func (s *AuthService) GetUserByID(id uint64) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *AuthService) UpdateProfile(userID uint64, nickname, phone, email, avatar string) error {
	updates := map[string]interface{}{
		"nickname": nickname,
		"phone":    phone,
		"email":    email,
		"avatar":   avatar,
	}
	return s.db.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}

func (s *AuthService) UpdateUsername(userID uint64, newUsername string) error {
	if newUsername == "" {
		return errors.New("用户名不能为空")
	}

	var existing model.User
	if err := s.db.Where("username = ? AND id != ?", newUsername, userID).First(&existing).Error; err == nil {
		return errors.New("用户名已被占用")
	}

	return s.db.Model(&model.User{}).Where("id = ?", userID).Update("username", newUsername).Error
}

func (s *AuthService) ResetToken(userID uint64) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	if err := s.db.Model(&model.User{}).Where("id = ?", userID).Update("token", token).Error; err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) ChangePassword(userID uint64, oldPassword, newPassword string) error {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("原密码错误")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.db.Model(&user).Update("password", string(hashedPassword)).Error
}

func (s *AuthService) AdminResetPassword(userID uint64, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.db.Model(&model.User{}).Where("id = ?", userID).Update("password", string(hashedPassword)).Error
}

func (s *AuthService) EnsureAdmin(username, password string) error {
	var admin model.User
	err := s.db.Where("role = ?", "admin").First(&admin).Error
	if err == nil {
		// Admin exists - ensure super_admin flag is set
		if !admin.IsSuperAdmin {
			s.db.Model(&admin).Update("is_super_admin", true)
		}
		return nil
	}

	user, err := s.createUser(username, password, "", "admin")
	if err != nil {
		return err
	}
	// First admin is super_admin
	return s.db.Model(user).Update("is_super_admin", true).Error
}

func (s *AuthService) createUser(username, password, email, role string) (*model.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:    username,
		Password:    string(hashedPassword),
		Email:       email,
		Role:        role,
		Token:       token,
		InviteLimit: 0,
	}

	return user, s.db.Create(user).Error
}

func generateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GrantInviteQuota allocates invite quota from inviter to invitee.
// If inviter is super_admin, pool check is skipped and invite_count is not incremented.
func (s *AuthService) GrantInviteQuota(inviterID, inviteeID uint64, amount int) error {
	if amount <= 0 {
		return errors.New("数量必须为正数")
	}

	var inviter model.User
	if err := s.db.First(&inviter, inviterID).Error; err != nil {
		return errors.New("邀请人不存在")
	}

	var invitee model.User
	if err := s.db.First(&invitee, inviteeID).Error; err != nil {
		return errors.New("被邀请人不存在")
	}

	if inviterID == inviteeID && !inviter.IsSuperAdmin {
		return errors.New("不能给自己分配邀请额度")
	}

	if invitee.Status == 0 {
		return errors.New("不能给已禁用的用户分配邀请额度")
	}

	// Check inviter is not super_admin for pool check
	if !inviter.IsSuperAdmin {
		if inviter.InviteLimit-inviter.InviteCount < amount {
			return errors.New("邀请额度不足")
		}
	}

	tx := s.db.Begin()

	// Increase inviter's allocated count (skip for super_admin)
	if !inviter.IsSuperAdmin {
		if err := tx.Model(&inviter).UpdateColumn("invite_count", gorm.Expr("invite_count + ?", amount)).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// Increase invitee's invite limit
	if err := tx.Model(&invitee).UpdateColumn("invite_limit", gorm.Expr("invite_limit + ?", amount)).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Create InviteLog
	if err := tx.Create(&model.InviteLog{
		InviterID: inviterID,
		InviteeID: inviteeID,
		Action:    "grant",
		Amount:    amount,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// RevokeInviteQuota revokes unused invite quota from invitee back to inviter.
func (s *AuthService) RevokeInviteQuota(inviterID, inviteeID uint64, amount int) error {
	if amount <= 0 {
		return errors.New("数量必须为正数")
	}

	var inviter model.User
	if err := s.db.First(&inviter, inviterID).Error; err != nil {
		return errors.New("邀请人不存在")
	}

	if inviterID == inviteeID && !inviter.IsSuperAdmin {
		return errors.New("不能撤销自己的邀请额度")
	}

	var invitee model.User
	if err := s.db.First(&invitee, inviteeID).Error; err != nil {
		return errors.New("被邀请人不存在")
	}

	// Authorization: check that inviter has granted quota to invitee that hasn't been fully revoked
	var totalGranted int
	s.db.Model(&model.InviteLog{}).Where("inviter_id = ? AND invitee_id = ? AND action = ?", inviterID, inviteeID, "grant").Select("COALESCE(SUM(amount),0)").Scan(&totalGranted)

	var totalRevoked int
	s.db.Model(&model.InviteLog{}).Where("inviter_id = ? AND invitee_id = ? AND action = ?", inviterID, inviteeID, "revoke").Select("COALESCE(SUM(amount),0)").Scan(&totalRevoked)

	maxRevokable := totalGranted - totalRevoked
	if !inviter.IsSuperAdmin && maxRevokable <= 0 {
		return errors.New("该用户无可撤销的额度")
	}
	if !inviter.IsSuperAdmin && amount > maxRevokable {
		amount = maxRevokable
	}

	// Revokeable = unused quota the invitee has
	revokeable := invitee.InviteLimit - invitee.InviteCount
	if revokeable <= 0 {
		return errors.New("无可撤销的邀请额度")
	}
	if amount > revokeable {
		amount = revokeable
	}

	tx := s.db.Begin()

	// Decrease invitee's invite limit
	if err := tx.Model(&invitee).UpdateColumn("invite_limit", gorm.Expr("invite_limit - ?", amount)).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Decrease inviter's invite count (never below 0, skip for super_admin)
	if !inviter.IsSuperAdmin {
		if err := tx.Model(&inviter).
			UpdateColumn("invite_count", gorm.Expr("GREATEST(invite_count - ?, 0)", amount)).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// Create InviteLog
	if err := tx.Create(&model.InviteLog{
		InviterID: inviterID,
		InviteeID: inviteeID,
		Action:    "revoke",
		Amount:    amount,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *AuthService) DeleteAccount(userID uint64) error {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return errors.New("用户不存在")
	}

	if user.IsSuperAdmin {
		return errors.New("超级管理员账号不可注销")
	}

	tx := s.db.Begin()

	// Determine transfer target for domains/permissions
	var transferTargetID *uint64
	if user.InvitedBy != nil {
		transferTargetID = user.InvitedBy
	} else if user.IsSuperAdmin {
		// Find another active admin to transfer to
		var otherAdmin model.User
		if err := tx.Where("role = 'admin' AND id != ? AND status = 1", userID).First(&otherAdmin).Error; err == nil {
			tid := otherAdmin.ID
			transferTargetID = &tid
		} else {
			tx.Rollback()
			return errors.New("无法注销：请先提升其他管理员以转移您的域名")
		}
	}

	// If we have a transfer target, transfer dependencies
	if transferTargetID != nil {
		targetID := *transferTargetID

		// 1. Transfer owned domains to target
		if err := tx.Model(&model.DomainNode{}).Where("owner_id = ?", userID).
			Update("owner_id", targetID).Error; err != nil {
			tx.Rollback()
			return err
		}

		// 2. Delete permissions where user is grantee
		if err := tx.Where("user_id = ?", userID).Delete(&model.DomainPermission{}).Error; err != nil {
			tx.Rollback()
			return err
		}

		// 3. Transfer permissions where user is grantor to target
		if err := tx.Model(&model.DomainPermission{}).Where("created_by = ?", userID).
			Update("created_by", targetID).Error; err != nil {
			tx.Rollback()
			return err
		}

		// Re-read user inside transaction for consistent quota snapshot
		if err := tx.First(&user, userID).Error; err != nil {
			tx.Rollback()
			return err
		}
		// 4. Reclaim unused invite quota to target
		unusedQuota := user.InviteLimit - user.InviteCount
		if unusedQuota > 0 {
			if err := tx.Model(&model.User{}).Where("id = ?", targetID).
				UpdateColumn("invite_count", gorm.Expr("GREATEST(invite_count - ?, 0)", unusedQuota)).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
		// Also free the registration slot
		if err := tx.Model(&model.User{}).Where("id = ?", targetID).
			UpdateColumn("invite_count", gorm.Expr("GREATEST(invite_count - 1, 0)")).Error; err != nil {
			tx.Rollback()
			return err
		}

		// 5. Create InviteLog for the reclaim (unused quota + registration slot)
		tx.Create(&model.InviteLog{
			InviterID: targetID,
			InviteeID: userID,
			Action:    "revoke",
			Amount:    unusedQuota + 1,
		})
	} else {
		// No inviter — find a SuperAdmin to transfer to
		var superAdmin model.User
		if err := tx.Where("is_super_admin = ? AND id != ? AND status = 1", true, userID).First(&superAdmin).Error; err == nil {
			targetID := superAdmin.ID
			// Transfer owned domains
			if err := tx.Model(&model.DomainNode{}).Where("owner_id = ?", userID).
				Update("owner_id", targetID).Error; err != nil {
				tx.Rollback()
				return err
			}
			// Delete permissions where user is grantee
			if err := tx.Where("user_id = ?", userID).Delete(&model.DomainPermission{}).Error; err != nil {
				tx.Rollback()
				return err
			}
			// Transfer permissions where user is grantor
			if err := tx.Model(&model.DomainPermission{}).Where("created_by = ?", userID).
				Update("created_by", targetID).Error; err != nil {
				tx.Rollback()
				return err
			}
			// Re-read user inside transaction for consistent snapshot
			if err := tx.First(&user, userID).Error; err != nil {
				tx.Rollback()
				return err
			}
			// Reclaim unused quota
			unusedQuota := user.InviteLimit - user.InviteCount
			if unusedQuota > 0 {
				if err := tx.Model(&model.User{}).Where("id = ?", targetID).
					UpdateColumn("invite_count", gorm.Expr("GREATEST(invite_count - ?, 0)", unusedQuota)).Error; err != nil {
					tx.Rollback()
					return err
				}
			}
			// Free the registration slot
			if err := tx.Model(&model.User{}).Where("id = ?", targetID).
				UpdateColumn("invite_count", gorm.Expr("GREATEST(invite_count - 1, 0)")).Error; err != nil {
				tx.Rollback()
				return err
			}
			// Log the reclaim
			tx.Create(&model.InviteLog{
				InviterID: targetID,
				InviteeID: userID,
				Action:    "revoke",
				Amount:    unusedQuota + 1,
			})
		} else {
			// No SuperAdmin found — fallback: zero out
			if err := tx.Model(&model.DomainNode{}).Where("owner_id = ?", userID).
				Update("owner_id", 0).Error; err != nil {
				tx.Rollback()
				return err
			}
			if err := tx.Where("user_id = ?", userID).Delete(&model.DomainPermission{}).Error; err != nil {
				tx.Rollback()
				return err
			}
			tx.Model(&user).UpdateColumn("invite_limit", 0)
		}
	}

	// Clean up user's RAM tokens
	if err := tx.Where("user_id = ?", userID).Delete(&model.RAMToken{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Clean up friendships (both directions)
	tx.Where("user_id = ? OR friend_id = ?", userID, userID).Delete(&model.Friendship{})

	// Clean up friend requests
	tx.Where("sender_id = ? OR receiver_id = ?", userID, userID).Delete(&model.FriendRequest{})

	// Clean up messages (both sent and received)
	tx.Where("sender_id = ? OR receiver_id = ?", userID, userID).Delete(&model.Message{})

	// Soft-delete: anonymize user instead of hard delete (preserve audit trail)
	deletedName := fmt.Sprintf("deleted_%d", userID)
	if err := tx.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"username":     deletedName,
		"nickname":     "已注销用户",
		"email":        "",
		"phone":        "",
		"avatar":       "",
		"password":     "!",
		"status":       0,
		"invite_limit": 0,
		"invite_count": 0,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

