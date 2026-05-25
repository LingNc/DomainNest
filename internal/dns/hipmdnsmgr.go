package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const hipmDnsMgrEndpoint = "https://dnsmgr.example.com"

// HiPMDnsMgrProvider implements the Provider interface for HiPM DNS Manager.
type HiPMDnsMgrProvider struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

type hipmAPIResp struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
	Msg  string          `json:"msg"`
}

type hipmDomain struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type hipmRecord struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
	Line  string `json:"line"`
}

type hipmRecordList struct {
	Total int         `json:"total"`
	List  []hipmRecord `json:"list"`
}

func init() {
	Register("hipmdnsmgr", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		baseURL := endpoint
		if baseURL == "" {
			baseURL = hipmDnsMgrEndpoint
		}
		return &HiPMDnsMgrProvider{
			baseURL:    baseURL,
			apiToken:   accessKeySecret,
			httpClient: &http.Client{},
		}, nil
	})
}

func (p *HiPMDnsMgrProvider) GetType() string { return "hipmdnsmgr" }

func (p *HiPMDnsMgrProvider) ListDomains() ([]Domain, error) {
	var apiResp hipmAPIResp
	err := p.request("GET", "/domains?page=1&pageSize=100", nil, &apiResp)
	if err != nil {
		return nil, err
	}
	if apiResp.Code != 0 {
		return nil, fmt.Errorf("hipmdnsmgr: %s", apiResp.Msg)
	}
	var domains []hipmDomain
	if err := json.Unmarshal(apiResp.Data, &domains); err != nil {
		return nil, err
	}
	result := make([]Domain, len(domains))
	for i, d := range domains {
		result[i] = Domain{DomainName: d.Name}
	}
	return result, nil
}

func (p *HiPMDnsMgrProvider) ListRecords(domainName string) ([]Record, error) {
	domainID, err := p.getDomainID(domainName)
	if err != nil {
		return nil, err
	}

	var apiResp hipmAPIResp
	err = p.request("GET", fmt.Sprintf("/domains/%d/records?page=1&pageSize=999", domainID), nil, &apiResp)
	if err != nil {
		return nil, err
	}
	if apiResp.Code != 0 {
		return nil, fmt.Errorf("hipmdnsmgr: %s", apiResp.Msg)
	}
	var recordList hipmRecordList
	if err := json.Unmarshal(apiResp.Data, &recordList); err != nil {
		return nil, err
	}
	records := make([]Record, len(recordList.List))
	for i, r := range recordList.List {
		records[i] = Record{
			RecordID: r.ID,
			Host:     r.Name,
			Type:     r.Type,
			Value:    r.Value,
			TTL:      int64(r.TTL),
		}
	}
	return records, nil
}

func (p *HiPMDnsMgrProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	domainID, err := p.getDomainID(domainName)
	if err != nil {
		return "", err
	}

	payload := map[string]interface{}{
		"name":  rr,
		"type":  recordType,
		"value": value,
		"ttl":   int(ttl),
		"line":  "0",
	}

	var apiResp hipmAPIResp
	err = p.request("POST", fmt.Sprintf("/domains/%d/records", domainID), payload, &apiResp)
	if err != nil {
		return "", err
	}
	if apiResp.Code != 0 {
		return "", fmt.Errorf("hipmdnsmgr: %s", apiResp.Msg)
	}
	return fmt.Sprintf("%d/%s", domainID, rr), nil
}

func (p *HiPMDnsMgrProvider) UpdateRecord(domainName, recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	// recordID format: "domainID/recordID"
	parts := splitRecordID(recordID)
	if len(parts) < 2 {
		return fmt.Errorf("hipmdnsmgr: invalid record ID: %s", recordID)
	}
	domainID := parts[0]
	recID := parts[1]

	payload := map[string]interface{}{
		"name":  rr,
		"type":  recordType,
		"value": value,
		"ttl":   int(ttl),
		"line":  "0",
	}

	var apiResp hipmAPIResp
	err := p.request("PUT", fmt.Sprintf("/domains/%s/records/%s", domainID, recID), payload, &apiResp)
	if err != nil {
		return err
	}
	if apiResp.Code != 0 {
		return fmt.Errorf("hipmdnsmgr: %s", apiResp.Msg)
	}
	return nil
}

func (p *HiPMDnsMgrProvider) DeleteRecord(recordID string) error {
	parts := splitRecordID(recordID)
	if len(parts) < 2 {
		return fmt.Errorf("hipmdnsmgr: invalid record ID: %s", recordID)
	}
	domainID := parts[0]
	recID := parts[1]

	var apiResp hipmAPIResp
	err := p.request("DELETE", fmt.Sprintf("/domains/%s/records/%s", domainID, recID), nil, &apiResp)
	if err != nil {
		return err
	}
	if apiResp.Code != 0 {
		return fmt.Errorf("hipmdnsmgr: %s", apiResp.Msg)
	}
	return nil
}

func (p *HiPMDnsMgrProvider) getDomainID(domainName string) (int, error) {
	var apiResp hipmAPIResp
	err := p.request("GET", "/domains?page=1&pageSize=100", nil, &apiResp)
	if err != nil {
		return 0, err
	}
	if apiResp.Code != 0 {
		return 0, fmt.Errorf("hipmdnsmgr: %s", apiResp.Msg)
	}
	var domains []hipmDomain
	if err := json.Unmarshal(apiResp.Data, &domains); err != nil {
		return 0, err
	}
	for _, d := range domains {
		if d.Name == domainName {
			return d.ID, nil
		}
	}
	return 0, fmt.Errorf("hipmdnsmgr: domain not found: %s", domainName)
}

func (p *HiPMDnsMgrProvider) request(method, path string, data interface{}, result *hipmAPIResp) error {
	base := trimSuffix(p.baseURL, "/api")
	apiURL := base + "/api" + path

	var body io.Reader
	if data != nil {
		payload, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("hipmdnsmgr: marshal request: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, apiURL, body)
	if err != nil {
		return fmt.Errorf("hipmdnsmgr: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("hipmdnsmgr: request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("hipmdnsmgr: decode response: %w", err)
	}
	return nil
}

func trimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

var _ Provider = (*HiPMDnsMgrProvider)(nil)
