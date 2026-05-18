package handler

import (
	"net/http"
	"strings"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

type DDNSHandler struct {
	ddnsService *service.DDNSService
}

func NewDDNSHandler(ddnsService *service.DDNSService) *DDNSHandler {
	return &DDNSHandler{ddnsService: ddnsService}
}

// Callback handles ddns-go callback requests with #{ip}, #{domain} variables
func (h *DDNSHandler) Callback(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		Domain     string `json:"domain" binding:"required"`
		IP         string `json:"ip" binding:"required"`
		RecordType string `json:"record_type"`
		TTL        int    `json:"ttl"`
		Token      string `json:"token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	result, err := h.ddnsService.Update(userID, req.Domain, req.IP, req.RecordType, req.TTL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// Webhook handles ddns-go webhook requests with aggregated domain data
func (h *DDNSHandler) Webhook(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req struct {
		IPv4Addr    string `json:"ipv4Addr"`
		IPv4Domains string `json:"ipv4Domains"`
		IPv6Addr    string `json:"ipv6Addr"`
		IPv6Domains string `json:"ipv6Domains"`
		Token       string `json:"token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var results []service.DDNSUpdateResult
	var errors []string

	// Process IPv4 domains
	if req.IPv4Addr != "" && req.IPv4Domains != "" {
		domains := strings.Split(req.IPv4Domains, ",")
		for _, domain := range domains {
			domain = strings.TrimSpace(domain)
			if domain == "" {
				continue
			}
			result, err := h.ddnsService.Update(userID, domain, req.IPv4Addr, "A", 600)
			if err != nil {
				errors = append(errors, domain+": "+err.Error())
			} else {
				results = append(results, *result)
			}
		}
	}

	// Process IPv6 domains
	if req.IPv6Addr != "" && req.IPv6Domains != "" {
		domains := strings.Split(req.IPv6Domains, ",")
		for _, domain := range domains {
			domain = strings.TrimSpace(domain)
			if domain == "" {
				continue
			}
			result, err := h.ddnsService.Update(userID, domain, req.IPv6Addr, "AAAA", 600)
			if err != nil {
				errors = append(errors, domain+": "+err.Error())
			} else {
				results = append(results, *result)
			}
		}
	}

	code := 0
	message := "success"
	if len(errors) > 0 {
		code = 1
		message = "partial success: " + strings.Join(errors, "; ")
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    code,
		"message": message,
		"data": gin.H{
			"results": results,
			"errors":  errors,
		},
	})
}
