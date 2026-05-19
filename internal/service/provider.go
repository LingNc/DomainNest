package service

import (
	"errors"
	"fmt"

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
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	if _, err := provider.ListDomains(); err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
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
		return nil, errors.New("provider not found")
	}
	return &p, nil
}

func (s *ProviderService) Update(providerID, userID uint64, name, endpoint string) error {
	return s.db.Model(&model.DNSProvider{}).Where("id = ? AND user_id = ?", providerID, userID).
		Updates(map[string]interface{}{"name": name, "endpoint": endpoint}).Error
}

func (s *ProviderService) Delete(providerID, userID uint64) error {
	var count int64
	s.db.Model(&model.DomainNode{}).Where("provider_id = ?", providerID).Count(&count)
	if count > 0 {
		return errors.New("cannot delete provider with associated domains")
	}
	return s.db.Where("id = ? AND user_id = ?", providerID, userID).Delete(&model.DNSProvider{}).Error
}

func (s *ProviderService) ListDomains(providerID uint64) ([]dns.Domain, error) {
	p, err := s.GetDNSProvider(providerID)
	if err != nil {
		return nil, err
	}
	return p.ListDomains()
}

func (s *ProviderService) ClaimDomain(userID, providerID uint64, domainName string) (*model.DomainNode, error) {
	var provider model.DNSProvider
	if err := s.db.Where("id = ? AND user_id = ?", providerID, userID).First(&provider).Error; err != nil {
		return nil, errors.New("provider not found")
	}
	p, err := s.GetDNSProvider(providerID)
	if err != nil {
		return nil, err
	}
	// Verify domain access
	if _, err := p.ListRecords(domainName); err != nil {
		return nil, fmt.Errorf("no access to domain %s: %w", domainName, err)
	}
	var existing model.DomainNode
	if err := s.db.Where("full_domain = ?", domainName).First(&existing).Error; err == nil {
		return nil, errors.New("domain already exists in system")
	}
	host := extractHost(domainName)
	node := &model.DomainNode{
		Host:       host,
		FullDomain: domainName,
		OwnerID:    userID,
		ProviderID: &providerID,
	}
	if err := s.db.Create(node).Error; err != nil {
		return nil, err
	}
	return node, nil
}

// GetDNSProvider creates a dns.Provider instance from the stored credentials.
func (s *ProviderService) GetDNSProvider(providerID uint64) (dns.Provider, error) {
	var provider model.DNSProvider
	if err := s.db.First(&provider, providerID).Error; err != nil {
		return nil, errors.New("provider not found")
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
