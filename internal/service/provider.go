package service

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"domainnest/internal/dns"
	"domainnest/internal/model"

	"gorm.io/gorm"
)

type ProviderService struct {
	db *gorm.DB
}

func NewProviderService(db *gorm.DB) *ProviderService {
	return &ProviderService{db: db}
}

func (s *ProviderService) Create(userID uint64, providerType, name, ak, sk, endpoint string) (*model.DNSProvider, error) {
	// Verify credentials by creating a provider instance and listing domains
	provider, err := dns.Create(providerType, ak, sk, endpoint)
	if err != nil {
		return nil, fmt.Errorf("创建DNS客户端失败: %w", err)
	}
	if _, err := provider.ListDomains(); err != nil {
		return nil, fmt.Errorf("身份验证失败: %w", err)
	}
	dp := &model.DNSProvider{
		UserID:          userID,
		ProviderType:    providerType,
		Name:            name,
		AccessKeyID:     ak,
		AccessKeySecret: sk,
		Endpoint:        endpoint,
		Status:          "active",
	}
	if err := s.db.Create(dp).Error; err != nil {
		return nil, err
	}
	return dp, nil
}

func (s *ProviderService) List(userID uint64) ([]model.DNSProvider, error) {
	var providers []model.DNSProvider
	err := s.db.Where("user_id = ?", userID).Order("id DESC").Find(&providers).Error
	return providers, err
}

func (s *ProviderService) Get(providerID, userID uint64) (*model.DNSProvider, error) {
	var p model.DNSProvider
	if err := s.db.Where("id = ? AND user_id = ?", providerID, userID).First(&p).Error; err != nil {
		return nil, errors.New("DNS服务商不存在")
	}
	return &p, nil
}

func (s *ProviderService) Update(providerID, userID uint64, name, endpoint string) error {
	return s.db.Model(&model.DNSProvider{}).Where("id = ? AND user_id = ?", providerID, userID).
		Updates(map[string]interface{}{"name": name, "endpoint": endpoint}).Error
}

func (s *ProviderService) Delete(providerID, userID uint64, confirm bool) (int, error) {
	var count int64
	s.db.Model(&model.DomainNode{}).Where("provider_id = ?", providerID).Count(&count)

	if count > 0 && !confirm {
		return int(count), fmt.Errorf("此操作将归档 %d 个关联域名，请确认", count)
	}

	if count > 0 && confirm {
		err := s.db.Transaction(func(tx *gorm.DB) error {
			// Archive all linked nodes
			if err := tx.Model(&model.DomainNode{}).Where("provider_id = ?", providerID).
				Updates(map[string]interface{}{
					"status":               "archived",
					"archived_provider_id": providerID,
					"provider_id":          nil,
				}).Error; err != nil {
				return fmt.Errorf("归档关联域名失败: %w", err)
			}
			// Delete the provider
			return tx.Where("id = ? AND user_id = ?", providerID, userID).Delete(&model.DNSProvider{}).Error
		})
		return int(count), err
	}

	// count == 0: just delete
	return 0, s.db.Where("id = ? AND user_id = ?", providerID, userID).Delete(&model.DNSProvider{}).Error
}

func (s *ProviderService) ListDomains(providerID uint64) ([]dns.Domain, error) {
	p, err := s.GetDNSProvider(providerID)
	if err != nil {
		return nil, err
	}
	return p.ListDomains()
}

// DomainWithStatus extends the DNS domain info with claim status.
type DomainWithStatus struct {
	DomainName  string `json:"domain_name"`
	RecordCount int64  `json:"record_count"`
	Claimed     bool   `json:"claimed"`
	NodeID      uint64 `json:"node_id,omitempty"`
	OwnerName   string `json:"owner_name,omitempty"`
}

func (s *ProviderService) ListDomainsWithStatus(providerID uint64) ([]DomainWithStatus, error) {
	p, err := s.GetDNSProvider(providerID)
	if err != nil {
		return nil, err
	}
	domains, err := p.ListDomains()
	if err != nil {
		return nil, err
	}

	names := make([]string, len(domains))
	for i, d := range domains {
		names[i] = d.DomainName
	}

	var nodes []model.DomainNode
	s.db.Where("full_domain IN ? AND provider_id = ?", names, providerID).
		Preload("Owner", func(db *gorm.DB) *gorm.DB { return db.Select("id,username,nickname") }).
		Find(&nodes)

	nodeMap := make(map[string]model.DomainNode)
	for _, n := range nodes {
		nodeMap[n.FullDomain] = n
	}

	result := make([]DomainWithStatus, len(domains))
	for i, d := range domains {
		result[i] = DomainWithStatus{
			DomainName:  d.DomainName,
			RecordCount: d.RecordCount,
		}
		if node, ok := nodeMap[d.DomainName]; ok {
			result[i].Claimed = true
			result[i].NodeID = node.ID
			if node.Owner.Nickname != "" {
				result[i].OwnerName = node.Owner.Nickname
			} else {
				result[i].OwnerName = node.Owner.Username
			}
		}
	}
	return result, nil
}

