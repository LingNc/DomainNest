package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const spaceshipAPIBase = "https://spaceship.dev/api/v1"

// SpaceshipProvider implements the Provider interface for Spaceship DNS.
type SpaceshipProvider struct {
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

func init() {
	Register("spaceship", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &SpaceshipProvider{
			apiKey:     accessKeyID,
			apiSecret:  accessKeySecret,
			httpClient: &http.Client{},
		}, nil
	})
}

func (p *SpaceshipProvider) GetType() string { return "spaceship" }

func (p *SpaceshipProvider) ListDomains() ([]Domain, error) {
	// Spaceship doesn't have a simple domains list endpoint.
	// The user must specify domains explicitly.
	return nil, fmt.Errorf("spaceship: ListDomains not supported; specify domains explicitly")
}

func (p *SpaceshipProvider) ListRecords(domainName string) ([]Record, error) {
	type item struct {
		Type    string `json:"type"`
		Address string `json:"address"`
		Name    string `json:"name"`
		TTL     int    `json:"ttl"`
	}
	type response struct {
		Items []item `json:"items"`
		Total int    `json:"total"`
	}

	var result response
	err := p.request(http.MethodGet, "/dns/records/"+domainName+"?take=500&skip=0", nil, &result)
	if err != nil {
		return nil, err
	}

	records := make([]Record, len(result.Items))
	for i, r := range result.Items {
		records[i] = Record{
			RecordID: r.Name + "/" + r.Type + "/" + r.Address,
			Host:     r.Name,
			Type:     r.Type,
			Value:    r.Address,
			TTL:      int64(r.TTL),
		}
	}
	return records, nil
}

func (p *SpaceshipProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	type item struct {
		Type    string `json:"type"`
		Address string `json:"address"`
		Name    string `json:"name"`
		TTL     int    `json:"ttl"`
	}
	payload := map[string]interface{}{
		"force": true,
		"items": []item{{
			Type:    recordType,
			Address: value,
			Name:    rr,
			TTL:     int(ttl),
		}},
	}
	_, err := p.requestRaw(http.MethodPut, "/dns/records/"+domainName, payload)
	if err != nil {
		return "", err
	}
	return rr + "/" + recordType + "/" + value, nil
}

func (p *SpaceshipProvider) UpdateRecord(recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	// Spaceship uses delete + create for updates
	return fmt.Errorf("spaceship: use DeleteRecord + AddRecord for updates")
}

func (p *SpaceshipProvider) DeleteRecord(recordID string) error {
	// Parse recordID: "name/type/address"
	parts := splitRecordID(recordID)
	if len(parts) != 3 {
		return fmt.Errorf("spaceship: invalid record ID format: %s", recordID)
	}
	type item struct {
		Type    string `json:"type"`
		Address string `json:"address"`
		Name    string `json:"name"`
	}
	payload := []item{{
		Type:    parts[1],
		Address: parts[2],
		Name:    parts[0],
	}}
	_, err := p.requestRaw(http.MethodDelete, "/dns/records/"+parts[0], payload)
	return err
}

func splitRecordID(id string) []string {
	var parts []string
	current := ""
	for _, c := range id {
		if c == '/' {
			parts = append(parts, current)
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

func (p *SpaceshipProvider) request(method, path string, data, result interface{}) error {
	resp, err := p.requestRaw(method, path, data)
	if err != nil {
		return err
	}
	if result != nil {
		if err := json.Unmarshal(resp, result); err != nil {
			return fmt.Errorf("spaceship: decode response: %w", err)
		}
	}
	return nil
}

func (p *SpaceshipProvider) requestRaw(method, path string, data interface{}) ([]byte, error) {
	apiURL := spaceshipAPIBase + path

	var body io.Reader
	if data != nil {
		payload, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("spaceship: marshal request: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, apiURL, body)
	if err != nil {
		return nil, fmt.Errorf("spaceship: create request: %w", err)
	}
	req.Header.Set("X-API-Key", p.apiKey)
	req.Header.Set("X-API-Secret", p.apiSecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("spaceship: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("spaceship: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("spaceship: API error status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

var _ Provider = (*SpaceshipProvider)(nil)
