package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"

	"domainnest/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db *gorm.DB
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

func (s *AuthService) Register(username, password, email, inviteCode string) (*model.User, error) {
	var existing model.User
	if err := s.db.Where("username = ?", username).First(&existing).Error; err == nil {
		return nil, errors.New("username already exists")
	}

	// 验证邀请码
	var inviter model.User
	if err := s.db.Where("invite_code = ?", inviteCode).First(&inviter).Error; err != nil {
		return nil, errors.New("invalid invite code")
	}
	// Pool check: inviter's available pool (InviteLimit - InviteCount) >= 1
	if inviter.InviteLimit-inviter.InviteCount < 1 {
		return nil, errors.New("invite limit reached")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	newInviteCode, err := generateInviteCode()
	if err != nil {
		return nil, err
	}

	invitedBy := inviter.ID
	user := &model.User{
		Username:    username,
		Password:    string(hashedPassword),
		Email:       email,
		Role:        "user",
		Token:       token,
		InvitedBy:   &invitedBy,
		InviteCode:  newInviteCode,
		InviteLimit: 0,
	}

	tx := s.db.Begin()
	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Model(&inviter).UpdateColumn("invite_count", gorm.Expr("invite_count + 1")).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Create InviteLog for registration
	if err := tx.Create(&model.InviteLog{
		InviterID: inviter.ID,
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
		return nil, errors.New("invalid credentials")
	}

	if user.Status == 0 {
		return nil, errors.New("account disabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
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
		return errors.New("username cannot be empty")
	}

	var existing model.User
	if err := s.db.Where("username = ? AND id != ?", newUsername, userID).First(&existing).Error; err == nil {
		return errors.New("username already taken")
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
		return errors.New("incorrect old password")
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

	inviteCode, err := generateInviteCode()
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:    username,
		Password:    string(hashedPassword),
		Email:       email,
		Role:        role,
		Token:       token,
		InviteCode:  inviteCode,
		InviteLimit: 100,
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

func generateInviteCode() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GrantInviteQuota allocates invite quota from inviter to invitee.
// If inviter is super_admin, pool check is skipped and invite_count is not incremented.
func (s *AuthService) GrantInviteQuota(inviterID, inviteeID uint64, amount int) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}

	var inviter model.User
	if err := s.db.First(&inviter, inviterID).Error; err != nil {
		return errors.New("inviter not found")
	}

	var invitee model.User
	if err := s.db.First(&invitee, inviteeID).Error; err != nil {
		return errors.New("invitee not found")
	}

	// Check inviter is not super_admin for pool check
	if !inviter.IsSuperAdmin {
		if inviter.InviteLimit-inviter.InviteCount < amount {
			return errors.New("insufficient invite quota")
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
	var inviter model.User
	if err := s.db.First(&inviter, inviterID).Error; err != nil {
		return errors.New("inviter not found")
	}

	var invitee model.User
	if err := s.db.First(&invitee, inviteeID).Error; err != nil {
		return errors.New("invitee not found")
	}

	// Revokeable = unused quota the invitee has
	revokeable := invitee.InviteLimit - invitee.InviteCount
	if revokeable <= 0 {
		return errors.New("no revocable quota")
	}
	if amount > revokeable {
		amount = revokeable
	}
	if amount <= 0 {
		return errors.New("amount must be positive")
	}

	tx := s.db.Begin()

	// Decrease invitee's invite limit
	if err := tx.Model(&invitee).UpdateColumn("invite_limit", gorm.Expr("invite_limit - ?", amount)).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Decrease inviter's invite count (never below 0)
	if err := tx.Model(&inviter).
		UpdateColumn("invite_count", gorm.Expr("GREATEST(invite_count - ?, 0)", amount)).Error; err != nil {
		tx.Rollback()
		return err
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

// OnUserDeactivate handles invite pool cleanup when a user account is deactivated/deleted.
// Reduces the inviter's InviteCount by 1 (the registration slot is freed).
func (s *AuthService) OnUserDeactivate(userID uint64) error {
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}

	if user.InvitedBy == nil {
		return nil
	}

	// Free the registration slot from the inviter
	return s.db.Model(&model.User{}).Where("id = ?", *user.InvitedBy).
		UpdateColumn("invite_count", gorm.Expr("GREATEST(invite_count - 1, 0)")).Error
}

