package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

const cloudflareBaseURL = "https://api.cloudflare.com/client/v4"

type cloudflareProvider struct {
	token      string
	httpClient *http.Client
}

// Cloudflare API response wrappers

type cfResponse struct {
	Success  bool     `json:"success"`
	Errors   []cfMsg  `json:"errors"`
	Messages []string `json:"messages"`
}

type cfMsg struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cfZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type cfZonesResp struct {
	cfResponse
	Result []cfZone `json:"result"`
}

type cfRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int64  `json:"ttl"`
}

type cfRecordsResp struct {
	cfResponse
	Result []cfRecord `json:"result"`
}

type cfCreateReq struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int64  `json:"ttl"`
}

type cfUpdateReq struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int64  `json:"ttl"`
}

func init() {
	Register("cloudflare", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		if accessKeyID == "" {
			return nil, fmt.Errorf("cloudflare: API token is required")
		}
		return &cloudflareProvider{
			token:      accessKeyID,
			httpClient: &http.Client{},
		}, nil
	})
}

func (p *cloudflareProvider) GetType() string { return "cloudflare" }

func (p *cloudflareProvider) ListDomains() ([]Domain, error) {
	var allDomains []Domain
	page := 1
	for {
		params := url.Values{}
		params.Set("page", strconv.Itoa(page))
		params.Set("per_page", "50")

		var resp cfZonesResp
		if err := p.request("GET", cloudflareBaseURL+"/zones?"+params.Encode(), nil, &resp); err != nil {
			return nil, fmt.Errorf("cloudflare: list domains: %w", err)
		}
		if !resp.Success {
			return nil, fmt.Errorf("cloudflare: list domains: %s", cfErrMsgs(resp.Errors))
		}

		for _, z := range resp.Result {
			allDomains = append(allDomains, Domain{
				DomainName:  z.Name,
				RecordCount: 0, // Cloudflare doesn't return record count in zone list
			})
		}

		if len(resp.Result) < 50 {
			break
		}
		page++
	}
	return allDomains, nil
}

func (p *cloudflareProvider) ListRecords(domainName string) ([]Record, error) {
	zoneID, err := p.getZoneID(domainName)
	if err != nil {
		return nil, err
	}

	var allRecords []Record
	page := 1
	for {
		params := url.Values{}
		params.Set("page", strconv.Itoa(page))
		params.Set("per_page", "100")

		var resp cfRecordsResp
		if err := p.request("GET", fmt.Sprintf("%s/zones/%s/dns_records?%s", cloudflareBaseURL, zoneID, params.Encode()), nil, &resp); err != nil {
			return nil, fmt.Errorf("cloudflare: list records: %w", err)
		}
		if !resp.Success {
			return nil, fmt.Errorf("cloudflare: list records: %s", cfErrMsgs(resp.Errors))
		}

		for _, r := range resp.Result {
			rec := Record{
				RecordID: r.ID,
				Host:     r.Name,
				Type:     r.Type,
				Value:    r.Content,
				TTL:      r.TTL,
			}
			allRecords = append(allRecords, rec)
		}

		if len(resp.Result) < 100 {
			break
		}
		page++
	}
	return allRecords, nil
}

func (p *cloudflareProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	zoneID, err := p.getZoneID(domainName)
	if err != nil {
		return "", err
	}

	name := buildFQDN(rr, domainName)
	body := cfCreateReq{
		Type:    recordType,
		Name:    name,
		Content: value,
		TTL:     ttl,
	}

	var resp cfResponse
	if err := p.request("POST", fmt.Sprintf("%s/zones/%s/dns_records", cloudflareBaseURL, zoneID), body, &resp); err != nil {
		return "", fmt.Errorf("cloudflare: add record: %w", err)
	}
	if !resp.Success {
		return "", fmt.Errorf("cloudflare: add record: %s", cfErrMsgs(resp.Errors))
	}

	// Extract record ID from the response by re-listing
	records, err := p.ListRecords(domainName)
	if err != nil {
		return "", nil // Record was created but we couldn't get the ID
	}
	for _, r := range records {
		if r.Host == name && r.Type == recordType && r.Value == value {
			return r.RecordID, nil
		}
	}
	return "", nil
}

