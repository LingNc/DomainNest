package dns

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const eranetAPIBase = "https://www.eranet.com"

// EranetProvider implements the Provider interface for Eranet DNS.
type EranetProvider struct {
	accessKeyID string
	secretKey   string
	httpClient  *http.Client
}

type eranetRecord struct {
	ID     int    `json:"id"`
	Domain string `json:"domain"`
	Host   string `json:"host"`
	Type   string `json:"type"`
	Value  string `json:"value"`
	State  string `json:"state"`
}

type eranetRecordListResp struct {
	RequestId string `json:"RequestId"`
	Error     string `json:"error"`
	Data      []eranetRecord `json:"Data"`
}

type eranetBaseResult struct {
	RequestId string `json:"RequestId"`
	Id        int    `json:"Id"`
	Error     string `json:"error"`
}

func init() {
	Register("eranet", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &EranetProvider{
			accessKeyID: accessKeyID,
			secretKey:   accessKeySecret,
			httpClient:  &http.Client{},
		}, nil
	})
}

func (p *EranetProvider) GetType() string { return "eranet" }

func (p *EranetProvider) ListDomains() ([]Domain, error) {
	return nil, fmt.Errorf("eranet: ListDomains not supported; specify domains explicitly")
}

func (p *EranetProvider) ListRecords(domainName string) ([]Record, error) {
	params := map[string]string{
		"Domain": domainName,
	}
	res, err := p.request("/api/Dns/DescribeRecordIndex", params, "GET")
	if err != nil {
		return nil, err
	}
	var result eranetRecordListResp
	if err := json.Unmarshal(res, &result); err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("eranet: %s", result.Error)
	}
	records := make([]Record, len(result.Data))
	for i, r := range result.Data {
		records[i] = Record{
			RecordID: strconv.Itoa(r.ID),
			Host:     r.Host,
			Type:     r.Type,
			Value:    r.Value,
			TTL:      600,
		}
	}
	return records, nil
}

func (p *EranetProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	params := map[string]string{
		"Domain": domainName,
		"Host":   rr,
		"Type":   recordType,
		"Value":  value,
		"Ttl":    strconv.FormatInt(ttl, 10),
	}
	res, err := p.request("/api/Dns/AddDomainRecord", params, "GET")
	if err != nil {
		return "", err
	}
	var result eranetBaseResult
	if err := json.Unmarshal(res, &result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", fmt.Errorf("eranet: %s", result.Error)
	}
	return strconv.Itoa(result.Id), nil
}

func (p *EranetProvider) UpdateRecord(domainName, recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	params := map[string]string{
		"Id":     recordID,
		"Domain": rr,
		"Host":   rr,
		"Type":   recordType,
		"Value":  value,
		"Ttl":    strconv.FormatInt(ttl, 10),
	}
	res, err := p.request("/api/Dns/UpdateDomainRecord", params, "GET")
	if err != nil {
		return err
	}
	var result eranetBaseResult
	if err := json.Unmarshal(res, &result); err != nil {
		return err
	}
	if result.Error != "" {
		return fmt.Errorf("eranet: %s", result.Error)
	}
	return nil
}

func (p *EranetProvider) DeleteRecord(recordID string) error {
	params := map[string]string{
		"Id": recordID,
	}
	res, err := p.request("/api/Dns/DeleteDomainRecord", params, "GET")
	if err != nil {
		return err
	}
	var result eranetBaseResult
	if err := json.Unmarshal(res, &result); err != nil {
		return err
	}
	if result.Error != "" {
		return fmt.Errorf("eranet: %s", result.Error)
	}
	return nil
}

func (p *EranetProvider) sign(params map[string]string, method string) string {
	params["AccessInstanceID"] = p.accessKeyID
	params["SignatureMethod"] = "HMAC-SHA1"
	params["SignatureNonce"] = fmt.Sprintf("%d", time.Now().UnixNano())
	params["Timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")

	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "Signature" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var canonicalizedQuery []string
	for _, k := range keys {
		canonicalizedQuery = append(canonicalizedQuery, percentEncode(k)+"="+percentEncode(params[k]))
	}
	canonicalizedQueryString := strings.Join(canonicalizedQuery, "&")

	stringToSign := method + "&" + percentEncode("/") + "&" + percentEncode(canonicalizedQueryString)

	key := p.secretKey + "&"
	h := hmac.New(sha1.New, []byte(key))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	params["Signature"] = signature

	keys = append(keys, "Signature")
	sort.Strings(keys)
	var finalQuery []string
	for _, k := range keys {
		finalQuery = append(finalQuery, percentEncode(k)+"="+percentEncode(params[k]))
	}
	return strings.Join(finalQuery, "&")
}

func (p *EranetProvider) request(apiPath string, params map[string]string, method string) ([]byte, error) {
	queryString := p.sign(params, method)
	fullURL := eranetAPIBase + apiPath + "?" + queryString

	req, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("eranet: create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eranet: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("eranet: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eranet: API error status %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

var _ Provider = (*EranetProvider)(nil)
