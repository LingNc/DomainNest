package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

const dynadotEndpoint = "https://www.dynadot.com/set_ddns"

// DynadotProvider implements the Provider interface for Dynadot DDNS.
// Note: Dynadot has a limited DDNS API that supports updating records
// but not full CRUD. This implementation adapts the available API.
type DynadotProvider struct {
	password   string
	httpClient *http.Client
}

type dynadotResp struct {
	Status    string   `json:"status"`
	ErrorCode int      `json:"error_code"`
	Content   []string `json:"content"`
}

func init() {
	Register("dynadot", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &DynadotProvider{
			password:   accessKeySecret,
			httpClient: &http.Client{},
		}, nil
	})
}

func (p *DynadotProvider) GetType() string { return "dynadot" }

func (p *DynadotProvider) ListDomains() ([]Domain, error) {
	// Dynadot DDNS API does not support listing domains.
	return nil, fmt.Errorf("dynadot: ListDomains not supported; specify domains explicitly")
}

func (p *DynadotProvider) ListRecords(domainName string) ([]Record, error) {
	// Dynadot DDNS API does not support listing records.
	return nil, fmt.Errorf("dynadot: ListRecords not supported; Dynadot uses a simplified DDNS API")
}

func (p *DynadotProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	params := map[string]string{
		"domain":       domainName,
		"subDomain":    rr,
		"type":         recordType,
		"ip":           value,
		"pwd":          p.password,
		"ttl":          strconv.FormatInt(ttl, 10),
		"containRoot":  "false",
	}
	if rr == "@" || rr == "" {
		params["containRoot"] = "true"
		params["subDomain"] = ""
	}

	var result dynadotResp
	err := p.request(params, &result)
	if err != nil {
		return "", err
	}
	if result.ErrorCode == -1 {
		return "", fmt.Errorf("dynadot: %s", result.Content)
	}
	return domainName + "/" + rr, nil
}

func (p *DynadotProvider) UpdateRecord(domainName, recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	// For Dynadot, update is the same as add (it's a DDNS set operation).
	_, err := p.AddRecord(recordID, rr, recordType, value, ttl, priority)
	return err
}

func (p *DynadotProvider) DeleteRecord(recordID string) error {
	// Dynadot DDNS API does not support deleting records.
	return fmt.Errorf("dynadot: DeleteRecord not supported")
}

func (p *DynadotProvider) request(params map[string]string, result interface{}) error {
	values := make(map[string][]string)
	for k, v := range params {
		values[k] = []string{v}
	}

	req, err := http.NewRequest("GET", dynadotEndpoint, nil)
	if err != nil {
		return fmt.Errorf("dynadot: create request: %w", err)
	}
	q := req.URL.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("dynadot: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("dynadot: read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("dynadot: API error status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("dynadot: decode response: %w", err)
		}
	}

	return nil
}

// Suppress unused import
var _ = bytes.NewReader

var _ Provider = (*DynadotProvider)(nil)
