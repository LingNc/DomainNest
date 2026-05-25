package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// godaddyAPIRecord represents a GoDaddy DNS record for JSON serialization.
type godaddyAPIRecord struct {
	Data     string `json:"data"`
	Name     string `json:"name"`
	TTL      int64  `json:"ttl"`
	Type     string `json:"type"`
	Priority *int64 `json:"priority,omitempty"`
}

// GoDaddyProvider implements the Provider interface for GoDaddy DNS.
type GoDaddyProvider struct {
	apiKey    string
	apiSecret string
	client    *http.Client
}

func init() {
	Register("godaddy", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &GoDaddyProvider{
			apiKey:    accessKeyID,
			apiSecret: accessKeySecret,
			client:    &http.Client{},
		}, nil
	})
}

func (p *GoDaddyProvider) GetType() string { return "godaddy" }

func (p *GoDaddyProvider) authHeader() http.Header {
	return http.Header{
		"Authorization": {fmt.Sprintf("sso-key %s:%s", p.apiKey, p.apiSecret)},
		"Content-Type":  {"application/json"},
	}
}

// ListDomains returns all domains from GoDaddy.
func (p *GoDaddyProvider) ListDomains() ([]Domain, error) {
	type gdDomain struct {
		Domain     string `json:"domain"`
		NumRecords int64  `json:"numRecords"`
	}

	var all []gdDomain
	url := "https://api.godaddy.com/v1/domains?limit=100"
	for url != "" {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header = p.authHeader()

		resp, err := p.client.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("godaddy ListDomains: %d %s", resp.StatusCode, string(body))
		}

		var page []gdDomain
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}
		all = append(all, page...)

		// Parse Link header for pagination.
		url = ""
		for _, link := range resp.Header.Values("Link") {
			if strings.Contains(link, `rel="next"`) {
				if start := strings.Index(link, "<"); start >= 0 {
					if end := strings.Index(link[start:], ">"); end > 0 {
						url = link[start+1 : start+end]
					}
				}
			}
		}
	}

	domains := make([]Domain, len(all))
	for i, d := range all {
		domains[i] = Domain{DomainName: d.Domain, RecordCount: d.NumRecords}
	}
	return domains, nil
}

// ListRecords returns all DNS records for a domain from GoDaddy.
// GoDaddy requires querying each record type separately.
func (p *GoDaddyProvider) ListRecords(domainName string) ([]Record, error) {
	recordTypes := []string{"A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV", "SOA", "PTR"}
	var records []Record

	for _, rType := range recordTypes {
		url := fmt.Sprintf("https://api.godaddy.com/v1/domains/%s/records/%s", domainName, rType)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header = p.authHeader()

		resp, err := p.client.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 400 {
			if resp.StatusCode == 404 || resp.StatusCode == 422 {
				continue
			}
			return nil, fmt.Errorf("godaddy ListRecords(%s): %d %s", rType, resp.StatusCode, string(body))
		}

		var apiRecs []godaddyAPIRecord
		if err := json.Unmarshal(body, &apiRecs); err != nil {
			return nil, err
		}

		for _, r := range apiRecs {
			records = append(records, Record{
				RecordID: godaddyRecordID(domainName, r.Type, r.Name),
				Host:     r.Name,
				Type:     r.Type,
				Value:    r.Data,
				TTL:      r.TTL,
				Priority: r.Priority,
			})
		}
	}
	return records, nil
}

// AddRecord creates a new DNS record in GoDaddy.
// GoDaddy PUT replaces all records of a given type+name, so we must
// fetch existing records first and append the new one.
func (p *GoDaddyProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	existing, err := p.fetchRecordsByNameType(domainName, recordType, rr)
	if err != nil {
		return "", err
	}

	newRec := godaddyAPIRecord{
		Data:     value,
		Name:     rr,
		TTL:      ttl,
		Type:     recordType,
		Priority: priority,
	}
	existing = append(existing, newRec)

	if err := p.putRecords(domainName, recordType, rr, existing); err != nil {
		return "", err
	}
	return godaddyRecordID(domainName, recordType, rr), nil
}

