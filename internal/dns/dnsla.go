package dns

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const dnslaAPIBase = "http://api.dns.la/api"

// DnslaProvider implements the Provider interface for DNS.LA.
type DnslaProvider struct {
	username   string
	token      string
	httpClient *http.Client
}

type dnslaRecord struct {
	ID   string `json:"id"`
	Host string `json:"host"`
	Type int    `json:"type"`
	Data string `json:"data"`
	TTL  int    `json:"ttl"`
}

type dnslaRecordListResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Total   int           `json:"total"`
		Results []dnslaRecord `json:"results"`
	} `json:"data"`
}

type dnslaStatusResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
}

func init() {
	Register("dnsla", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &DnslaProvider{
			username:   accessKeyID,
			token:      accessKeySecret,
			httpClient: &http.Client{},
		}, nil
	})
}

func (p *DnslaProvider) GetType() string { return "dnsla" }

func (p *DnslaProvider) ListDomains() ([]Domain, error) {
	// DNS.LA does not have a simple domains list endpoint.
	// Users should specify domains explicitly.
	return nil, fmt.Errorf("dnsla: ListDomains not supported; specify domains explicitly")
}

func (p *DnslaProvider) ListRecords(domainName string) ([]Record, error) {
	var result dnslaRecordListResp
	err := p.request(http.MethodGet, "/recordList?domain="+domainName+"&pageIndex=1&pageSize=999", nil, &result)
	if err != nil {
		return nil, err
	}
	if result.Code != 200 {
		return nil, fmt.Errorf("dnsla: %s", result.Msg)
	}
	records := make([]Record, len(result.Data.Results))
	for i, r := range result.Data.Results {
		recType := "A"
		if r.Type == 28 {
			recType = "AAAA"
		}
		records[i] = Record{
			RecordID: r.ID,
			Host:     r.Host,
			Type:     recType,
			Value:    r.Data,
			TTL:      int64(r.TTL),
		}
	}
	return records, nil
}

func (p *DnslaProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	typeInt := 1
	if recordType == "AAAA" {
		typeInt = 28
	}
	payload := map[string]interface{}{
		"Domain": domainName,
		"Host":   rr,
		"Type":   typeInt,
		"Data":   value,
		"TTL":    int(ttl),
	}
	var result dnslaStatusResp
	err := p.request(http.MethodPost, "/record", payload, &result)
	if err != nil {
		return "", err
	}
	if result.Code != 200 {
		return "", fmt.Errorf("dnsla: %s", result.Msg)
	}
	return result.Data.ID, nil
}

func (p *DnslaProvider) UpdateRecord(domainName, recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	typeInt := 1
	if recordType == "AAAA" {
		typeInt = 28
	}
	payload := map[string]interface{}{
		"Id":   recordID,
		"Host": rr,
		"Type": typeInt,
		"Data": value,
		"TTL":  int(ttl),
	}
	var result dnslaStatusResp
	err := p.request(http.MethodPut, "/record", payload, &result)
	if err != nil {
		return err
	}
	if result.Code != 200 {
		return fmt.Errorf("dnsla: %s", result.Msg)
	}
	return nil
}

func (p *DnslaProvider) DeleteRecord(recordID string) error {
	return p.request(http.MethodDelete, "/record?id="+recordID, nil, nil)
}

func (p *DnslaProvider) request(method, path string, data, result interface{}) error {
	apiURL := dnslaAPIBase + path

	var body io.Reader
	if data != nil {
		payload, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("dnsla: marshal request: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, apiURL, body)
	if err != nil {
		return fmt.Errorf("dnsla: create request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(p.username + ":" + p.token))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json;charset=utf-8")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dnsla: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("dnsla: API error status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("dnsla: decode response: %w", err)
		}
	}

	return nil
}

var _ Provider = (*DnslaProvider)(nil)
