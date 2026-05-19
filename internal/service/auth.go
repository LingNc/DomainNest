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
	if inviter.InviteCount >= inviter.InviteLimit {
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
		InviteLimit: 5,
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

func (s *AuthService) UpdateProfile(userID uint64, nickname, phone, email string) error {
	updates := map[string]interface{}{
		"nickname": nickname,
		"phone":    phone,
		"email":    email,
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
