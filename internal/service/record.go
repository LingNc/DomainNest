package service

import (
	"errors"

	"domainnest/internal/model"

	"gorm.io/gorm"
)

type RecordService struct {
	db *gorm.DB
}

func NewRecordService(db *gorm.DB) *RecordService {
	return &RecordService{db: db}
}

func (s *RecordService) GetRecords(nodeID, userID uint64) ([]model.DNSRecord, error) {
	var node model.DomainNode
	if err := s.db.Where("id = ? AND owner_id = ?", nodeID, userID).First(&node).Error; err != nil {
		return nil, errors.New("domain node not found or access denied")
	}

	var records []model.DNSRecord
	err := s.db.Where("node_id = ?", nodeID).Find(&records).Error
	return records, err
}

func (s *RecordService) CreateRecord(nodeID, userID uint64, host, recordType, value string, ttl int, priority *int, line string) (*model.DNSRecord, error) {
	var node model.DomainNode
	if err := s.db.Where("id = ? AND owner_id = ?", nodeID, userID).First(&node).Error; err != nil {
		return nil, errors.New("domain node not found or access denied")
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
