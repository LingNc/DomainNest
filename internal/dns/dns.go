package dns

import "fmt"

// Provider defines the interface that all DNS providers must implement.
type Provider interface {
	// ListDomains returns all domains managed by this provider.
	ListDomains() ([]Domain, error)

	// ListRecords returns all DNS records for a domain.
	ListRecords(domainName string) ([]Record, error)

	// AddRecord creates a new DNS record. Returns the provider-specific record ID.
	AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error)

	// UpdateRecord updates an existing DNS record by its provider-specific ID.
	UpdateRecord(domainName, recordID, rr, recordType, value string, ttl int64, priority *int64) error

	// DeleteRecord deletes a DNS record by its provider-specific ID.
	DeleteRecord(recordID string) error

	// GetType returns the provider type identifier (e.g. "aliyun", "cloudflare").
	GetType() string
}

// Domain represents a domain managed by a DNS provider.
type Domain struct {
	DomainName  string `json:"domain_name"`
	RecordCount int64  `json:"record_count"`
}

// Record represents a DNS record.
type Record struct {
	RecordID string `json:"record_id"`
	Host     string `json:"host"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	TTL      int64  `json:"ttl"`
	Priority *int64 `json:"priority,omitempty"`
	Line     string `json:"line,omitempty"`
	Enabled  bool   `json:"enabled,omitempty"`
}

// ProviderFactory creates a Provider from credentials.
type ProviderFactory func(accessKeyID, accessKeySecret, endpoint string) (Provider, error)

var registry = map[string]ProviderFactory{}

// Register registers a provider factory for a given type.
func Register(providerType string, factory ProviderFactory) {
	registry[providerType] = factory
}

// Create creates a provider instance from the given type and credentials.
func Create(providerType, accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
	factory, ok := registry[providerType]
	if !ok {
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
	return factory(accessKeyID, accessKeySecret, endpoint)
}

// SupportedTypes returns all registered provider types.
func SupportedTypes() []string {
	types := make([]string, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	return types
}
