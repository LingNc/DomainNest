package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const vercelAPIBase = "https://api.vercel.com"

// VercelProvider implements the Provider interface for Vercel DNS.
type VercelProvider struct {
	token      string
	teamID     string
	httpClient *http.Client
}

// VercelRecord represents a Vercel DNS record from the API.
type VercelRecord struct {
	ID        string  `json:"id"`
	Slug      string  `json:"slug"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Value     string  `json:"value"`
	TTL       int64   `json:"ttl"`
	Comment   *string `json:"comment,omitempty"`
}

type vercelListRecordsResp struct {
	Records []VercelRecord `json:"records"`
}

type vercelListDomainsResp struct {
	Domains []struct {
		Name string `json:"name"`
	} `json:"domains"`
}

func init() {
	Register("vercel", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &VercelProvider{
			token:      accessKeySecret,
			teamID:     accessKeyID,
			httpClient: &http.Client{},
		}, nil
	})
}

func (p *VercelProvider) GetType() string { return "vercel" }

func (p *VercelProvider) ListDomains() ([]Domain, error) {
	var result vercelListDomainsResp
	err := p.request(http.MethodGet, "/v6/domains", nil, &result)
	if err != nil {
		return nil, err
	}
	domains := make([]Domain, len(result.Domains))
	for i, d := range result.Domains {
		domains[i] = Domain{DomainName: d.Name}
	}
	return domains, nil
}

func (p *VercelProvider) ListRecords(domainName string) ([]Record, error) {
	var result vercelListRecordsResp
	err := p.request(http.MethodGet, "/v4/domains/"+domainName+"/records", nil, &result)
	if err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(result.Records))
	for _, r := range result.Records {
		rec := Record{
			RecordID: r.ID,
			Host:     r.Name,
			Type:     r.Type,
			Value:    r.Value,
			TTL:      r.TTL,
		}
		records = append(records, rec)
	}
	return records, nil
}

func (p *VercelProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	ttlInt := int(ttl)
	if ttlInt < 60 {
		ttlInt = 60
	}
	payload := map[string]interface{}{
		"name":  rr,
		"type":  recordType,
		"value": value,
		"ttl":   ttlInt,
	}
	if priority != nil {
		payload["mxPriority"] = *priority
	}
	var result struct {
		Record struct {
			ID string `json:"uid"`
		} `json:"record"`
	}
	err := p.request(http.MethodPost, "/v2/domains/"+domainName+"/records", payload, &result)
	if err != nil {
		return "", err
	}
	return result.Record.ID, nil
}

func (p *VercelProvider) UpdateRecord(recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	ttlInt := int(ttl)
	if ttlInt < 60 {
		ttlInt = 60
	}
	payload := map[string]interface{}{
		"name":  rr,
		"type":  recordType,
		"value": value,
		"ttl":   ttlInt,
	}
	if priority != nil {
		payload["mxPriority"] = *priority
	}
	return p.request(http.MethodPatch, "/v1/domains/records/"+recordID, payload, nil)
}

func (p *VercelProvider) DeleteRecord(recordID string) error {
	return p.request(http.MethodDelete, "/v1/domains/records/"+recordID, nil, nil)
}

func (p *VercelProvider) request(method, path string, data, result interface{}) error {
	apiURL := vercelAPIBase + path
	if p.teamID != "" {
		if strings.Contains(apiURL, "?") {
			apiURL += "&teamId=" + p.teamID
		} else {
			apiURL += "?teamId=" + p.teamID
		}
	}

	var body io.Reader
	if data != nil {
		payload, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("vercel: marshal request: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, apiURL, body)
	if err != nil {
		return fmt.Errorf("vercel: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vercel: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("vercel: API error status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("vercel: decode response: %w", err)
		}
	}

	return nil
}

var _ Provider = (*VercelProvider)(nil)
