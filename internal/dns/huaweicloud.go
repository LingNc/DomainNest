package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const huaweiCloudDefaultEndpoint = "https://dns.myhuaweicloud.com"

type huaweiCloudProvider struct {
	accessKey  string
	secretKey  string
	endpoint   string
	httpClient *http.Client
}

// HuaweiCloud API response types

type hwZone struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	RecordNum int64  `json:"record_num_num"`
}

type hwZonesResp struct {
	Zones []hwZone `json:"zones"`
}

type hwRecordset struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	ZoneID  string   `json:"zone_id"`
	Status  string   `json:"status"`
	Type    string   `json:"type"`
	TTL     int64    `json:"ttl"`
	Records []string `json:"records"`
}

type hwRecordsetsResp struct {
	Recordsets []hwRecordset `json:"recordsets"`
}

type hwCreateRecordsetReq struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Records []string `json:"records"`
	TTL     int64    `json:"ttl"`
}

func init() {
	Register("huaweicloud", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		if accessKeyID == "" || accessKeySecret == "" {
			return nil, fmt.Errorf("huaweicloud: Access Key and Secret Key are required")
		}
		ep := endpoint
		if ep == "" {
			ep = huaweiCloudDefaultEndpoint
		}
		return &huaweiCloudProvider{
			accessKey:  accessKeyID,
			secretKey:  accessKeySecret,
			endpoint:   ep,
			httpClient: &http.Client{},
		}, nil
	})
}

func (p *huaweiCloudProvider) GetType() string { return "huaweicloud" }

func (p *huaweiCloudProvider) ListDomains() ([]Domain, error) {
	var allZones []hwZone
	page := 1
	for {
		params := url.Values{}
		params.Set("page", fmt.Sprintf("%d", page))
		params.Set("limit", "100")

		var resp hwZonesResp
		if err := p.request("GET", p.endpoint+"/v2/zones", params, nil, &resp); err != nil {
			return nil, fmt.Errorf("huaweicloud: list domains: %w", err)
		}

		allZones = append(allZones, resp.Zones...)
		if len(resp.Zones) < 100 {
			break
		}
		page++
	}

	domains := make([]Domain, 0, len(allZones))
	for _, z := range allZones {
		name := strings.TrimSuffix(z.Name, ".")
		domains = append(domains, Domain{
			DomainName:  name,
			RecordCount: z.RecordNum,
		})
	}
	return domains, nil
}

func (p *huaweiCloudProvider) ListRecords(domainName string) ([]Record, error) {
	zoneID, err := p.getZoneID(domainName)
	if err != nil {
		return nil, err
	}

	var allRecords []Record
	page := 1
	for {
		params := url.Values{}
		params.Set("page", fmt.Sprintf("%d", page))
		params.Set("limit", "100")

		var resp hwRecordsetsResp
		if err := p.request("GET", fmt.Sprintf("%s/v2/zones/%s/recordsets", p.endpoint, zoneID), params, nil, &resp); err != nil {
			return nil, fmt.Errorf("huaweicloud: list records: %w", err)
		}

		for _, rs := range resp.Recordsets {
			name := strings.TrimSuffix(rs.Name, ".")
			host := name
			suffix := "." + domainName
			if strings.HasSuffix(name, suffix) {
				host = strings.TrimSuffix(name, suffix)
			} else if name == domainName {
				host = "@"
			}

			value := ""
			if len(rs.Records) > 0 {
				value = strings.Join(rs.Records, ",")
			}

			allRecords = append(allRecords, Record{
				RecordID: rs.ID,
				Host:     host,
				Type:     rs.Type,
				Value:    value,
				TTL:      rs.TTL,
			})
		}

		if len(resp.Recordsets) < 100 {
			break
		}
		page++
	}
	return allRecords, nil
}

func (p *huaweiCloudProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	zoneID, err := p.getZoneID(domainName)
	if err != nil {
		return "", err
	}

	name := buildFQDN(rr, domainName) + "."
	body := hwCreateRecordsetReq{
		Name:    name,
		Type:    recordType,
		Records: []string{value},
		TTL:     ttl,
	}

	var resp hwRecordset
	if err := p.request("POST", fmt.Sprintf("%s/v2/zones/%s/recordsets", p.endpoint, zoneID), nil, body, &resp); err != nil {
		return "", fmt.Errorf("huaweicloud: add record: %w", err)
	}

	return resp.ID, nil
}

func (p *huaweiCloudProvider) UpdateRecord(recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	zoneID, err := p.findRecordZone(recordID)
	if err != nil {
		return fmt.Errorf("huaweicloud: update record: %w", err)
	}

	body := map[string]interface{}{
		"records": []string{value},
		"ttl":     ttl,
	}

	var resp hwRecordset
	if err := p.request("PUT", fmt.Sprintf("%s/v2/zones/%s/recordsets/%s", p.endpoint, zoneID, recordID), nil, body, &resp); err != nil {
		return fmt.Errorf("huaweicloud: update record: %w", err)
	}
	return nil
}

func (p *huaweiCloudProvider) DeleteRecord(recordID string) error {
	zoneID, err := p.findRecordZone(recordID)
	if err != nil {
		return fmt.Errorf("huaweicloud: delete record: %w", err)
	}

	if err := p.request("DELETE", fmt.Sprintf("%s/v2/zones/%s/recordsets/%s", p.endpoint, zoneID, recordID), nil, nil, nil); err != nil {
		return fmt.Errorf("huaweicloud: delete record: %w", err)
	}
	return nil
}

