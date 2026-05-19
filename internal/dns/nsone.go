package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const nsoneAPIBase = "https://api.nsone.net/v1"

// NSOneProvider implements the Provider interface for NS1 DNS.
type NSOneProvider struct {
	apiKey     string
	httpClient *http.Client
}

type nsoneZone struct {
	Name string `json:"zone"`
}

type nsoneRecord struct {
	Domain string   `json:"domain"`
	Type   string   `json:"type"`
	Zone   string   `json:"zone"`
	TTL    int      `json:"ttl"`
	Answers []struct {
		Answer []string `json:"answer"`
	} `json:"answers"`
}

type nsoneRecordRequest struct {
	Answers [][]string `json:"answers"`
	Type    string     `json:"type"`
	TTL     int        `json:"ttl"`
}

func init() {
	Register("nsone", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &NSOneProvider{
			apiKey:     accessKeySecret,
			httpClient: &http.Client{},
		}, nil
	})
}

func (p *NSOneProvider) GetType() string { return "nsone" }

func (p *NSOneProvider) ListDomains() ([]Domain, error) {
	var zones []nsoneZone
	err := p.request(http.MethodGet, "/zones", nil, &zones)
	if err != nil {
		return nil, err
	}
	domains := make([]Domain, len(zones))
	for i, z := range zones {
		domains[i] = Domain{DomainName: z.Name}
	}
	return domains, nil
}

func (p *NSOneProvider) ListRecords(domainName string) ([]Record, error) {
	var records []nsoneRecord
	err := p.request(http.MethodGet, "/zones/"+domainName, nil, &records)
	if err != nil {
		return nil, err
	}
	var result []Record
	for _, r := range records {
		for _, a := range r.Answers {
			if len(a.Answer) > 0 {
				rec := Record{
					RecordID: r.Domain + "/" + r.Type,
					Host:     r.Domain,
					Type:     r.Type,
					Value:    a.Answer[0],
					TTL:      int64(r.TTL),
				}
				result = append(result, rec)
			}
		}
	}
	return result, nil
}

func (p *NSOneProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	fullDomain := rr + "." + domainName
	if rr == "@" || rr == "" {
		fullDomain = domainName
	}

	ttlInt := int(ttl)
	if ttlInt <= 0 {
		ttlInt = 60
	}

	payload := nsoneRecordRequest{
		Answers: [][]string{{value}},
		Type:    recordType,
		TTL:     ttlInt,
	}

	err := p.request(http.MethodPut, "/zones/"+domainName+"/"+fullDomain+"/"+recordType, payload, nil)
	if err != nil {
		return "", err
	}
	return fullDomain + "/" + recordType, nil
}

func (p *NSOneProvider) UpdateRecord(recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	// recordID format: "domain/type"
	payload := nsoneRecordRequest{
		Answers: [][]string{{value}},
		Type:    recordType,
		TTL:     int(ttl),
	}

	// POST to update (NSOne uses POST for update, PUT for create)
	return p.request(http.MethodPost, "/zones/"+recordID, payload, nil)
}

func (p *NSOneProvider) DeleteRecord(recordID string) error {
	return p.request(http.MethodDelete, "/zones/"+recordID, nil, nil)
}

func (p *NSOneProvider) request(method, path string, data, result interface{}) error {
	apiURL := nsoneAPIBase + path

	var body io.Reader
	if data != nil {
		payload, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("nsone: marshal request: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, apiURL, body)
	if err != nil {
		return fmt.Errorf("nsone: create request: %w", err)
	}
	req.Header.Set("X-NSONE-Key", p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("nsone: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("nsone: API error status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("nsone: decode response: %w", err)
		}
	}

	return nil
}

var _ Provider = (*NSOneProvider)(nil)
