package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"

	"domainnest/internal/service"

	"github.com/gin-gonic/gin"
)

type AliyunCompatHandler struct {
	svc *service.AliyunCompatService
}

func NewAliyunCompatHandler(svc *service.AliyunCompatService) *AliyunCompatHandler {
	return &AliyunCompatHandler{svc: svc}
}

func (h *AliyunCompatHandler) Dispatch(c *gin.Context) {
	action := c.Query("Action")
	switch action {
	case "DescribeDomains":
		h.DescribeDomains(c)
	case "DescribeDomainRecords":
		h.DescribeDomainRecords(c)
	case "AddDomainRecord":
		h.AddDomainRecord(c)
	case "UpdateDomainRecord":
		h.UpdateDomainRecord(c)
	case "DeleteDomainRecord":
		h.DeleteDomainRecord(c)
	default:
		h.writeError(c, "InvalidAction.NotFound", "The specified action is not valid.")
	}
}

func (h *AliyunCompatHandler) DescribeDomains(c *gin.Context) {
	userID := c.GetUint64("user_id")
	domains, err := h.svc.DescribeDomains(userID)
	if err != nil {
		h.writeError(c, "InternalError", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"RequestId":   genReqID(),
		"TotalCount":  len(domains),
		"PageNumber":  1,
		"PageSize":    50,
		"Domains":     gin.H{"Domain": domains},
	})
}

func (h *AliyunCompatHandler) DescribeDomainRecords(c *gin.Context) {
	userID := c.GetUint64("user_id")
	domainName := c.Query("DomainName")
	if domainName == "" {
		h.writeError(c, "MissingParameter", "DomainName is required.")
		return
	}
	rrKeyword := c.Query("RRKeyWord")
	typeKeyword := c.Query("TypeKeyWord")
	valueKeyword := c.Query("ValueKeyWord")
	pageNumber, _ := strconv.Atoi(c.DefaultQuery("PageNumber", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("PageSize", "20"))

	result, _, err := h.svc.DescribeDomainRecords(userID, domainName, rrKeyword, typeKeyword, valueKeyword, pageNumber, pageSize)
	if err != nil {
		h.writeError(c, "InvalidDomainName.NoExist", "The specified domain name does not exist.")
		return
	}

	var records []gin.H
	for _, r := range result.Items {
		status := "Enable"
		if !r.Enabled {
			status = "Disable"
		}
		records = append(records, gin.H{
			"RecordId":   strconv.FormatUint(r.ID, 10),
			"DomainName": domainName,
			"RR":         r.Host,
			"Type":       r.RecordType,
			"Value":      r.Value,
			"TTL":        r.TTL,
			"Priority":   r.Priority,
			"Line":       r.Line,
			"Status":     status,
			"Locked":     false,
		})
	}
	if records == nil {
		records = []gin.H{}
	}

	c.JSON(http.StatusOK, gin.H{
		"RequestId":      genReqID(),
		"TotalCount":     result.Total,
		"PageNumber":     result.Page,
		"PageSize":       result.PageSize,
		"DomainRecords":  gin.H{"Record": records},
	})
}

func (h *AliyunCompatHandler) AddDomainRecord(c *gin.Context) {
	userID := c.GetUint64("user_id")
	domainName := c.Query("DomainName")
	rr := c.Query("RR")
	recordType := c.Query("Type")
	value := c.Query("Value")
	ttl, _ := strconv.Atoi(c.DefaultQuery("TTL", "600"))
	if ttl == 0 { ttl = 600 }

	if domainName == "" || rr == "" || recordType == "" || value == "" {
		h.writeError(c, "MissingParameter", "Required parameters are missing.")
		return
	}

	var priority *int
	if p := c.Query("Priority"); p != "" {
		v, _ := strconv.Atoi(p)
		priority = &v
	}
	line := c.DefaultQuery("Line", "default")

	record, err := h.svc.AddDomainRecord(userID, domainName, rr, recordType, value, ttl, priority, line)
	if err != nil {
		h.writeError(c, "DomainRecordDuplicate", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"RequestId": genReqID(),
		"RecordId":  strconv.FormatUint(record.ID, 10),
		"RR":        record.Host,
		"Type":      record.RecordType,
		"Value":     record.Value,
		"TTL":       record.TTL,
		"Line":      record.Line,
	})
}

func (h *AliyunCompatHandler) UpdateDomainRecord(c *gin.Context) {
	userID := c.GetUint64("user_id")
	recordID := c.Query("RecordId")
	rr := c.Query("RR")
	recordType := c.Query("Type")
	value := c.Query("Value")
	ttl, _ := strconv.Atoi(c.DefaultQuery("TTL", "600"))

	if recordID == "" || rr == "" || recordType == "" || value == "" {
		h.writeError(c, "MissingParameter", "Required parameters are missing.")
		return
	}

	var priority *int
	if p := c.Query("Priority"); p != "" {
		v, _ := strconv.Atoi(p)
		priority = &v
	}

	record, err := h.svc.UpdateDomainRecord(userID, recordID, rr, recordType, value, ttl, priority)
	if err != nil {
		h.writeError(c, "DomainRecordNotBelongToUser", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"RequestId": genReqID(),
		"RecordId":  strconv.FormatUint(record.ID, 10),
		"RR":        record.Host,
		"Type":      record.RecordType,
		"Value":     record.Value,
		"TTL":       record.TTL,
	})
}

func (h *AliyunCompatHandler) DeleteDomainRecord(c *gin.Context) {
	userID := c.GetUint64("user_id")
	recordID := c.Query("RecordId")
	if recordID == "" {
		h.writeError(c, "MissingParameter", "RecordId is required.")
		return
	}

	err := h.svc.DeleteDomainRecord(userID, recordID)
	if err != nil {
		h.writeError(c, "DomainRecordNotBelongToUser", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"RequestId": genReqID(),
		"RecordId":  recordID,
	})
}

func (h *AliyunCompatHandler) writeError(c *gin.Context, code, message string) {
	c.JSON(http.StatusOK, gin.H{
		"RequestId": genReqID(),
		"HostId":    "alidns",
		"Code":      code,
		"Message":   message,
	})
}

func genReqID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}