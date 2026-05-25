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
	edgeoneEndpoint = "https://teo.tencentcloudapi.com"
	edgeoneVersion  = "2022-09-01"
)

// EdgeOneProvider implements the Provider interface for Tencent EdgeOne DNS.
type EdgeOneProvider struct {
	secretId  string
	secretKey string
	httpClient *http.Client
}

type edgeOneRecord struct {
	ZoneId   string `json:"ZoneId"`
	Name     string `json:"Name"`
	Type     string `json:"Type"`
	Content  string `json:"Content"`
	TTL      int    `json:"TTL"`
	RecordId string `json:"RecordId"`
	Status   string `json:"Status"`
}

func init() {
	Register("edgeone", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &EdgeOneProvider{
			secretId:   accessKeyID,
			secretKey:  accessKeySecret,
			httpClient: &http.Client{},
		}, nil
	})
}

func (p *EdgeOneProvider) GetType() string { return "edgeone" }

func (p *EdgeOneProvider) ListDomains() ([]Domain, error) {
	var result struct {
		Response struct {
			Zones []struct {
				ZoneId   string `json:"ZoneId"`
				ZoneName string `json:"ZoneName"`
			} `json:"Zones"`
			TotalCount int `json:"TotalCount"`
		} `json:"Response"`
	}
	err := p.request("DescribeZones", `{"Limit":500}`, &result)
	if err != nil {
		return nil, err
	}
	domains := make([]Domain, len(result.Response.Zones))
	for i, z := range result.Response.Zones {
		domains[i] = Domain{DomainName: z.ZoneName}
	}
	return domains, nil
}

func (p *EdgeOneProvider) ListRecords(domainName string) ([]Record, error) {
	zoneId, err := p.getZoneId(domainName)
	if err != nil {
		return nil, err
	}

	payload := fmt.Sprintf(`{"ZoneId":"%s","Limit":500}`, zoneId)
	var result struct {
		Response struct {
			DnsRecords []edgeOneRecord `json:"DnsRecords"`
			TotalCount int             `json:"TotalCount"`
		} `json:"Response"`
	}
	err = p.request("DescribeDnsRecords", payload, &result)
	if err != nil {
		return nil, err
	}

	records := make([]Record, len(result.Response.DnsRecords))
	for i, r := range result.Response.DnsRecords {
		records[i] = Record{
			RecordID: r.RecordId,
			Host:     r.Name,
			Type:     r.Type,
			Value:    r.Content,
			TTL:      int64(r.TTL),
		}
	}
	return records, nil
}

func (p *EdgeOneProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	zoneId, err := p.getZoneId(domainName)
	if err != nil {
		return "", err
	}

	fullName := rr
	if rr != "@" && rr != "" {
		fullName = rr + "." + domainName
	}

	payload := fmt.Sprintf(`{"ZoneId":"%s","Type":"%s","Name":"%s","Content":"%s","TTL":%d}`,
		zoneId, recordType, fullName, value, ttl)
	var result struct {
		Response struct {
			RecordId string `json:"RecordId"`
		} `json:"Response"`
	}
	err = p.request("CreateDnsRecord", payload, &result)
	if err != nil {
		return "", err
	}
	return result.Response.RecordId, nil
}

func (p *EdgeOneProvider) UpdateRecord(domainName, recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	payload := fmt.Sprintf(`{"RecordId":"%s","Type":"%s","Name":"%s","Content":"%s","TTL":%d}`,
		recordID, recordType, rr, value, ttl)
	return p.request("ModifyDnsRecords", payload, nil)
}

func (p *EdgeOneProvider) DeleteRecord(recordID string) error {
	payload := fmt.Sprintf(`{"RecordId":"%s"}`, recordID)
	return p.request("DeleteDnsRecords", payload, nil)
}

func (p *EdgeOneProvider) getZoneId(domainName string) (string, error) {
	payload := fmt.Sprintf(`{"Filters":[{"Name":"zone-name","Values":["%s"]}]}`, domainName)
	var result struct {
		Response struct {
			Zones []struct {
				ZoneId   string `json:"ZoneId"`
				ZoneName string `json:"ZoneName"`
			} `json:"Zones"`
		} `json:"Response"`
	}
	err := p.request("DescribeZones", payload, &result)
	if err != nil {
		return "", err
	}
	for _, z := range result.Response.Zones {
		if z.ZoneName == domainName {
			return z.ZoneId, nil
		}
	}
	return "", fmt.Errorf("edgeone: zone not found: %s", domainName)
}

func (p *EdgeOneProvider) request(action, payload string, result interface{}) error {
	req, err := http.NewRequest("POST", edgeoneEndpoint, bytes.NewReader([]byte(payload)))
	if err != nil {
		return fmt.Errorf("edgeone: create request: %w", err)
	}

	tencentCloudSigner(p.secretId, p.secretKey, action, payload, "teo", req)
	req.Header.Set("X-TC-Version", edgeoneVersion)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("edgeone: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("edgeone: API error status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("edgeone: decode response: %w", err)
		}
	}

	return nil
}

// Suppress unused import
var _ = strconv.Itoa

var _ Provider = (*EdgeOneProvider)(nil)