func (s *ProviderService) ClaimDomain(userID, providerID uint64, domainName string) (*model.DomainNode, error) {
	var provider model.DNSProvider
	if err := s.db.Where("id = ? AND user_id = ?", providerID, userID).First(&provider).Error; err != nil {
		return nil, errors.New("DNS服务商不存在")
	}
	p, err := s.GetDNSProvider(providerID)
	if err != nil {
		return nil, err
	}
	// Verify domain access
	if _, err := p.ListRecords(domainName); err != nil {
		return nil, fmt.Errorf("无权访问域名 %s: %w", domainName, err)
	}
	var existing model.DomainNode
	if err := s.db.Where("full_domain = ?", domainName).First(&existing).Error; err == nil {
		return nil, errors.New("域名已存在于系统中")
	}
	host := extractHost(domainName)
	node := &model.DomainNode{
		Host:       host,
		FullDomain: domainName,
		OwnerID:    userID,
		ProviderID: &providerID,
	}
	// Hard-delete any soft-deleted row that still occupies the unique index
	var stale model.DomainNode
	if s.db.Unscoped().Where("full_domain = ?", domainName).First(&stale).Error == nil {
		s.db.Exec("DELETE FROM domain_nodes WHERE id = ?", stale.ID)
	}
	if err := s.db.Create(node).Error; err != nil {
		return nil, err
	}
	// Async import existing provider records
	go s.importProviderRecords(node.ID, providerID, domainName)
	return node, nil
}

func (s *ProviderService) importProviderRecords(nodeID, providerID uint64, domainName string) {
	p, err := s.GetDNSProvider(providerID)
	if err != nil {
		log.Printf("importRecords: GetDNSProvider failed for node %d: %v", nodeID, err)
		return
	}
	records, err := p.ListRecords(domainName)
	if err != nil {
		log.Printf("importRecords: ListRecords failed for node %d domain %s: %v", nodeID, domainName, err)
		return
	}
	if len(records) == 0 {
		s.db.Model(&model.DomainNode{}).Where("id = ?", nodeID).Update("records_imported", true)
		return
	}

	for _, r := range records {
		host := mapProviderRRToHost(r.Host, domainName)
		priority := convertPriority(r.Priority)
		record := &model.DNSRecord{
			NodeID:           nodeID,
			Host:             host,
			RecordType:       r.Type,
			Value:            r.Value,
			TTL:              int(r.TTL),
			Priority:         priority,
			Line:             r.Line,
			Enabled:          r.Enabled,
			ProviderRecordID: r.RecordID,
			SyncStatus:       "synced",
			Source:           "provider",
		}
		if err := s.db.Create(record).Error; err != nil {
			log.Printf("importRecords: failed to create record %s/%s for node %d: %v", host, r.Type, nodeID, err)
		}
	}

	s.db.Model(&model.DomainNode{}).Where("id = ?", nodeID).Update("records_imported", true)
}

// mapProviderRRToHost converts a provider's RR (full subdomain) to DomainNest's host part.
// For example, "test.example.com" with domain "example.com" becomes "test".
// The bare domain itself maps to "@".
func mapProviderRRToHost(rr, domainName string) string {
	if rr == domainName || rr == "@" {
		return "@"
	}
	suffix := "." + domainName
	if strings.HasSuffix(rr, suffix) {
		return strings.TrimSuffix(rr, suffix)
	}
	return rr
}

func convertPriority(p *int64) *int {
	if p == nil {
		return nil
	}
	v := int(*p)
	return &v
}

// GetDNSProvider creates a dns.Provider instance from the stored credentials.
func (s *ProviderService) GetDNSProvider(providerID uint64) (dns.Provider, error) {
	var provider model.DNSProvider
	if err := s.db.First(&provider, providerID).Error; err != nil {
		return nil, errors.New("DNS服务商不存在")
	}
	return dns.Create(provider.ProviderType, provider.AccessKeyID, provider.AccessKeySecret, provider.Endpoint)
}

func extractHost(domain string) string {
	parts := splitDomainParts(domain)
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return domain
}

func splitDomainParts(domain string) []string {
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
