package dns

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

const aliesaEndpoint = "https://esa.cn-hangzhou.aliyuncs.com/"

// AliesaProvider implements the Provider interface for Alibaba Cloud ESA (Edge Security Acceleration).
type AliesaProvider struct {
	accessKeyID string
	secretKey   string
	httpClient  *http.Client
}

type aliesaSite struct {
	SiteId    int64  `json:"SiteId"`
	SiteName  string `json:"SiteName"`
	AccessType string `json:"AccessType"`
}

type aliesaRecord struct {
	RecordId   int64  `json:"RecordId"`
	RecordName string `json:"RecordName"`
	Type       string `json:"Type"`
	Data       struct {
		Value string `json:"Value"`
	} `json:"Data"`
	TTL int `json:"Ttl"`
}

func init() {
	Register("aliesa", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &AliesaProvider{
			accessKeyID: accessKeyID,
			secretKey:   accessKeySecret,
			httpClient:  &http.Client{},
		}, nil
	})
}

func (p *AliesaProvider) GetType() string { return "aliesa" }

func (p *AliesaProvider) ListDomains() ([]Domain, error) {
	params := url.Values{}
	params.Set("Action", "ListSites")
	params.Set("PageSize", "500")

	var result struct {
		TotalCount int `json:"TotalCount"`
		Sites      []struct {
			SiteId   int64  `json:"SiteId"`
			SiteName string `json:"SiteName"`
		} `json:"Sites"`
	}
	err := p.request("GET", params, &result)
	if err != nil {
		return nil, err
	}
	domains := make([]Domain, len(result.Sites))
	for i, s := range result.Sites {
		domains[i] = Domain{DomainName: s.SiteName}
	}
	return domains, nil
}

func (p *AliesaProvider) ListRecords(domainName string) ([]Record, error) {
	siteId, err := p.getSiteId(domainName)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("Action", "ListRecords")
	params.Set("SiteId", strconv.FormatInt(siteId, 10))
	params.Set("PageSize", "500")

	var result struct {
		TotalCount int `json:"TotalCount"`
		Records    []aliesaRecord `json:"Records"`
	}
	err = p.request("GET", params, &result)
	if err != nil {
		return nil, err
	}

	records := make([]Record, len(result.Records))
	for i, r := range result.Records {
		records[i] = Record{
			RecordID: strconv.FormatInt(r.RecordId, 10),
			Host:     r.RecordName,
			Type:     r.Type,
			Value:    r.Data.Value,
			TTL:      int64(r.TTL),
		}
	}
	return records, nil
}

func (p *AliesaProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	siteId, err := p.getSiteId(domainName)
	if err != nil {
		return "", err
	}

	recordName := rr
	if rr != "@" && rr != "" {
		recordName = rr + "." + domainName
	}

	params := url.Values{}
	params.Set("Action", "CreateRecord")
	params.Set("SiteId", strconv.FormatInt(siteId, 10))
	params.Set("RecordName", recordName)
	params.Set("Type", recordType)
	params.Set("Data.Value", value)
	params.Set("Ttl", strconv.FormatInt(ttl, 10))

	var result struct {
		RecordId int64 `json:"RecordId"`
	}
	err = p.request("GET", params, &result)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(result.RecordId, 10), nil
}

func (p *AliesaProvider) UpdateRecord(domainName, recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	params := url.Values{}
	params.Set("Action", "UpdateRecord")
	params.Set("RecordId", recordID)
	params.Set("Type", recordType)
	params.Set("Data.Value", value)
	params.Set("Ttl", strconv.FormatInt(ttl, 10))

	return p.request("GET", params, nil)
}

func (p *AliesaProvider) DeleteRecord(recordID string) error {
	params := url.Values{}
	params.Set("Action", "DeleteRecord")
	params.Set("RecordId", recordID)

	return p.request("GET", params, nil)
}

func (p *AliesaProvider) getSiteId(domainName string) (int64, error) {
	params := url.Values{}
	params.Set("Action", "ListSites")
	params.Set("SiteName", domainName)

	var result struct {
		TotalCount int `json:"TotalCount"`
		Sites      []struct {
			SiteId int64 `json:"SiteId"`
		} `json:"Sites"`
	}
	err := p.request("GET", params, &result)
	if err != nil {
		return 0, err
	}
	if result.TotalCount == 0 {
		return 0, fmt.Errorf("aliesa: site not found: %s", domainName)
	}
	return result.Sites[0].SiteId, nil
}

func (p *AliesaProvider) request(method string, params url.Values, result interface{}) error {
	aliyunSigner(p.accessKeyID, p.secretKey, &params, method, "2024-09-10")

	apiURL := aliesaEndpoint + "?" + params.Encode()

	req, err := http.NewRequest(method, apiURL, nil)
	if err != nil {
		return fmt.Errorf("aliesa: create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("aliesa: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("aliesa: API error status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("aliesa: decode response: %w", err)
		}
	}

	return nil
}

var _ Provider = (*AliesaProvider)(nil)
