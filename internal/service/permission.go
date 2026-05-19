package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"

	"domainnest/internal/model"

	"gorm.io/gorm"
)

var permLevels = map[string]int{
	"none":  0,
	"read":  1,
	"write": 2,
	"admin": 3,
	"owner": 4,
}

func PermLevelValue(level string) int {
	if v, ok := permLevels[level]; ok {
		return v
	}
	return 0
}

type PermissionService struct {
	db *gorm.DB
}

func NewPermissionService(db *gorm.DB) *PermissionService {
	return &PermissionService{db: db}
}

// AccessLevel returns the effective permission level for a user on a domain node.
// super_admin gets level 5, owner gets level 4, then checks DomainPermission table.
func (s *PermissionService) AccessLevel(userID, domainNodeID uint64) (int, string) {
	// Check super_admin
	var user model.User
	if err := s.db.First(&user, userID).Error; err == nil && user.IsSuperAdmin {
		return 5, "super_admin"
	}

	// Check owner
	var node model.DomainNode
	if err := s.db.First(&node, domainNodeID).Error; err == nil && node.OwnerID == userID {
		return 4, "owner"
	}

	// Check permission table
	var perm model.DomainPermission
	if err := s.db.Where("user_id = ? AND domain_node_id = ?", userID, domainNodeID).First(&perm).Error; err == nil {
		return PermLevelValue(perm.PermissionLevel), perm.PermissionLevel
	}

	return 0, "none"
}

// RequireLevel returns an error if the user's access level is below the minimum.
func (s *PermissionService) RequireLevel(userID, domainNodeID uint64, minLevel int) error {
	level, name := s.AccessLevel(userID, domainNodeID)
	if level < minLevel {
		return fmt.Errorf("access denied: requires %s or higher, has %s", levelName(minLevel), name)
	}
	return nil
}

// CanUseRecordType checks if the user is allowed to use the given record type on the domain.
func (s *PermissionService) CanUseRecordType(userID, domainNodeID uint64, recordType string) bool {
	level, _ := s.AccessLevel(userID, domainNodeID)
	if level >= 4 { // owner or super_admin
		return true
	}

	var perm model.DomainPermission
	if err := s.db.Where("user_id = ? AND domain_node_id = ?", userID, domainNodeID).First(&perm).Error; err != nil {
		return false
	}

	if perm.AllowedTypes == "" || perm.AllowedTypes == "[]" {
		return true
	}

	var types []string
	if err := json.Unmarshal([]byte(perm.AllowedTypes), &types); err != nil {
		return true
	}

	for _, t := range types {
		if t == recordType {
			return true
		}
	}
	return false
}

