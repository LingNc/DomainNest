package dns

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const nameComAPIBase = "https://api.name.com"

// NameComProvider implements the Provider interface for Name.com DNS.
type NameComProvider struct {
	username   string
	token      string
	httpClient *http.Client
}

type nameComRecord struct {
	ID         int    `json:"id"`
	DomainName string `json:"domainName"`
	Host       string `json:"host"`
	Fqdn       string `json:"fqdn"`
	Type       string `json:"type"`
	Answer     string `json:"answer"`
	TTL        int    `json:"ttl"`
	Priority   int    `json:"priority"`
}

type nameComListResp struct {
	Records []nameComRecord `json:"records"`
}

func init() {
	Register("name_com", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &NameComProvider{
			username:   accessKeyID,
			token:      accessKeySecret,
			httpClient: &http.Client{},
		}, nil
	})
}

func (p *NameComProvider) GetType() string { return "name_com" }

func (p *NameComProvider) ListDomains() ([]Domain, error) {
	var result struct {
		Domains []struct {
			DomainName string `json:"domainName"`
		} `json:"domains"`
	}
	err := p.request(http.MethodGet, "/v4/domains", nil, &result)
	if err != nil {
		return nil, err
	}
	domains := make([]Domain, len(result.Domains))
	for i, d := range result.Domains {
		domains[i] = Domain{DomainName: d.DomainName}
	}
	return domains, nil
}

func (p *NameComProvider) ListRecords(domainName string) ([]Record, error) {
	var result nameComListResp
	err := p.request(http.MethodGet, "/v4/domains/"+domainName+"/records", nil, &result)
	if err != nil {
		return nil, err
	}
	records := make([]Record, len(result.Records))
	for i, r := range result.Records {
		priorityVal := int64(r.Priority)
		records[i] = Record{
			RecordID: fmt.Sprintf("%d", r.ID),
			Host:     r.Host,
			Type:     r.Type,
			Value:    r.Answer,
			TTL:      int64(r.TTL),
		}
		if priorityVal > 0 {
			records[i].Priority = &priorityVal
		}
	}
	return records, nil
}

func (p *NameComProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	payload := map[string]interface{}{
		"host":   rr,
		"type":   recordType,
		"answer": value,
		"ttl":    int(ttl),
	}
	if priority != nil {
		payload["priority"] = int(*priority)
	}
	var result nameComRecord
	err := p.request(http.MethodPost, "/v4/domains/"+domainName+"/records", payload, &result)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", result.ID), nil
}

func (p *NameComProvider) UpdateRecord(recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	payload := map[string]interface{}{
		"host":   rr,
		"type":   recordType,
		"answer": value,
		"ttl":    int(ttl),
	}
	if priority != nil {
		payload["priority"] = int(*priority)
	}
	return p.request(http.MethodPut, "/v4/domains/"+recordID, payload, nil)
}

func (p *NameComProvider) DeleteRecord(recordID string) error {
	return p.request(http.MethodDelete, "/v4/domains/"+recordID, nil, nil)
}

func (p *NameComProvider) request(method, path string, data, result interface{}) error {
	apiURL := nameComAPIBase + path

	var body io.Reader
	if data != nil {
		payload, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("name_com: marshal request: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, apiURL, body)
	if err != nil {
		return fmt.Errorf("name_com: create request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(p.username + ":" + p.token))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("name_com: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("name_com: API error status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("name_com: decode response: %w", err)
		}
	}

	return nil
}

var _ Provider = (*NameComProvider)(nil)
