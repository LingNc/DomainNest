package handler

import (
	"net/http"
	"time"

	"domainnest/internal/config"
	"domainnest/internal/middleware"
	"domainnest/internal/model"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	authService *service.AuthService
	emailSvc    *service.EmailService
	db          *gorm.DB
	jwtSecret   string
	jwtExpire   int
}

func NewAuthHandler(authService *service.AuthService, emailSvc *service.EmailService, db *gorm.DB, cfg *config.JWTConfig) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		emailSvc:    emailSvc,
		db:          db,
		jwtSecret:   cfg.Secret,
		jwtExpire:   cfg.ExpireHours,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username   string `json:"username" binding:"required,min=3,max=64"`
		Password   string `json:"password" binding:"required,min=6"`
		Email      string `json:"email"`
		InviteCode string `json:"invite_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	user, err := h.authService.Register(req.Username, req.Password, req.Email, req.InviteCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	user, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": err.Error()})
		return
	}

	token, err := middleware.GenerateToken(h.jwtSecret, user.ID, user.Username, user.Role, h.jwtExpire)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"role":     user.Role,
			},
		},
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := c.GetUint64("user_id")

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":           user.ID,
			"username":     user.Username,
			"email":        user.Email,
			"nickname":     user.Nickname,
			"phone":        user.Phone,
			"avatar":       user.Avatar,
			"role":         user.Role,
			"ddns_token":   user.Token,
			"invite_code":  user.InviteCode,
			"invite_limit": user.InviteLimit,
			"invite_count": user.InviteCount,
		},
	})
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		Username string `json:"username"`
		Nickname string `json:"nickname"`
		Phone    string `json:"phone"`
		Email    string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if req.Username != "" {
		if err := h.authService.UpdateUsername(userID, req.Username); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
			return
		}
	}

	if err := h.authService.UpdateProfile(userID, req.Nickname, req.Phone, req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "profile updated"})
}

func (h *AuthHandler) ResetToken(c *gin.Context) {
	userID := c.GetUint64("user_id")

	newToken, err := h.authService.ResetToken(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to reset token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "token reset successfully",
		"data": gin.H{
			"token": newToken,
		},
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.authService.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "password changed successfully"})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 无论邮箱是否存在都返回成功（防枚举）
	var user model.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "if the email exists, a reset link has been sent"})
		return
	}

	token, err := service.GenerateToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to generate token"})
		return
	}

	reset := &model.PasswordReset{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	if err := h.db.Create(reset).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to create reset token"})
		return
	}

	origin := c.GetHeader("Origin")
	if origin == "" {
		origin = "http://" + c.Request.Host
	}
	resetLink := origin + "/reset-password?token=" + token

	go h.emailSvc.SendPasswordReset(user.Email, resetLink)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "if the email exists, a reset link has been sent"})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var reset model.PasswordReset
	if err := h.db.Where("token = ? AND used = false AND expires_at > ?", req.Token, time.Now()).First(&reset).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid or expired token"})
		return
	}

	if err := h.authService.AdminResetPassword(reset.UserID, req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "failed to reset password"})
		return
	}

	h.db.Model(&reset).Update("used", true)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "password reset successfully"})
}
