package service

import (
	"errors"
	"fmt"

	"domainnest/internal/aliyun"
	"domainnest/internal/model"

	"gorm.io/gorm"
)

type DDNSService struct {
	db            *gorm.DB
	domainService *DomainService
	recordService *RecordService
	aliyunClient  *aliyun.Client
}

func NewDDNSService(db *gorm.DB, domainService *DomainService, recordService *RecordService, aliyunClient *aliyun.Client) *DDNSService {
	return &DDNSService{
		db:            db,
		domainService: domainService,
		recordService: recordService,
		aliyunClient:  aliyunClient,
	}
}

type DDNSUpdateResult struct {
	Domain     string `json:"domain"`
	IP         string `json:"ip"`
	RecordType string `json:"record_type"`
	Action     string `json:"action"`
}

func (s *DDNSService) Update(userID uint64, domain, ip, recordType string, ttl int) (*DDNSUpdateResult, error) {
	if recordType == "" {
		recordType = "A"
	}
	if ttl == 0 {
		ttl = 600
	}

	node, rr, err := s.domainService.FindNodeByDomain(domain, userID)
	if err != nil {
		return nil, err
	}

	rootDomain := getRootDomain(node.FullDomain)
	rrForAliyun := getRRForAliyun(node.FullDomain, rootDomain, rr)

	record, err := s.recordService.FindRecordByNodeAndHost(node.ID, rr, recordType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.createRecord(node.ID, rootDomain, rrForAliyun, rr, recordType, ip, ttl)
		}
		return nil, err
	}

	if record.Value == ip && record.SyncStatus == "synced" {
		return &DDNSUpdateResult{
			Domain:     domain,
			IP:         ip,
			RecordType: recordType,
			Action:     "updated",
		}, nil
	}

	return s.updateRecord(record, rootDomain, rrForAliyun, ip, ttl)
}

func (s *DDNSService) createRecord(nodeID uint64, rootDomain, rrForAliyun, host, recordType, ip string, ttl int) (*DDNSUpdateResult, error) {
	record := &model.DNSRecord{
		NodeID:     nodeID,
		Host:       host,
		RecordType: recordType,
		Value:      ip,
		TTL:        ttl,
		SyncStatus: "pending",
	}
	if err := s.db.Create(record).Error; err != nil {
		return nil, fmt.Errorf("failed to create local record: %w", err)
	}

	aliyunRecordID, err := s.aliyunClient.AddRecord(rootDomain, rrForAliyun, recordType, ip, int64(ttl), nil)
	if err != nil {
		s.recordService.UpdateSyncStatus(record.ID, "failed", "")
		return nil, fmt.Errorf("failed to sync to aliyun: %w", err)
	}

	s.recordService.UpdateSyncStatus(record.ID, "synced", aliyunRecordID)

	return &DDNSUpdateResult{
		Domain:     rootDomain,
		IP:         ip,
		RecordType: recordType,
		Action:     "created",
	}, nil
}

func (s *DDNSService) updateRecord(record *model.DNSRecord, rootDomain, rrForAliyun, ip string, ttl int) (*DDNSUpdateResult, error) {
	updates := map[string]interface{}{
		"value":       ip,
		"ttl":         ttl,
		"sync_status": "pending",
	}
	if err := s.db.Model(record).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update local record: %w", err)
	}

	var priority *int64
	if record.Priority != nil {
		p := int64(*record.Priority)
		priority = &p
	}

	if record.AliyunRecordID != "" {
		err := s.aliyunClient.UpdateRecord(record.AliyunRecordID, rrForAliyun, record.RecordType, ip, int64(ttl), priority)
		if err != nil {
			s.recordService.UpdateSyncStatus(record.ID, "failed", record.AliyunRecordID)
			return nil, fmt.Errorf("failed to sync to aliyun: %w", err)
		}
		s.recordService.UpdateSyncStatus(record.ID, "synced", record.AliyunRecordID)
	} else {
		aliyunRecordID, err := s.aliyunClient.AddRecord(rootDomain, rrForAliyun, record.RecordType, ip, int64(ttl), priority)
		if err != nil {
			s.recordService.UpdateSyncStatus(record.ID, "failed", "")
			return nil, fmt.Errorf("failed to sync to aliyun: %w", err)
		}
		s.recordService.UpdateSyncStatus(record.ID, "synced", aliyunRecordID)
	}

	return &DDNSUpdateResult{
		Domain:     rootDomain,
		IP:         ip,
		RecordType: record.RecordType,
		Action:     "updated",
	}, nil
}

func (s *DDNSService) SyncRecord(recordID uint64) error {
	record, err := s.recordService.GetRecordByID(recordID)
	if err != nil {
		return err
	}

	if !record.Enabled {
		if record.AliyunRecordID != "" {
			if err := s.aliyunClient.DeleteRecord(record.AliyunRecordID); err != nil {
				s.recordService.UpdateSyncStatus(record.ID, "failed", record.AliyunRecordID)
				return err
			}
			s.recordService.UpdateSyncStatus(record.ID, "disabled", "")
		}
		return nil
	}

	var node model.DomainNode
	if err := s.db.First(&node, record.NodeID).Error; err != nil {
		return err
	}

	rootDomain := getRootDomain(node.FullDomain)
	rrForAliyun := getRRForAliyun(node.FullDomain, rootDomain, record.Host)

	var priority *int64
	if record.Priority != nil {
		p := int64(*record.Priority)
		priority = &p
	}

	if record.AliyunRecordID != "" {
		err := s.aliyunClient.UpdateRecord(record.AliyunRecordID, rrForAliyun, record.RecordType, record.Value, int64(record.TTL), priority)
		if err != nil {
			s.recordService.UpdateSyncStatus(record.ID, "failed", record.AliyunRecordID)
			return err
		}
		s.recordService.UpdateSyncStatus(record.ID, "synced", record.AliyunRecordID)
	} else {
		aliyunRecordID, err := s.aliyunClient.AddRecord(rootDomain, rrForAliyun, record.RecordType, record.Value, int64(record.TTL), priority)
		if err != nil {
			s.recordService.UpdateSyncStatus(record.ID, "failed", "")
			return err
		}
		s.recordService.UpdateSyncStatus(record.ID, "synced", aliyunRecordID)
	}

	return nil
}

func getRootDomain(fullDomain string) string {
	parts := splitDomain(fullDomain)
	if len(parts) < 2 {
		return fullDomain
	}
	return parts[len(parts)-2] + "." + parts[len(parts)-1]
}

func getRRForAliyun(fullDomain, rootDomain, host string) string {
	subDomain := fullDomain
	if fullDomain != rootDomain {
		subDomain = fullDomain[:len(fullDomain)-len(rootDomain)-1]
	}

	if host == "@" {
		if subDomain == "" {
			return "@"
		}
		return subDomain
	}

	if subDomain == "" {
		return host
	}
	return host + "." + subDomain
}

func splitDomain(domain string) []string {
	var parts []string
	current := ""
	for _, c := range domain {
		if c == '.' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
