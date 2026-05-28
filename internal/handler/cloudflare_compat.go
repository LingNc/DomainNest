package handler

import (
	"net/http"
	"strconv"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

type CloudflareCompatHandler struct {
	svc *service.AliyunCompatService
}

func NewCloudflareCompatHandler(svc *service.AliyunCompatService) *CloudflareCompatHandler {
	return &CloudflareCompatHandler{svc: svc}
}

func (h *CloudflareCompatHandler) ListZones(c *gin.Context) {
	userID := c.GetUint64("user_id")
	domains, err := h.svc.DescribeDomains(userID)
	if err != nil {
		h.writeError(c, "InternalError", err.Error())
		return
	}

	var result []gin.H
	for _, d := range domains {
		result = append(result, gin.H{
			"id":      d["DomainName"],
			"name":    d["DomainName"],
			"status":  "active",
			"paused":  false,
			"type":    "full",
			"email":   d["RegistrantEmail"],
			"plan":    gin.H{"name": "Free"},
		})
	}
	if result == nil {
		result = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"result":   result,
		"errors":   []gin.H{},
		"messages": []gin.H{},
	})
}

func (h *CloudflareCompatHandler) ListRecords(c *gin.Context) {
	userID := c.GetUint64("user_id")
	zoneID := c.Param("zone_id")
	if zoneID == "" {
		h.writeError(c, "MissingParameter", "zone_id is required.")
		return
	}

	rrKeyword := c.Query("name")
	typeKeyword := c.Query("type")
	valueKeyword := c.Query("content")
	pageNumber, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	result, _, err := h.svc.DescribeDomainRecords(userID, zoneID, rrKeyword, typeKeyword, valueKeyword, pageNumber, pageSize)
	if err != nil {
		h.writeError(c, "InvalidDomainName.NoExist", "The specified domain name does not exist or access denied.")
		return
	}

	var records []gin.H
	for _, r := range result.Items {
		records = append(records, gin.H{
			"id":       strconv.FormatUint(r.ID, 10),
			"zone_id":  zoneID,
			"name":     r.Host,
			"type":     r.RecordType,
			"content":  r.Value,
			"ttl":      r.TTL,
			"priority": r.Priority,
			"proxied":  false,
			"locked":   false,
			"created":  r.CreatedAt.Unix(),
			"modified": r.UpdatedAt.Unix(),
		})
	}
	if records == nil {
		records = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"result":   records,
		"errors":   []gin.H{},
		"messages": []gin.H{},
	})
}

func (h *CloudflareCompatHandler) CreateRecord(c *gin.Context) {
	userID := c.GetUint64("user_id")
	zoneID := c.Param("zone_id")
	if zoneID == "" {
		h.writeError(c, "MissingParameter", "zone_id is required.")
		return
	}

	var body struct {
		Type    string `json:"type" binding:"required"`
		Name    string `json:"name" binding:"required"`
		Content string `json:"content" binding:"required"`
		TTL     int    `json:"ttl"`
		Priority int   `json:"priority"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		h.writeError(c, "MissingParameter", "Required parameters are missing.")
		return
	}

	ttl := body.TTL
	if ttl == 0 {
		ttl = 600
	}

	var priority *int
	if body.Priority > 0 {
		priority = &body.Priority
	}

	record, err := h.svc.AddDomainRecord(userID, zoneID, body.Name, body.Type, body.Content, ttl, priority, "default")
	if err != nil {
		h.writeError(c, "DomainRecordDuplicate", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result": gin.H{
			"id":       strconv.FormatUint(record.ID, 10),
			"zone_id":  zoneID,
			"name":     record.Host,
			"type":     record.RecordType,
			"content":  record.Value,
			"ttl":      record.TTL,
			"priority": record.Priority,
			"proxied":  false,
			"locked":   false,
			"created":  record.CreatedAt.Unix(),
			"modified": record.UpdatedAt.Unix(),
		},
		"errors":   []gin.H{},
		"messages": []gin.H{},
	})
}

func (h *CloudflareCompatHandler) UpdateRecord(c *gin.Context) {
	userID := c.GetUint64("user_id")
	zoneID := c.Param("zone_id")
	recordID := c.Param("record_id")
	if zoneID == "" || recordID == "" {
		h.writeError(c, "MissingParameter", "zone_id and record_id are required.")
		return
	}

	var body struct {
		Type    string `json:"type"`
		Name    string `json:"name"`
		Content string `json:"content"`
		TTL     int    `json:"ttl"`
		Priority int   `json:"priority"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		h.writeError(c, "MissingParameter", "Required parameters are missing.")
		return
	}

	recordType := body.Type
	if recordType == "" {
		recordType = "A"
	}
	name := body.Name
	if name == "" {
		name = zoneID
	}
	ttl := body.TTL
	if ttl == 0 {
		ttl = 600
	}

	var priority *int
	if body.Priority > 0 {
		priority = &body.Priority
	}

	record, err := h.svc.UpdateDomainRecord(userID, recordID, name, recordType, body.Content, ttl, priority)
	if err != nil {
		h.writeError(c, "DomainRecordNotBelongToUser", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"result": gin.H{
			"id":       strconv.FormatUint(record.ID, 10),
			"zone_id":  zoneID,
			"name":     record.Host,
			"type":     record.RecordType,
			"content":  record.Value,
			"ttl":      record.TTL,
			"priority": record.Priority,
			"proxied":  false,
			"locked":   false,
			"created":  record.CreatedAt.Unix(),
			"modified": record.UpdatedAt.Unix(),
		},
		"errors":   []gin.H{},
		"messages": []gin.H{},
	})
}

func (h *CloudflareCompatHandler) DeleteRecord(c *gin.Context) {
	userID := c.GetUint64("user_id")
	recordID := c.Param("record_id")
	if recordID == "" {
		h.writeError(c, "MissingParameter", "record_id is required.")
		return
	}

	err := h.svc.DeleteDomainRecord(userID, recordID)
	if err != nil {
		h.writeError(c, "DomainRecordNotBelongToUser", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"result":   gin.H{"id": recordID},
		"errors":   []gin.H{},
		"messages": []gin.H{},
	})
}

func (h *CloudflareCompatHandler) writeError(c *gin.Context, code, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"errors":   []gin.H{{"code": code, "message": message}},
	})
}