package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"domainnest/internal/model"

	"gorm.io/gorm"
)

type RAMTokenService struct {
	db *gorm.DB
}

func NewRAMTokenService(db *gorm.DB) *RAMTokenService {
	return &RAMTokenService{db: db}
}

func (s *RAMTokenService) Create(userID uint64, name string, allowedDomains []uint64, allowedTypes, allowedIPs []string) (*model.RAMToken, error) {
	if name == "" {
		return nil, errors.New("token name is required")
	}

	token, err := generateRAMToken()
	if err != nil {
		return nil, err
	}

	domainsJSON := ""
	if len(allowedDomains) > 0 {
		b, _ := json.Marshal(allowedDomains)
		domainsJSON = string(b)
	}
	typesJSON := ""
	if len(allowedTypes) > 0 {
		b, _ := json.Marshal(allowedTypes)
		typesJSON = string(b)
	}
	ipsJSON := ""
	if len(allowedIPs) > 0 {
		b, _ := json.Marshal(allowedIPs)
		ipsJSON = string(b)
	}

	ramToken := &model.RAMToken{
		UserID:         userID,
		Name:           name,
		Token:          token,
		Enabled:        true,
		AllowedDomains: domainsJSON,
		AllowedTypes:   typesJSON,
		AllowedIPs:     ipsJSON,
	}

	if err := s.db.Create(ramToken).Error; err != nil {
		return nil, err
	}
	return ramToken, nil
}

func (s *RAMTokenService) List(userID uint64) ([]model.RAMToken, error) {
	var tokens []model.RAMToken
	err := s.db.Where("user_id = ?", userID).Order("id ASC").Find(&tokens).Error
	return tokens, err
}

func (s *RAMTokenService) Get(tokenID, userID uint64) (*model.RAMToken, error) {
	var token model.RAMToken
	if err := s.db.Where("id = ? AND user_id = ?", tokenID, userID).First(&token).Error; err != nil {
		return nil, errors.New("token not found")
	}
	return &token, nil
}

// GetByID retrieves a RAM token by ID without user filtering (for middleware/internal use).
func (s *RAMTokenService) GetByID(tokenID uint64) (*model.RAMToken, error) {
	var token model.RAMToken
	if err := s.db.First(&token, tokenID).Error; err != nil {
		return nil, errors.New("token not found")
	}
	return &token, nil
}

func (s *RAMTokenService) Update(tokenID, userID uint64, name string, enabled *bool, allowedDomains []uint64, allowedTypes, allowedIPs []string) (*model.RAMToken, error) {
	var token model.RAMToken
	if err := s.db.Where("id = ? AND user_id = ?", tokenID, userID).First(&token).Error; err != nil {
		return nil, errors.New("token not found")
	}

	updates := map[string]interface{}{}
	if name != "" {
		updates["name"] = name
	}
	if enabled != nil {
		updates["enabled"] = *enabled
	}
	if allowedDomains != nil {
		b, _ := json.Marshal(allowedDomains)
		updates["allowed_domains"] = string(b)
	}
	if allowedTypes != nil {
		b, _ := json.Marshal(allowedTypes)
		updates["allowed_types"] = string(b)
	}
	if allowedIPs != nil {
		b, _ := json.Marshal(allowedIPs)
		updates["allowed_ips"] = string(b)
	}

	if len(updates) > 0 {
		if err := s.db.Model(&token).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	s.db.First(&token, token.ID)
	return &token, nil
}

func (s *RAMTokenService) ResetToken(tokenID, userID uint64) (*model.RAMToken, error) {
	var token model.RAMToken
	if err := s.db.Where("id = ? AND user_id = ?", tokenID, userID).First(&token).Error; err != nil {
		return nil, errors.New("token not found")
	}

	newToken, err := generateRAMToken()
	if err != nil {
		return nil, err
	}

	if err := s.db.Model(&token).Update("token", newToken).Error; err != nil {
		return nil, err
	}

	s.db.First(&token, token.ID)
	return &token, nil
}

func (s *RAMTokenService) Delete(tokenID, userID uint64) error {
	var token model.RAMToken
	if err := s.db.Where("id = ? AND user_id = ?", tokenID, userID).First(&token).Error; err != nil {
		return errors.New("token not found")
	}
	return s.db.Delete(&token).Error
}

// ValidateAndLookup checks a raw token string and returns the RAMToken + owner user ID.
func (s *RAMTokenService) ValidateAndLookup(tokenStr string) (*model.RAMToken, error) {
	var token model.RAMToken
	if err := s.db.Where("token = ? AND enabled = ?", tokenStr, true).First(&token).Error; err != nil {
		return nil, errors.New("invalid or disabled RAM token")
	}

	// Update usage stats
	now := time.Now()
	s.db.Model(&token).Updates(map[string]interface{}{
		"usage_count":  gorm.Expr("usage_count + 1"),
		"last_used_at": now,
	})

	return &token, nil
}

// CheckDomainAccess verifies the RAM token can access the given domain node.
func (s *RAMTokenService) CheckDomainAccess(token *model.RAMToken, domainNodeID uint64) error {
	if token.AllowedDomains == "" || token.AllowedDomains == "[]" {
		return nil
	}

	var allowedIDs []uint64
	if err := json.Unmarshal([]byte(token.AllowedDomains), &allowedIDs); err != nil {
		return nil
	}

	for _, id := range allowedIDs {
		if id == domainNodeID {
			return nil
		}
	}

	return fmt.Errorf("RAM token does not have access to domain %d", domainNodeID)
}

// CheckRecordType verifies the RAM token can use the given record type.
func (s *RAMTokenService) CheckRecordType(token *model.RAMToken, recordType string) error {
	if token.AllowedTypes == "" || token.AllowedTypes == "[]" {
		return nil
	}

	var types []string
	if err := json.Unmarshal([]byte(token.AllowedTypes), &types); err != nil {
		return nil
	}

	for _, t := range types {
		if t == recordType {
			return nil
		}
	}

	return fmt.Errorf("RAM token does not allow record type %s", recordType)
}

// ValidateIP checks if the IP value is within the token's allowed CIDRs.
func (s *RAMTokenService) ValidateIP(token *model.RAMToken, value string) error {
	if token.AllowedIPs == "" || token.AllowedIPs == "[]" {
		return nil
	}

	var cidrs []string
	if err := json.Unmarshal([]byte(token.AllowedIPs), &cidrs); err != nil {
		return nil
	}

	ip, err := netip.ParseAddr(value)
	if err != nil {
		return nil // not an IP value, skip check
	}

	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			continue
		}
		if prefix.Contains(ip) {
			return nil
		}
	}

	return fmt.Errorf("IP %s is not within allowed ranges", value)
}

func generateRAMToken() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "rn_" + hex.EncodeToString(bytes), nil
}
