package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const gcoreAPIBase = "https://api.gcore.com/dns/v2"

// GcoreProvider implements the Provider interface for Gcore DNS.
type GcoreProvider struct {
	apiKey     string
	httpClient *http.Client
}

type gcoreZone struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type gcoreRRSet struct {
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	TTL             int                    `json:"ttl"`
	ResourceRecords []gcoreResourceRecord  `json:"resource_records"`
}

type gcoreResourceRecord struct {
	Content []interface{} `json:"content"`
	Enabled bool          `json:"enabled"`
	ID      int           `json:"id,omitempty"`
}

func init() {
	Register("gcore", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &GcoreProvider{
			apiKey:     accessKeySecret,
			httpClient: &http.Client{},
		}, nil
	})
}

func (p *GcoreProvider) GetType() string { return "gcore" }

func (p *GcoreProvider) ListDomains() ([]Domain, error) {
	var result struct {
		Zones []gcoreZone `json:"zones"`
	}
	err := p.request(http.MethodGet, "/zones", nil, &result)
	if err != nil {
		return nil, err
	}
	domains := make([]Domain, len(result.Zones))
	for i, z := range result.Zones {
		domains[i] = Domain{DomainName: z.Name}
	}
	return domains, nil
}

func (p *GcoreProvider) ListRecords(domainName string) ([]Record, error) {
	var result struct {
		RRSets []gcoreRRSet `json:"rrsets"`
	}
	err := p.request(http.MethodGet, "/zones/"+domainName, nil, &result)
	if err != nil {
		return nil, err
	}

	var records []Record
	for _, rr := range result.RRSets {
		for _, rec := range rr.ResourceRecords {
			if len(rec.Content) > 0 {
				value := fmt.Sprintf("%v", rec.Content[0])
				records = append(records, Record{
					RecordID: fmt.Sprintf("%d", rec.ID),
					Host:     rr.Name,
					Type:     rr.Type,
					Value:    value,
					TTL:      int64(rr.TTL),
				})
			}
		}
	}
	return records, nil
}

func (p *GcoreProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	recordName := rr
	if rr == "@" || rr == "" {
		recordName = domainName
	} else {
		recordName = rr + "." + domainName
	}

	payload := map[string]interface{}{
		"ttl": int(ttl),
		"resource_records": []map[string]interface{}{
			{
				"content": []interface{}{value},
				"enabled": true,
			},
		},
	}

	endpoint := fmt.Sprintf("/zones/%s/%s/%s", domainName, recordName, recordType)
	err := p.request(http.MethodPost, endpoint, payload, nil)
	if err != nil {
		return "", err
	}
	return recordName + "/" + recordType, nil
}

func (p *GcoreProvider) UpdateRecord(recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	payload := map[string]interface{}{
		"ttl": int(ttl),
		"resource_records": []map[string]interface{}{
			{
				"content": []interface{}{value},
				"enabled": true,
			},
		},
	}

	// recordID is expected to be "recordName/recordType"
	return p.request(http.MethodPut, "/zones/"+recordID, payload, nil)
}

func (p *GcoreProvider) DeleteRecord(recordID string) error {
	return p.request(http.MethodDelete, "/zones/"+recordID, nil, nil)
}

func (p *GcoreProvider) request(method, path string, data, result interface{}) error {
	apiURL := gcoreAPIBase + path

	var body io.Reader
	if data != nil {
		payload, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("gcore: marshal request: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, apiURL, body)
	if err != nil {
		return fmt.Errorf("gcore: create request: %w", err)
	}
	req.Header.Set("Authorization", "APIKey "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gcore: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("gcore: API error status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("gcore: decode response: %w", err)
		}
	}

	return nil
}

var _ Provider = (*GcoreProvider)(nil)