func (p *huaweiCloudProvider) getZoneID(domainName string) (string, error) {
	params := url.Values{}
	params.Set("name", domainName)

	var resp hwZonesResp
	if err := p.request("GET", p.endpoint+"/v2/zones", params, nil, &resp); err != nil {
		return "", fmt.Errorf("get zone: %w", err)
	}
	if len(resp.Zones) == 0 {
		return "", fmt.Errorf("zone not found for domain: %s", domainName)
	}

	for _, z := range resp.Zones {
		if strings.TrimSuffix(z.Name, ".") == domainName {
			return z.ID, nil
		}
	}
	return resp.Zones[0].ID, nil
}

func (p *huaweiCloudProvider) findRecordZone(recordID string) (string, error) {
	page := 1
	for {
		params := url.Values{}
		params.Set("page", fmt.Sprintf("%d", page))
		params.Set("limit", "100")

		var zonesResp hwZonesResp
		if err := p.request("GET", p.endpoint+"/v2/zones", params, nil, &zonesResp); err != nil {
			return "", err
		}

		for _, zone := range zonesResp.Zones {
			var rs hwRecordset
			if err := p.request("GET", fmt.Sprintf("%s/v2/zones/%s/recordsets/%s", p.endpoint, zone.ID, recordID), nil, nil, &rs); err != nil {
				continue
			}
			if rs.ID == recordID {
				return zone.ID, nil
			}
		}

		if len(zonesResp.Zones) < 100 {
			break
		}
		page++
	}
	return "", fmt.Errorf("record not found in any zone: %s", recordID)
}

func (p *huaweiCloudProvider) request(method, urlStr string, queryParams url.Values, body interface{}, result interface{}) error {
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

	if queryParams != nil {
		req.URL.RawQuery = queryParams.Encode()
	}

	// Sign using HuaweiCloud SDK-HMAC-SHA256
	hwSignRequest(p.accessKey, p.secretKey, req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result == nil {
		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		return nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return json.Unmarshal(respBody, result)
}

// hwSignRequest implements HuaweiCloud SDK-HMAC-SHA256 signing.
// Based on AWS Signature V4 pattern used by Huawei Cloud.
func hwSignRequest(accessKey, secretKey string, r *http.Request) {
	t := time.Now().UTC()
	r.Header.Set("X-Sdk-Date", t.Format("20060102T150405Z"))

	signedHeaders := hwSignedHeaders(r)
	canonicalReq := hwBuildCanonicalRequest(r, signedHeaders)
	stringToSign := hwBuildStringToSign(canonicalReq, t)
	signature := hwComputeSignature(secretKey, stringToSign)

	authValue := fmt.Sprintf("SDK-HMAC-SHA256 Access=%s, SignedHeaders=%s, Signature=%s",
		accessKey, joinStrings(signedHeaders, ";"), signature)
	r.Header.Set("Authorization", authValue)
}

func hwSignedHeaders(r *http.Request) []string {
	var headers []string
	for key := range r.Header {
		headers = append(headers, strings.ToLower(key))
	}
	// Always include host
	hasHost := false
	for _, h := range headers {
		if h == "host" {
			hasHost = true
			break
		}
	}
	if !hasHost {
		headers = append(headers, "host")
	}
	sort.Strings(headers)
	return headers
}

func hwBuildCanonicalRequest(r *http.Request, signedHeaders []string) string {
	var payload string
	if r.Body != nil {
		bodyBytes, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		payload = hashSHA256Hex(bodyBytes)
	} else {
		payload = hashSHA256Hex([]byte(""))
	}

	uri := r.URL.Path
	if uri == "" {
		uri = "/"
	}
	if uri[len(uri)-1] != '/' {
		uri += "/"
	}

	query := hwCanonicalQueryString(r)
	headers := hwCanonicalHeaders(r, signedHeaders)

	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		r.Method, uri, query, headers, joinStrings(signedHeaders, ";"), payload)
}

func hwCanonicalQueryString(r *http.Request) string {
	query := r.URL.Query()
	if len(query) == 0 {
		return ""
	}

	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var parts []string
	for _, key := range keys {
		values := query[key]
		sort.Strings(values)
		for _, v := range values {
			parts = append(parts, percentEncode(key)+"="+percentEncode(v))
		}
	}
	return joinStrings(parts, "&")
}

func hwCanonicalHeaders(r *http.Request, signedHeaders []string) string {
	header := make(map[string][]string)
	for k, v := range r.Header {
		header[strings.ToLower(k)] = v
	}

	var parts []string
	for _, key := range signedHeaders {
		values := header[key]
		if key == "host" {
			values = []string{r.Host}
		}
		sort.Strings(values)
		for _, v := range values {
			parts = append(parts, key+":"+strings.TrimSpace(v))
		}
	}
	return joinStrings(parts, "\n") + "\n"
}

func hwBuildStringToSign(canonicalRequest string, t time.Time) string {
	hash := sha256HexBytes([]byte(canonicalRequest))
	return fmt.Sprintf("SDK-HMAC-SHA256\n%s\n%s", t.Format("20060102T150405Z"), hash)
}

func hwComputeSignature(secretKey, stringToSign string) string {
	hm := hmacSHA256Bytes([]byte(secretKey), stringToSign)
	return fmt.Sprintf("%x", hm)
}

func sha256HexBytes(data []byte) string {
	return hashSHA256Hex(data)
}
