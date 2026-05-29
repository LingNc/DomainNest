package service

import (
	"strconv"

	"domainnest/internal/errs"
	"domainnest/internal/model"

	"gorm.io/gorm"
)

type AliyunCompatService struct {
	db        *gorm.DB
	domainSvc *DomainService
	recordSvc *RecordService
	permSvc   *PermissionService
}

func NewAliyunCompatService(db *gorm.DB, domainSvc *DomainService, recordSvc *RecordService, permSvc *PermissionService) *AliyunCompatService {
	return &AliyunCompatService{db: db, domainSvc: domainSvc, recordSvc: recordSvc, permSvc: permSvc}
}

// DescribeDomains lists all root domains accessible to the user.
func (s *AliyunCompatService) DescribeDomains(userID uint64) ([]map[string]interface{}, error) {
	nodes, err := s.domainSvc.GetUserNodes(userID)
	if err != nil {
		return nil, err
	}
	var domains []map[string]interface{}
	for _, n := range nodes {
		if n.ParentID == nil {
			cnt, _ := s.recordSvc.CountAccessibleRecords(n.ID, userID)
			domains = append(domains, map[string]interface{}{
				"DomainName":      n.FullDomain,
				"RecordCount":     cnt,
				"RegistrantEmail": "",
				"AliDomain":        false,
			})
		}
	}
	return domains, nil
}

// DescribeDomainRecords lists DNS records for a domain with optional filters.
func (s *AliyunCompatService) DescribeDomainRecords(userID uint64, domainName string, rrKeyword, typeKeyword, valueKeyword string, pageNumber, pageSize int) (*RecordListResult, uint64, error) {
	node, _, err := s.domainSvc.FindNodeByDomain(domainName, userID)
	if err != nil {
		return nil, 0, errs.New(errs.DomainNotFoundOrNoAccess, "域名不存在或无访问权限")
	}

	if err := s.permSvc.RequireLevel(userID, node.ID, 1); err != nil {
		return nil, 0, err
	}

	if pageNumber < 1 {
		pageNumber = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	q := RecordQuery{
		Host:       rrKeyword,
		RecordType: typeKeyword,
		Value:      valueKeyword,
		Page:       pageNumber,
		PageSize:   pageSize,
	}

	result, err := s.recordSvc.ListRecords(node.ID, userID, q)
	if err != nil {
		return nil, 0, err
	}
	return result, node.ID, nil
}

// AddDomainRecord creates a new DNS record.
func (s *AliyunCompatService) AddDomainRecord(userID uint64, domainName, rr, recordType, value string, ttl int, priority *int, line string) (*model.DNSRecord, error) {
	fqdn := rr
	if rr == "@" || rr == "" {
		fqdn = domainName
	} else {
		fqdn = rr + "." + domainName
	}

	node, _, err := s.domainSvc.FindNodeByDomain(fqdn, userID)
	if err != nil {
		return nil, errs.New(errs.DomainNotFoundOrNoAccess, "域名不存在或无访问权限")
	}

	host := rr
	if rr == "@" || rr == "" {
		host = "@"
	}

	return s.recordSvc.CreateRecord(node.ID, userID, host, recordType, value, ttl, priority, line)
}

// UpdateDomainRecord updates an existing DNS record.
func (s *AliyunCompatService) UpdateDomainRecord(userID uint64, recordIDStr, rr, recordType, value string, ttl int, priority *int) (*model.DNSRecord, error) {
	recordID, err := strconv.ParseUint(recordIDStr, 10, 64)
	if err != nil {
		return nil, errs.New(errs.InvalidRecordID, "无效的记录ID")
	}

	record, err := s.recordSvc.GetRecordByID(recordID)
	if err != nil {
		return nil, errs.New(errs.RecordNotFound, "记录不存在")
	}

	if err := s.permSvc.RequireLevel(userID, record.NodeID, 2); err != nil {
		return nil, err
	}

	host := rr
	if rr != "" {
		host = rr
	}

	return s.recordSvc.UpdateRecord(recordID, userID, value, &ttl, priority, host)
}

// DeleteDomainRecord deletes a DNS record by ID string.
func (s *AliyunCompatService) DeleteDomainRecord(userID uint64, recordIDStr string) error {
	recordID, err := strconv.ParseUint(recordIDStr, 10, 64)
	if err != nil {
		return errs.New(errs.InvalidRecordID, "无效的记录ID")
	}

	return s.recordSvc.DeleteRecord(recordID, userID)
}

// ResolveDomain resolves a domain name to nodeID, similar to FindNodeByDomain but returns just the node.
func (s *AliyunCompatService) ResolveDomain(domain string, userID uint64) (*model.DomainNode, uint64, error) {
	node, _, err := s.domainSvc.FindNodeByDomain(domain, userID)
	if err != nil {
		return nil, 0, err
	}
	return node, node.ID, nil
}

// GetRecord retrieves a record by ID.
func (s *AliyunCompatService) GetRecord(recordID uint64) (*model.DNSRecord, error) {
	return s.recordSvc.GetRecordByID(recordID)
}