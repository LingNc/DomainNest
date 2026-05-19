package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

const (
	tencentCloudDefaultEndpoint = "https://dnspod.tencentcloudapi.com"
	tencentCloudVersion         = "2021-03-23"
)

type tencentCloudProvider struct {
	secretID   string
	secretKey  string
	endpoint   string
	httpClient *http.Client
}

// Response types for TencentCloud DNSPod API

type tcStatus struct {
	Response struct {
		Error struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
	} `json:"Response"`
}

type tcDomainListResp struct {
	Response struct {
		Error struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
		DomainCountInfo struct {
			AllTotal int `json:"AllTotal"`
		} `json:"DomainCountInfo"`
		DomainList []struct {
			Domain      string `json:"Domain"`
			RecordCount int64  `json:"RecordCount"`
		} `json:"DomainList"`
	} `json:"Response"`
}

type tcRecordListResp struct {
	Response struct {
		Error struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
		RecordCountInfo struct {
			TotalCount int `json:"TotalCount"`
		} `json:"RecordCountInfo"`
		RecordList []struct {
			RecordId   int64  `json:"RecordId"`
			SubDomain  string `json:"SubDomain"`
			RecordType string `json:"RecordType"`
			RecordLine string `json:"RecordLine"`
			Value      string `json:"Value"`
			TTL        int64  `json:"TTL"`
			MX         int64  `json:"MX"`
		} `json:"RecordList"`
	} `json:"Response"`
}

type tcCreateRecordResp struct {
	Response struct {
		Error struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
		Record struct {
			RecordId int64 `json:"RecordId"`
		} `json:"Record"`
	} `json:"Response"`
}

func init() {
	Register("tencentcloud", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		if accessKeyID == "" || accessKeySecret == "" {
			return nil, fmt.Errorf("tencentcloud: SecretId and SecretKey are required")
		}
		ep := endpoint
		if ep == "" {
			ep = tencentCloudDefaultEndpoint
		}
		return &tencentCloudProvider{
			secretID:   accessKeyID,
			secretKey:  accessKeySecret,
			endpoint:   ep,
			httpClient: &http.Client{},
		}, nil
	})
}

func (p *tencentCloudProvider) GetType() string { return "tencentcloud" }

func (p *tencentCloudProvider) ListDomains() ([]Domain, error) {
	var allDomains []Domain
	offset := 0
	for {
		reqBody := map[string]interface{}{
			"Limit":  100,
			"Offset": offset,
		}
		var resp tcDomainListResp
		if err := p.request("DescribeDomainList", reqBody, &resp); err != nil {
			return nil, fmt.Errorf("tencentcloud: list domains: %w", err)
		}
		if resp.Response.Error.Code != "" {
			return nil, fmt.Errorf("tencentcloud: list domains: %s - %s", resp.Response.Error.Code, resp.Response.Error.Message)
		}

		for _, d := range resp.Response.DomainList {
			allDomains = append(allDomains, Domain{
				DomainName:  d.Domain,
				RecordCount: d.RecordCount,
			})
		}

		if len(resp.Response.DomainList) < 100 {
			break
		}
		offset += 100
	}
	return allDomains, nil
}

func (p *tencentCloudProvider) ListRecords(domainName string) ([]Record, error) {
	var allRecords []Record
	offset := 0
	for {
		reqBody := map[string]interface{}{
			"Domain": domainName,
			"Limit":  100,
			"Offset": offset,
		}
		var resp tcRecordListResp
		if err := p.request("DescribeRecordList", reqBody, &resp); err != nil {
			return nil, fmt.Errorf("tencentcloud: list records: %w", err)
		}
		if resp.Response.Error.Code != "" {
			return nil, fmt.Errorf("tencentcloud: list records: %s - %s", resp.Response.Error.Code, resp.Response.Error.Message)
		}

		for _, r := range resp.Response.RecordList {
			rec := Record{
				RecordID: strconv.FormatInt(r.RecordId, 10),
				Host:     r.SubDomain,
				Type:     r.RecordType,
				Value:    r.Value,
				TTL:      r.TTL,
			}
			if r.MX > 0 {
				mx := r.MX
				rec.Priority = &mx
			}
			allRecords = append(allRecords, rec)
		}

		if len(resp.Response.RecordList) < 100 {
			break
		}
		offset += 100
	}
	return allRecords, nil
}

func (p *tencentCloudProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	reqBody := map[string]interface{}{
		"Domain":     domainName,
		"SubDomain":  rr,
		"RecordType": recordType,
		"RecordLine": "默认",
		"Value":      value,
		"TTL":        ttl,
	}
	if priority != nil {
		reqBody["MX"] = *priority
	}

	var resp tcCreateRecordResp
	if err := p.request("CreateRecord", reqBody, &resp); err != nil {
		return "", fmt.Errorf("tencentcloud: add record: %w", err)
	}
	if resp.Response.Error.Code != "" {
		return "", fmt.Errorf("tencentcloud: add record: %s - %s", resp.Response.Error.Code, resp.Response.Error.Message)
	}

	return strconv.FormatInt(resp.Response.Record.RecordId, 10), nil
}

func (p *tencentCloudProvider) UpdateRecord(recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	id, err := strconv.ParseInt(recordID, 10, 64)
	if err != nil {
		return fmt.Errorf("tencentcloud: invalid record ID: %s", recordID)
	}

	reqBody := map[string]interface{}{
		"RecordId":   id,
		"SubDomain":  rr,
		"RecordType": recordType,
		"RecordLine": "默认",
		"Value":      value,
		"TTL":        ttl,
	}
	if priority != nil {
		reqBody["MX"] = *priority
	}

	var resp tcStatus
	if err := p.request("ModifyRecord", reqBody, &resp); err != nil {
		return fmt.Errorf("tencentcloud: update record: %w", err)
	}
	if resp.Response.Error.Code != "" {
		return fmt.Errorf("tencentcloud: update record: %s - %s", resp.Response.Error.Code, resp.Response.Error.Message)
	}
	return nil
}

func (p *tencentCloudProvider) DeleteRecord(recordID string) error {
	id, err := strconv.ParseInt(recordID, 10, 64)
	if err != nil {
		return fmt.Errorf("tencentcloud: invalid record ID: %s", recordID)
	}

	reqBody := map[string]interface{}{
		"RecordId": id,
	}

	var resp tcStatus
	if err := p.request("DeleteRecord", reqBody, &resp); err != nil {
		return fmt.Errorf("tencentcloud: delete record: %w", err)
	}
	if resp.Response.Error.Code != "" {
		return fmt.Errorf("tencentcloud: delete record: %s - %s", resp.Response.Error.Code, resp.Response.Error.Message)
	}
	return nil
}

func (p *tencentCloudProvider) request(action string, data interface{}, result interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	payload := string(jsonBytes)

	req, err := http.NewRequest("POST", p.endpoint, bytes.NewReader(jsonBytes))
	if err != nil {
		return err
	}

	// Use the existing TencentCloud signer from signing.go
	tencentCloudSigner(p.secretID, p.secretKey, action, payload, "dnspod", req)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, result)
}