func (p *cloudflareProvider) UpdateRecord(domainName, recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	// We need the zone ID. The recordID alone isn't enough for Cloudflare.
	// We'll search across zones to find the record.
	zoneID, name, err := p.findRecordZone(recordID, rr)
	if err != nil {
		return fmt.Errorf("cloudflare: update record: %w", err)
	}

	body := cfUpdateReq{
		Type:    recordType,
		Name:    name,
		Content: value,
		TTL:     ttl,
	}

	var resp cfResponse
	if err := p.request("PUT", fmt.Sprintf("%s/zones/%s/dns_records/%s", cloudflareBaseURL, zoneID, recordID), body, &resp); err != nil {
		return fmt.Errorf("cloudflare: update record: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("cloudflare: update record: %s", cfErrMsgs(resp.Errors))
	}
	return nil
}

func (p *cloudflareProvider) DeleteRecord(recordID string) error {
	zoneID, _, err := p.findRecordZone(recordID, "")
	if err != nil {
		return fmt.Errorf("cloudflare: delete record: %w", err)
	}

	var resp cfResponse
	if err := p.request("DELETE", fmt.Sprintf("%s/zones/%s/dns_records/%s", cloudflareBaseURL, zoneID, recordID), nil, &resp); err != nil {
		return fmt.Errorf("cloudflare: delete record: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("cloudflare: delete record: %s", cfErrMsgs(resp.Errors))
	}
	return nil
}

// getZoneID returns the Cloudflare zone ID for a domain name.
func (p *cloudflareProvider) getZoneID(domainName string) (string, error) {
	params := url.Values{}
	params.Set("name", domainName)

	var resp cfZonesResp
	if err := p.request("GET", cloudflareBaseURL+"/zones?"+params.Encode(), nil, &resp); err != nil {
		return "", fmt.Errorf("get zone: %w", err)
	}
	if !resp.Success {
		return "", fmt.Errorf("get zone: %s", cfErrMsgs(resp.Errors))
	}
	if len(resp.Result) == 0 {
		return "", fmt.Errorf("zone not found for domain: %s", domainName)
	}
	return resp.Result[0].ID, nil
}

// findRecordZone finds the zone ID and record name for a given record ID.
// It iterates through all zones and their records to find the match.
func (p *cloudflareProvider) findRecordZone(recordID, rr string) (zoneID, recordName string, err error) {
	page := 1
	for {
		params := url.Values{}
		params.Set("page", strconv.Itoa(page))
		params.Set("per_page", "50")

		var zonesResp cfZonesResp
		if err := p.request("GET", cloudflareBaseURL+"/zones?"+params.Encode(), nil, &zonesResp); err != nil {
			return "", "", err
		}
		if !zonesResp.Success {
			return "", "", fmt.Errorf("%s", cfErrMsgs(zonesResp.Errors))
		}

		for _, zone := range zonesResp.Result {
			var recResp cfRecordsResp
			if err := p.request("GET", fmt.Sprintf("%s/zones/%s/dns_records/%s", cloudflareBaseURL, zone.ID, recordID), nil, &recResp); err != nil {
				continue
			}
			// Single record endpoint returns the record directly
			if recResp.Success && len(recResp.Result) > 0 {
				return zone.ID, recResp.Result[0].Name, nil
			}
		}

		if len(zonesResp.Result) < 50 {
			break
		}
		page++
	}
	return "", "", fmt.Errorf("record not found: %s", recordID)
}

func (p *cloudflareProvider) request(method, urlStr string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(method, urlStr, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(respBody, result)
}

func cfErrMsgs(errs []cfMsg) string {
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Message
	}
	return joinStrings(msgs, "; ")
}

func buildFQDN(rr, domainName string) string {
	if rr == "@" || rr == "" {
		return domainName
	}
	return rr + "." + domainName
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += sep + s
	}
	return result
}
