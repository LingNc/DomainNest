package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"domainnest/internal/model"

	"gorm.io/gorm"
)

var allowedRecordTypes = map[string]bool{
	"A": true, "AAAA": true, "CNAME": true, "ALIAS": true,
	"MX": true, "TXT": true, "CAA": true, "NS": true, "SRV": true,
}

func IsValidRecordType(t string) bool {
	return allowedRecordTypes[t]
}

type RecordService struct {
	db *gorm.DB
}

func NewRecordService(db *gorm.DB) *RecordService {
	return &RecordService{db: db}
}

type RecordQuery struct {
	Host       string
	RecordType string
	Value      string
	Enabled    *bool
	SyncStatus string
	Page       int
	PageSize   int
}

type RecordListResult struct {
	Items    []model.DNSRecord `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

func (s *RecordService) ListRecords(nodeID, userID uint64, q RecordQuery) (*RecordListResult, error) {
	var node model.DomainNode
	if err := s.db.Where("id = ? AND owner_id = ?", nodeID, userID).First(&node).Error; err != nil {
		return nil, errors.New("domain node not found or access denied")
	}

	query := s.db.Model(&model.DNSRecord{}).Where("node_id = ?", nodeID)
	if q.Host != "" {
		query = query.Where("host LIKE ?", "%"+q.Host+"%")
	}
	if q.RecordType != "" {
		query = query.Where("record_type = ?", q.RecordType)
	}
	if q.Value != "" {
		query = query.Where("value LIKE ?", "%"+q.Value+"%")
	}
	if q.Enabled != nil {
		query = query.Where("enabled = ?", *q.Enabled)
	}
	if q.SyncStatus != "" {
		query = query.Where("sync_status = ?", q.SyncStatus)
	}

	var total int64
	query.Count(&total)

	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}

	var records []model.DNSRecord
	query.Order("id ASC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&records)

	return &RecordListResult{Items: records, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

func (s *RecordService) CreateRecord(nodeID, userID uint64, host, recordType, value string, ttl int, priority *int, line string) (*model.DNSRecord, error) {
	if !IsValidRecordType(recordType) {
		return nil, fmt.Errorf("unsupported record type: %s", recordType)
	}

	var node model.DomainNode
	if err := s.db.Where("id = ? AND owner_id = ?", nodeID, userID).First(&node).Error; err != nil {
		return nil, errors.New("domain node not found or access denied")
	}

	if err := validateRecordValue(recordType, value, priority); err != nil {
		return nil, err
	}

	if ttl == 0 {
		ttl = 600
	}
	if line == "" {
		line = "default"
	}

	record := &model.DNSRecord{
		NodeID:     nodeID,
		Host:       host,
		RecordType: recordType,
		Value:      value,
		TTL:        ttl,
		Priority:   priority,
		Line:       line,
		Enabled:    true,
		SyncStatus: "pending",
	}

	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}

	return record, nil
}

func (s *RecordService) UpdateRecord(recordID, userID uint64, value string, ttl *int, priority *int) (*model.DNSRecord, error) {
	var record model.DNSRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return nil, errors.New("record not found")
	}

	var node model.DomainNode
	if err := s.db.Where("id = ? AND owner_id = ?", record.NodeID, userID).First(&node).Error; err != nil {
		return nil, errors.New("access denied")
	}

	if value != "" {
		if err := validateRecordValue(record.RecordType, value, priority); err != nil {
			return nil, err
		}
	}

	updates := map[string]interface{}{
		"sync_status": "pending",
	}
	if value != "" {
		updates["value"] = value
	}
	if ttl != nil {
		updates["ttl"] = *ttl
	}
	if priority != nil {
		updates["priority"] = *priority
	}

	if err := s.db.Model(&record).Updates(updates).Error; err != nil {
		return nil, err
	}

	s.db.First(&record, recordID)
	return &record, nil
}

func (s *RecordService) DeleteRecord(recordID, userID uint64) error {
	var record model.DNSRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return errors.New("record not found")
	}

	var node model.DomainNode
	if err := s.db.Where("id = ? AND owner_id = ?", record.NodeID, userID).First(&node).Error; err != nil {
		return errors.New("access denied")
	}

	return s.db.Delete(&record).Error
}

func (s *RecordService) ToggleRecord(recordID, userID uint64, enabled bool) (*model.DNSRecord, error) {
	var record model.DNSRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return nil, errors.New("record not found")
	}

	var node model.DomainNode
	if err := s.db.Where("id = ? AND owner_id = ?", record.NodeID, userID).First(&node).Error; err != nil {
		return nil, errors.New("access denied")
	}

	syncStatus := "pending"
	if !enabled {
		syncStatus = "disabled"
	}

	updates := map[string]interface{}{
		"enabled":     enabled,
		"sync_status": syncStatus,
	}
	if err := s.db.Model(&record).Updates(updates).Error; err != nil {
		return nil, err
	}

	s.db.First(&record, recordID)
	return &record, nil
}

func (s *RecordService) BatchDelete(recordIDs []uint64, userID uint64) error {
	for _, id := range recordIDs {
		var record model.DNSRecord
		if err := s.db.First(&record, id).Error; err != nil {
			return fmt.Errorf("record %d not found", id)
		}
		var node model.DomainNode
		if err := s.db.Where("id = ? AND owner_id = ?", record.NodeID, userID).First(&node).Error; err != nil {
			return fmt.Errorf("access denied for record %d", id)
		}
	}

	return s.db.Delete(&model.DNSRecord{}, "id IN ?", recordIDs).Error
}

func (s *RecordService) BatchToggle(recordIDs []uint64, userID uint64, enabled bool) error {
	for _, id := range recordIDs {
		var record model.DNSRecord
		if err := s.db.First(&record, id).Error; err != nil {
			return fmt.Errorf("record %d not found", id)
		}
		var node model.DomainNode
		if err := s.db.Where("id = ? AND owner_id = ?", record.NodeID, userID).First(&node).Error; err != nil {
			return fmt.Errorf("access denied for record %d", id)
		}
	}

	syncStatus := "pending"
	if !enabled {
		syncStatus = "disabled"
	}

	return s.db.Model(&model.DNSRecord{}).Where("id IN ?", recordIDs).
		Updates(map[string]interface{}{"enabled": enabled, "sync_status": syncStatus}).Error
}

func (s *RecordService) GetRecordByID(recordID uint64) (*model.DNSRecord, error) {
	var record model.DNSRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *RecordService) UpdateSyncStatus(recordID uint64, status, aliyunRecordID string) error {
	updates := map[string]interface{}{
		"sync_status": status,
	}
	if aliyunRecordID != "" {
		updates["aliyun_record_id"] = aliyunRecordID
	}
	return s.db.Model(&model.DNSRecord{}).Where("id = ?", recordID).Updates(updates).Error
}

func (s *RecordService) FindRecordByNodeAndHost(nodeID uint64, host, recordType string) (*model.DNSRecord, error) {
	var record model.DNSRecord
	err := s.db.Where("node_id = ? AND host = ? AND record_type = ?", nodeID, host, recordType).
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func validateRecordValue(recordType, value string, priority *int) error {
	switch recordType {
	case "MX":
		if priority == nil {
			return errors.New("MX records require a priority value")
		}
	case "SRV":
		parts := strings.Fields(value)
		if len(parts) != 4 {
			return errors.New("SRV value must be 'priority weight port target'")
		}
		if _, err := strconv.Atoi(parts[0]); err != nil {
			return errors.New("SRV priority must be a number")
		}
		if _, err := strconv.Atoi(parts[1]); err != nil {
			return errors.New("SRV weight must be a number")
		}
		if _, err := strconv.Atoi(parts[2]); err != nil {
			return errors.New("SRV port must be a number")
		}
	case "CAA":
		parts := strings.SplitN(value, " ", 3)
		if len(parts) != 3 {
			return errors.New("CAA value must be 'flag tag value'")
		}
	}
	return nil
}
