package dns

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const cloudnsAPIBase = "https://api.cloudns.net/dns"

// ClouDNSProvider implements the Provider interface for ClouDNS.
type ClouDNSProvider struct {
	authID     string
	authPass   string
	httpClient *http.Client
}

type cloudnsRecord struct {
	ID     string `json:"id"`
	Host   string `json:"host"`
	Record string `json:"record"`
	Type   string `json:"type"`
	TTL    string `json:"ttl"`
}

type cloudnsResp struct {
	Status            string `json:"status"`
	StatusDescription string `json:"statusDescription"`
}

func init() {
	Register("cloudns", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &ClouDNSProvider{
			authID:     accessKeyID,
			authPass:   accessKeySecret,
			httpClient: &http.Client{},
		}, nil
	})
}

func (p *ClouDNSProvider) GetType() string { return "cloudns" }

func (p *ClouDNSProvider) ListDomains() ([]Domain, error) {
	params := p.baseParams()
	var result map[string]struct {
		Name string `json:"name"`
	}
	err := p.request("/domains/list.json", params, &result)
	if err != nil {
		return nil, err
	}
	domains := make([]Domain, 0, len(result))
	for _, d := range result {
		domains = append(domains, Domain{DomainName: d.Name})
	}
	return domains, nil
}

func (p *ClouDNSProvider) ListRecords(domainName string) ([]Record, error) {
	params := p.baseParams()
	params.Set("domain-name", domainName)

	var result map[string]cloudnsRecord
	err := p.request("/records.json", params, &result)
	if err != nil {
		return nil, err
	}

	records := make([]Record, 0, len(result))
	for _, r := range result {
		rec := Record{
			RecordID: r.ID,
			Host:     r.Host,
			Type:     r.Type,
			Value:    r.Record,
		}
		records = append(records, rec)
	}
	return records, nil
}

func (p *ClouDNSProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	params := p.baseParams()
	params.Set("domain-name", domainName)
	params.Set("host", rr)
	params.Set("record", value)
	params.Set("type", recordType)
	params.Set("ttl", fmt.Sprintf("%d", ttl))

	var result struct {
		Status            string `json:"status"`
		StatusDescription string `json:"statusDescription"`
		Data              string `json:"data"`
	}
	err := p.request("/add-record.json", params, &result)
	if err != nil {
		return "", err
	}
	if result.Status != "Success" {
		return "", fmt.Errorf("cloudns: %s", result.StatusDescription)
	}
	return result.Data, nil
}

func (p *ClouDNSProvider) UpdateRecord(recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	params := p.baseParams()
	params.Set("record-id", recordID)
	params.Set("host", rr)
	params.Set("record", value)
	params.Set("ttl", fmt.Sprintf("%d", ttl))

	var result cloudnsResp
	err := p.request("/modify-record.json", params, &result)
	if err != nil {
		return err
	}
	if result.Status != "Success" {
		return fmt.Errorf("cloudns: %s", result.StatusDescription)
	}
	return nil
}

func (p *ClouDNSProvider) DeleteRecord(recordID string) error {
	params := p.baseParams()
	params.Set("record-id", recordID)

	var result cloudnsResp
	err := p.request("/delete-record.json", params, &result)
	if err != nil {
		return err
	}
	if result.Status != "Success" {
		return fmt.Errorf("cloudns: %s", result.StatusDescription)
	}
	return nil
}

func (p *ClouDNSProvider) baseParams() url.Values {
	params := url.Values{}
	params.Set("auth-id", p.authID)
	params.Set("auth-password", p.authPass)
	return params
}

func (p *ClouDNSProvider) request(action string, params url.Values, result interface{}) error {
	apiURL := cloudnsAPIBase + action

	resp, err := p.httpClient.PostForm(apiURL, params)
	if err != nil {
		return fmt.Errorf("cloudns: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("cloudns: API error status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("cloudns: decode response: %w", err)
		}
	}

	return nil
}

var _ Provider = (*ClouDNSProvider)(nil)
