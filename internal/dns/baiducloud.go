package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

const baiduEndpoint = "https://bcd.baidubce.com"

// BaiduCloudProvider implements the Provider interface for Baidu Cloud DNS.
type BaiduCloudProvider struct {
	accessKeyID string
	secretKey   string
	httpClient  *http.Client
}

type baiduRecord struct {
	RecordId uint   `json:"recordId"`
	Domain   string `json:"domain"`
	View     string `json:"view"`
	Rdtype   string `json:"rdtype"`
	Rdata    string `json:"rdata"`
	ZoneName string `json:"zoneName"`
	TTL      int    `json:"ttl"`
	Status   string `json:"status"`
}

type baiduRecordsResp struct {
	TotalCount int           `json:"totalCount"`
	Result     []baiduRecord `json:"result"`
}

func init() {
	Register("baiducloud", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &BaiduCloudProvider{
			accessKeyID: accessKeyID,
			secretKey:   accessKeySecret,
			httpClient:  &http.Client{},
		}, nil
	})
}

func (p *BaiduCloudProvider) GetType() string { return "baiducloud" }

func (p *BaiduCloudProvider) ListDomains() ([]Domain, error) {
	payload := map[string]interface{}{
		"pageNo":   1,
		"pageSize": 100,
	}
	var result struct {
		TotalCount int `json:"totalCount"`
		Result     []struct {
			Name string `json:"name"`
		} `json:"result"`
	}
	err := p.request("POST", "/v1/domain/list", payload, &result)
	if err != nil {
		return nil, err
	}
	domains := make([]Domain, len(result.Result))
	for i, d := range result.Result {
		domains[i] = Domain{DomainName: d.Name}
	}
	return domains, nil
}

func (p *BaiduCloudProvider) ListRecords(domainName string) ([]Record, error) {
	payload := map[string]interface{}{
		"domain":   domainName,
		"pageNo":   1,
		"pageSize": 100,
	}
	var result baiduRecordsResp
	err := p.request("POST", "/v1/domain/resolve/list", payload, &result)
	if err != nil {
		return nil, err
	}
	records := make([]Record, len(result.Result))
	for i, r := range result.Result {
		records[i] = Record{
			RecordID: strconv.FormatUint(uint64(r.RecordId), 10),
			Host:     r.Domain,
			Type:     r.Rdtype,
			Value:    r.Rdata,
			TTL:      int64(r.TTL),
		}
	}
	return records, nil
}

func (p *BaiduCloudProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	payload := map[string]interface{}{
		"domain":   rr,
		"rdType":   recordType,
		"rdata":    value,
		"zoneName": domainName,
		"ttl":      int(ttl),
	}
	var result struct {
		RecordId uint `json:"recordId"`
	}
	err := p.request("POST", "/v1/domain/resolve/add", payload, &result)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(uint64(result.RecordId), 10), nil
}

func (p *BaiduCloudProvider) UpdateRecord(recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	id, _ := strconv.ParseUint(recordID, 10, 64)
	payload := map[string]interface{}{
		"recordId": uint(id),
		"domain":   rr,
		"rdType":   recordType,
		"rdata":    value,
		"ttl":      int(ttl),
	}
	return p.request("POST", "/v1/domain/resolve/edit", payload, nil)
}

func (p *BaiduCloudProvider) DeleteRecord(recordID string) error {
	id, _ := strconv.ParseUint(recordID, 10, 64)
	payload := map[string]interface{}{
		"recordId": uint(id),
	}
	return p.request("POST", "/v1/domain/resolve/delete", payload, nil)
}

func (p *BaiduCloudProvider) request(method, path string, data, result interface{}) error {
	apiURL := baiduEndpoint + path

	var body []byte
	var err error
	if data != nil {
		body, err = json.Marshal(data)
		if err != nil {
			return fmt.Errorf("baiducloud: marshal request: %w", err)
		}
	}

	req, err := http.NewRequest(method, apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("baiducloud: create request: %w", err)
	}

	baiduSigner(p.accessKeyID, p.secretKey, "bcd.baidubce.com", req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("baiducloud: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("baiducloud: API error status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("baiducloud: decode response: %w", err)
		}
	}

	return nil
}

var _ Provider = (*BaiduCloudProvider)(nil)
