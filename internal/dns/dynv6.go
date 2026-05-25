package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const dynv6APIBase = "https://dynv6.com"

// Dynv6Provider implements the Provider interface for dynv6.com DNS.
type Dynv6Provider struct {
	token      string
	httpClient *http.Client
}

type dynv6Zone struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type dynv6Record struct {
	ID     uint   `json:"id"`
	ZoneID uint   `json:"zone_id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Data   string `json:"data"`
}

func init() {
	Register("dynv6", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &Dynv6Provider{
			token:      accessKeySecret,
			httpClient: &http.Client{},
		}, nil
	})
}

func (p *Dynv6Provider) GetType() string { return "dynv6" }

func (p *Dynv6Provider) ListDomains() ([]Domain, error) {
	var zones []dynv6Zone
	err := p.request(http.MethodGet, "/api/v2/zones", nil, &zones)
	if err != nil {
		return nil, err
	}
	domains := make([]Domain, len(zones))
	for i, z := range zones {
		domains[i] = Domain{DomainName: z.Name}
	}
	return domains, nil
}

func (p *Dynv6Provider) ListRecords(domainName string) ([]Record, error) {
	zone, err := p.findZone(domainName)
	if err != nil {
		return nil, err
	}
	if zone == nil {
		return nil, fmt.Errorf("dynv6: zone not found: %s", domainName)
	}

	var records []dynv6Record
	err = p.request(http.MethodGet, fmt.Sprintf("/api/v2/zones/%d/records", zone.ID), nil, &records)
	if err != nil {
		return nil, err
	}

	result := make([]Record, len(records))
	for i, r := range records {
		result[i] = Record{
			RecordID: fmt.Sprintf("%d", r.ID),
			Host:     r.Name,
			Type:     r.Type,
			Value:    r.Data,
			TTL:      300, // dynv6 doesn't expose per-record TTL
		}
	}
	return result, nil
}

func (p *Dynv6Provider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	zone, err := p.findZone(domainName)
	if err != nil {
		return "", err
	}
	if zone == nil {
		return "", fmt.Errorf("dynv6: zone not found: %s", domainName)
	}

	payload := map[string]interface{}{
		"name": rr,
		"type": recordType,
		"data": value,
	}
	var result dynv6Record
	err = p.request(http.MethodPost, fmt.Sprintf("/api/v2/zones/%d/records", zone.ID), payload, &result)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", result.ID), nil
}

func (p *Dynv6Provider) UpdateRecord(domainName, recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	payload := map[string]interface{}{
		"name": rr,
		"type": recordType,
		"data": value,
	}
	return p.request(http.MethodPatch, "/api/v2/zones/records/"+recordID, payload, nil)
}

func (p *Dynv6Provider) DeleteRecord(recordID string) error {
	return p.request(http.MethodDelete, "/api/v2/zones/records/"+recordID, nil, nil)
}

func (p *Dynv6Provider) findZone(domainName string) (*dynv6Zone, error) {
	var zones []dynv6Zone
	err := p.request(http.MethodGet, "/api/v2/zones", nil, &zones)
	if err != nil {
		return nil, err
	}
	for _, z := range zones {
		if z.Name == domainName {
			return &z, nil
		}
	}
	return nil, nil
}

func (p *Dynv6Provider) request(method, path string, data, result interface{}) error {
	apiURL := dynv6APIBase + path

	var body io.Reader
	if data != nil {
		payload, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("dynv6: marshal request: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, apiURL, body)
	if err != nil {
		return fmt.Errorf("dynv6: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dynv6: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("dynv6: API error status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("dynv6: decode response: %w", err)
		}
	}

	return nil
}

var _ Provider = (*Dynv6Provider)(nil)
