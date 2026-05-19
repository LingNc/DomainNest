package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func applyLogFilters(query *gorm.DB, c *gin.Context) *gorm.DB {
	if actions := c.Query("action"); actions != "" {
		actionList := strings.Split(actions, ",")
		query = query.Where("action IN ?", actionList)
	}
	if targetTypes := c.Query("target_type"); targetTypes != "" {
		types := strings.Split(targetTypes, ",")
		query = query.Where("target_type IN ?", types)
	}
	if q := c.Query("q"); q != "" {
		like := "%" + q + "%"
		query = query.Where("(action LIKE ? OR detail LIKE ?)", like, like)
	}
	if startTime := c.Query("start_time"); startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime := c.Query("end_time"); endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}
	return query
}
