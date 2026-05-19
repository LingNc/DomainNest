package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

const rainyunAPIBase = "https://api.v2.rainyun.com"

// RainyunProvider implements the Provider interface for Rainyun DNS.
type RainyunProvider struct {
	apiKey     string
	httpClient *http.Client
}

type rainyunRecord struct {
	RecordID int64  `json:"record_id"`
	Host     string `json:"host"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	Line     string `json:"line"`
	TTL      int    `json:"ttl"`
	Level    int    `json:"level"`
}

type rainyunResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func init() {
	Register("rainyun", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &RainyunProvider{
			apiKey:     accessKeySecret,
			httpClient: &http.Client{},
		}, nil
	})
}

func (p *RainyunProvider) GetType() string { return "rainyun" }

func (p *RainyunProvider) ListDomains() ([]Domain, error) {
	// Rainyun uses domain IDs. ListDomains may not be available; return empty.
	return nil, fmt.Errorf("rainyun: ListDomains not supported; specify domain IDs in accessKeyID")
}

func (p *RainyunProvider) ListRecords(domainName string) ([]Record, error) {
	type recordListResp struct {
		TotalRecords int             `json:"TotalRecords"`
		Records      []rainyunRecord `json:"Records"`
	}

	var result recordListResp
	err := p.request(http.MethodGet, fmt.Sprintf("/product/domain/%s/dns/?limit=100&page_no=1", domainName), nil, &result)
	if err != nil {
		return nil, err
	}

	records := make([]Record, len(result.Records))
	for i, r := range result.Records {
		priority := int64(r.Level)
		records[i] = Record{
			RecordID: strconv.FormatInt(r.RecordID, 10),
			Host:     r.Host,
			Type:     r.Type,
			Value:    r.Value,
			TTL:      int64(r.TTL),
			Priority: &priority,
		}
	}
	return records, nil
}

func (p *RainyunProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	payload := map[string]interface{}{
		"host":      rr,
		"line":      "DEFAULT",
		"level":     10,
		"ttl":       int(ttl),
		"type":      recordType,
		"value":     value,
		"record_id": 0,
	}
	if priority != nil {
		payload["level"] = int(*priority)
	}

	err := p.request(http.MethodPost, fmt.Sprintf("/product/domain/%s/dns", domainName), payload, nil)
	if err != nil {
		return "", err
	}
	return rr + "/" + recordType, nil
}

func (p *RainyunProvider) UpdateRecord(recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	payload := map[string]interface{}{
		"host":      rr,
		"line":      "DEFAULT",
		"level":     10,
		"ttl":       int(ttl),
		"type":      recordType,
		"value":     value,
		"record_id": recordID,
	}
	if priority != nil {
		payload["level"] = int(*priority)
	}

	return p.request(http.MethodPatch, "/product/domain/"+rr+"/dns", payload, nil)
}

func (p *RainyunProvider) DeleteRecord(recordID string) error {
	return p.request(http.MethodDelete, "/product/domain/"+recordID+"/dns", nil, nil)
}

func (p *RainyunProvider) request(method, path string, data, result interface{}) error {
	apiURL := rainyunAPIBase + path

	var body io.Reader
	if data != nil {
		payload, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("rainyun: marshal request: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, apiURL, body)
	if err != nil {
		return fmt.Errorf("rainyun: create request: %w", err)
	}
	req.Header.Set("x-api-key", p.apiKey)
	if method == http.MethodPost || method == http.MethodPatch || method == http.MethodPut {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("rainyun: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("rainyun: read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("rainyun: API error status %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp rainyunResp
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return fmt.Errorf("rainyun: decode response: %w", err)
	}
	if apiResp.Code != 200 {
		return fmt.Errorf("rainyun: API error code %d: %s", apiResp.Code, apiResp.Message)
	}

	if result != nil && len(apiResp.Data) > 0 {
		if err := json.Unmarshal(apiResp.Data, result); err != nil {
			return fmt.Errorf("rainyun: decode data: %w", err)
		}
	}

	return nil
}

var _ Provider = (*RainyunProvider)(nil)
