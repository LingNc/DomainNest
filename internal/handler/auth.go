package handler

import (
	"net/http"

	"domainnest/internal/config"
	"domainnest/internal/middleware"
	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *service.AuthService
	jwtSecret   string
	jwtExpire   int
}

func NewAuthHandler(authService *service.AuthService, cfg *config.JWTConfig) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		jwtSecret:   cfg.Secret,
		jwtExpire:   cfg.ExpireHours,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3,max=64"`
		Password string `json:"password" binding:"required,min=6"`
		Email    string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	user, err := h.authService.Register(req.Username, req.Password, req.Email)
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
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid credentials"})
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
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"role":       user.Role,
			"ddns_token": user.Token,
		},
	})
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
