package dns

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// TrafficRouteProvider implements the Provider interface for Volcengine TrafficRoute DNS.
type TrafficRouteProvider struct {
	accessKeyID string
	secretKey   string
	httpClient  *http.Client
}

type trafficRouteRecord struct {
	RecordID string `json:"RecordId"`
	Host     string `json:"Host"`
	Type     string `json:"Type"`
	Value    string `json:"Value"`
	TTL      int    `json:"TTL"`
	Line     string `json:"Line"`
	ZID      int    `json:"ZID"`
}

func init() {
	Register("trafficroute", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &TrafficRouteProvider{
			accessKeyID: accessKeyID,
			secretKey:   accessKeySecret,
			httpClient:  &http.Client{},
		}, nil
	})
}

func (p *TrafficRouteProvider) GetType() string { return "trafficroute" }

func (p *TrafficRouteProvider) ListDomains() ([]Domain, error) {
	req, err := trafficRouteSigner("GET", nil, nil, p.accessKeyID, p.secretKey, "ListZones", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Result struct {
			Zones []struct {
				ZoneName string `json:"ZoneName"`
				ZID      int    `json:"ZID"`
			} `json:"Zones"`
		} `json:"Result"`
	}
	err = p.doRequest(req, &result)
	if err != nil {
		return nil, err
	}
	domains := make([]Domain, len(result.Result.Zones))
	for i, z := range result.Result.Zones {
		domains[i] = Domain{DomainName: z.ZoneName}
	}
	return domains, nil
}

func (p *TrafficRouteProvider) ListRecords(domainName string) ([]Record, error) {
	zid, err := p.getZID(domainName)
	if err != nil {
		return nil, err
	}

	query := map[string][]string{
		"ZID": {strconv.Itoa(zid)},
	}
	req, err := trafficRouteSigner("GET", query, nil, p.accessKeyID, p.secretKey, "ListRecords", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result struct {
			Records []trafficRouteRecord `json:"Records"`
		} `json:"Result"`
	}
	err = p.doRequest(req, &result)
	if err != nil {
		return nil, err
	}

	records := make([]Record, len(result.Result.Records))
	for i, r := range result.Result.Records {
		records[i] = Record{
			RecordID: r.RecordID,
			Host:     r.Host,
			Type:     r.Type,
			Value:    r.Value,
			TTL:      int64(r.TTL),
		}
	}
	return records, nil
}

func (p *TrafficRouteProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	zid, err := p.getZID(domainName)
	if err != nil {
		return "", err
	}

	payload := map[string]interface{}{
		"ZID":   zid,
		"Host":  rr,
		"Type":  recordType,
		"Value": value,
		"TTL":   int(ttl),
		"Line":  "default",
	}
	body, _ := json.Marshal(payload)
	req, err := trafficRouteSigner("POST", nil, nil, p.accessKeyID, p.secretKey, "CreateRecord", body)
	if err != nil {
		return "", err
	}

	var result struct {
		Result struct {
			RecordID string `json:"RecordId"`
		} `json:"Result"`
	}
	err = p.doRequest(req, &result)
	if err != nil {
		return "", err
	}
	return result.Result.RecordID, nil
}

func (p *TrafficRouteProvider) UpdateRecord(recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	payload := map[string]interface{}{
		"RecordId": recordID,
		"Host":     rr,
		"Type":     recordType,
		"Value":    value,
		"TTL":      int(ttl),
		"Line":     "default",
	}
	body, _ := json.Marshal(payload)
	req, err := trafficRouteSigner("POST", nil, nil, p.accessKeyID, p.secretKey, "UpdateRecord", body)
	if err != nil {
		return err
	}
	return p.doRequest(req, nil)
}

func (p *TrafficRouteProvider) DeleteRecord(recordID string) error {
	payload := map[string]interface{}{
		"RecordId": recordID,
	}
	body, _ := json.Marshal(payload)
	req, err := trafficRouteSigner("POST", nil, nil, p.accessKeyID, p.secretKey, "DeleteRecord", body)
	if err != nil {
		return err
	}
	return p.doRequest(req, nil)
}

func (p *TrafficRouteProvider) getZID(domainName string) (int, error) {
	query := map[string][]string{
		"Key": {domainName},
	}
	req, err := trafficRouteSigner("GET", query, nil, p.accessKeyID, p.secretKey, "ListZones", nil)
	if err != nil {
		return 0, err
	}

	var result struct {
		Result struct {
			Zones []struct {
				ZoneName string `json:"ZoneName"`
				ZID      int    `json:"ZID"`
			} `json:"Zones"`
		} `json:"Result"`
	}
	err = p.doRequest(req, &result)
	if err != nil {
		return 0, err
	}
	for _, z := range result.Result.Zones {
		if z.ZoneName == domainName {
			return z.ZID, nil
		}
	}
	return 0, fmt.Errorf("trafficroute: zone not found: %s", domainName)
}

func (p *TrafficRouteProvider) doRequest(req *http.Request, result interface{}) error {
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("trafficroute: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("trafficroute: API error status %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("trafficroute: decode response: %w", err)
		}
	}
	return nil
}

var _ Provider = (*TrafficRouteProvider)(nil)