// UpdateRecord updates an existing DNS record in GoDaddy.
// recordID format: "domain|TYPE/NAME"
func (p *GoDaddyProvider) UpdateRecord(domainName, recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	domain, origType, origName, err := parseGodaddyRecordID(recordID)
	if err != nil {
		return err
	}

	// Fetch all records of this type+name, find and replace the matching one.
	existing, err := p.fetchRecordsByNameType(domain, origType, origName)
	if err != nil {
		return err
	}

	updated := false
	for i, r := range existing {
		if r.Type == origType && r.Name == origName {
			existing[i] = godaddyAPIRecord{
				Data:     value,
				Name:     rr,
				TTL:      ttl,
				Type:     recordType,
				Priority: priority,
			}
			updated = true
			break
		}
	}
	if !updated {
		// If not found, just set it.
		existing = append(existing, godaddyAPIRecord{
			Data:     value,
			Name:     rr,
			TTL:      ttl,
			Type:     recordType,
			Priority: priority,
		})
	}

	// If the name or type changed, we need to delete from old and add to new.
	if origType != recordType || origName != rr {
		// Remove from old location.
		if err := p.deleteByNameType(domain, origType, origName); err != nil {
			return err
		}
		// Add to new location (fetch new location's existing records first).
		newExisting, err := p.fetchRecordsByNameType(domain, recordType, rr)
		if err != nil {
			return err
		}
		newExisting = append(newExisting, godaddyAPIRecord{
			Data:     value,
			Name:     rr,
			TTL:      ttl,
			Type:     recordType,
			Priority: priority,
		})
		return p.putRecords(domain, recordType, rr, newExisting)
	}

	return p.putRecords(domain, recordType, rr, existing)
}

// DeleteRecord deletes a DNS record from GoDaddy.
// recordID format: "domain|TYPE/NAME"
func (p *GoDaddyProvider) DeleteRecord(recordID string) error {
	domain, recordType, name, err := parseGodaddyRecordID(recordID)
	if err != nil {
		return err
	}
	return p.deleteByNameType(domain, recordType, name)
}

// --- internal helpers ---

func (p *GoDaddyProvider) fetchRecordsByNameType(domainName, recordType, name string) ([]godaddyAPIRecord, error) {
	url := fmt.Sprintf("https://api.godaddy.com/v1/domains/%s/records/%s/%s", domainName, recordType, name)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header = p.authHeader()

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 404 {
		return nil, nil
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("godaddy fetchRecords: %d %s", resp.StatusCode, string(body))
	}

	var recs []godaddyAPIRecord
	if err := json.Unmarshal(body, &recs); err != nil {
		return nil, err
	}
	return recs, nil
}

func (p *GoDaddyProvider) putRecords(domainName, recordType, name string, records []godaddyAPIRecord) error {
	data, err := json.Marshal(records)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.godaddy.com/v1/domains/%s/records/%s/%s", domainName, recordType, name)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header = p.authHeader()

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("godaddy putRecords: %d %s", resp.StatusCode, string(body))
	}
	return nil
}

func (p *GoDaddyProvider) deleteByNameType(domainName, recordType, name string) error {
	url := fmt.Sprintf("https://api.godaddy.com/v1/domains/%s/records/%s/%s", domainName, recordType, name)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header = p.authHeader()

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		return fmt.Errorf("godaddy deleteRecord: %d %s", resp.StatusCode, string(body))
	}
	return nil
}

// godaddyRecordID encodes a record ID as "domain|TYPE/NAME".
func godaddyRecordID(domain, recordType, name string) string {
	return fmt.Sprintf("%s|%s/%s", domain, recordType, name)
}

// parseGodaddyRecordID decodes a "domain|TYPE/NAME" record ID.
func parseGodaddyRecordID(recordID string) (domain, recordType, name string, err error) {
	parts := strings.SplitN(recordID, "|", 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid godaddy recordID: %s", recordID)
	}
	domain = parts[0]
	typeName := strings.SplitN(parts[1], "/", 2)
	if len(typeName) != 2 {
		return "", "", "", fmt.Errorf("invalid godaddy recordID: %s", recordID)
	}
	return domain, typeName[0], typeName[1], nil
}