// ValidateIPValue checks if the record value (for A/AAAA records) is within allowed CIDRs.
func (s *PermissionService) ValidateIPValue(userID, domainNodeID uint64, recordType, value string) error {
	if recordType != "A" && recordType != "AAAA" {
		return nil
	}

	level, _ := s.AccessLevel(userID, domainNodeID)
	if level >= 4 {
		return nil
	}

	var perm model.DomainPermission
	if err := s.db.Where("user_id = ? AND domain_node_id = ?", userID, domainNodeID).First(&perm).Error; err != nil {
		return nil
	}

	if perm.AllowedIPs == "" || perm.AllowedIPs == "[]" {
		return nil
	}

	var cidrs []string
	if err := json.Unmarshal([]byte(perm.AllowedIPs), &cidrs); err != nil {
		return nil
	}

	ip, err := netip.ParseAddr(value)
	if err != nil {
		return fmt.Errorf("invalid IP address: %s", value)
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

// ValidateHostPrefix checks if the given host matches the required prefix restriction.
// If prefix is empty, all hosts are allowed.
// e.g. prefix="test-" allows "test-app", "test-web" but not "prod-app".
func (s *PermissionService) ValidateHostPrefix(userID, domainNodeID uint64, host string) error {
	level, _ := s.AccessLevel(userID, domainNodeID)
	if level >= 4 { // owner or super_admin
		return nil
	}

	var perm model.DomainPermission
	if err := s.db.Where("user_id = ? AND domain_node_id = ?", userID, domainNodeID).First(&perm).Error; err != nil {
		return nil
	}

	if perm.HostPrefix == "" {
		return nil
	}

	if !strings.HasPrefix(host, perm.HostPrefix) {
		return fmt.Errorf("host must start with '%s'", perm.HostPrefix)
	}
	return nil
}

// ValidateDepth checks if the subdomain depth is within the allowed limit.
// maxDepth=nil means unlimited. Depth is measured from the domain node.
// e.g. domain "example.com", host "a.b.c" has depth 3.
func (s *PermissionService) ValidateDepth(userID, domainNodeID uint64, host string) error {
	level, _ := s.AccessLevel(userID, domainNodeID)
	if level >= 4 {
		return nil
	}

	var perm model.DomainPermission
	if err := s.db.Where("user_id = ? AND domain_node_id = ?", userID, domainNodeID).First(&perm).Error; err != nil {
		return nil
	}

	if perm.MaxDepth == nil {
		return nil
	}

	// "@" means the domain itself, depth 1
	depth := 1
	if host != "@" {
		depth = len(strings.Split(host, "."))
	}

	if depth > *perm.MaxDepth {
		return fmt.Errorf("subdomain depth %d exceeds maximum allowed %d", depth, *perm.MaxDepth)
	}
	return nil
}

// Grant creates or updates a permission entry.
func (s *PermissionService) Grant(userID, domainNodeID uint64, level, allowedTypes, allowedIPs, hostPrefix string, maxDepth *int, createdBy uint64) error {
	if PermLevelValue(level) == 0 && level != "read" {
		return fmt.Errorf("invalid permission level: %s", level)
	}

	// Cannot grant higher than your own level
	grantorLevel, _ := s.AccessLevel(createdBy, domainNodeID)
	if PermLevelValue(level) >= grantorLevel {
		return errors.New("cannot grant permission equal to or higher than your own")
	}

	var existing model.DomainPermission
	err := s.db.Where("user_id = ? AND domain_node_id = ?", userID, domainNodeID).First(&existing).Error
	if err == nil {
		return s.db.Model(&existing).Updates(map[string]interface{}{
			"permission_level": level,
			"allowed_types":    allowedTypes,
			"allowed_ips":      allowedIPs,
			"host_prefix":      hostPrefix,
			"max_depth":        maxDepth,
		}).Error
	}

	perm := &model.DomainPermission{
		UserID:          userID,
		DomainNodeID:    domainNodeID,
		PermissionLevel: level,
		AllowedTypes:    allowedTypes,
		AllowedIPs:      allowedIPs,
		HostPrefix:      hostPrefix,
		MaxDepth:        maxDepth,
		CreatedBy:       createdBy,
	}
	return s.db.Create(perm).Error
}

// Revoke removes a permission entry.
func (s *PermissionService) Revoke(userID, domainNodeID uint64) error {
	return s.db.Where("user_id = ? AND domain_node_id = ?", userID, domainNodeID).
		Delete(&model.DomainPermission{}).Error
}

// ListPermissions returns all permissions for a domain node.
func (s *PermissionService) ListPermissions(domainNodeID uint64) ([]model.DomainPermission, error) {
	var perms []model.DomainPermission
	err := s.db.Preload("User").Where("domain_node_id = ?", domainNodeID).Find(&perms).Error
	return perms, err
}

// GetUserPermissions returns all domain permissions for a user.
func (s *PermissionService) GetUserPermissions(userID uint64) ([]model.DomainPermission, error) {
	var perms []model.DomainPermission
	err := s.db.Preload("DomainNode").Preload("Creator").Where("user_id = ?", userID).Find(&perms).Error
	return perms, err
}

// OwnedDomainIDs returns all domain IDs owned by the user.
func (s *PermissionService) OwnedDomainIDs(userID uint64) ([]uint64, error) {
	var nodes []model.DomainNode
	if err := s.db.Where("owner_id = ?", userID).Select("id").Find(&nodes).Error; err != nil {
		return nil, err
	}
	ids := make([]uint64, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	return ids, nil
}

// AccessibleDomainIDs returns all domain IDs the user can access (owned + delegated).
func (s *PermissionService) AccessibleDomainIDs(userID uint64) ([]uint64, error) {
	owned, err := s.OwnedDomainIDs(userID)
	if err != nil {
		return nil, err
	}

	var perms []model.DomainPermission
	if err := s.db.Where("user_id = ?", userID).Select("domain_node_id").Find(&perms).Error; err != nil {
		return nil, err
	}

	seen := make(map[uint64]bool)
	for _, id := range owned {
		seen[id] = true
	}
	for _, p := range perms {
		seen[p.DomainNodeID] = true
	}

	ids := make([]uint64, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids, nil
}

func levelName(level int) string {
	switch level {
	case 1:
		return "read"
	case 2:
		return "write"
	case 3:
		return "admin"
	case 4:
		return "owner"
	case 5:
		return "super_admin"
	default:
		return "none"
	}
}
