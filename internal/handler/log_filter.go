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
	if actionExclude := c.Query("action_exclude"); actionExclude != "" {
		excludeList := strings.Split(actionExclude, ",")
		query = query.Where("action NOT IN ?", excludeList)
	}
	if targetTypes := c.Query("target_type"); targetTypes != "" {
		types := strings.Split(targetTypes, ",")
		query = query.Where("target_type IN ?", types)
	}
	if targetTypeExclude := c.Query("target_type_exclude"); targetTypeExclude != "" {
		excludeList := strings.Split(targetTypeExclude, ",")
		query = query.Where("target_type NOT IN ?", excludeList)
	}
	if q := c.Query("q"); q != "" {
		like := "%" + q + "%"
		query = query.Where("(action LIKE ? OR detail LIKE ?)", like, like)
	}
	if qExclude := c.Query("q_exclude"); qExclude != "" {
		notLike := "%" + qExclude + "%"
		query = query.Where("action NOT LIKE ? AND detail NOT LIKE ?", notLike, notLike)
	}
	if startTime := c.Query("start_time"); startTime != "" {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime := c.Query("end_time"); endTime != "" {
		query = query.Where("created_at <= ?", endTime)
	}
	if targetUserID := c.Query("target_user_id"); targetUserID != "" {
		query = query.Where("target_user_id = ?", targetUserID)
	}
	return query
}
