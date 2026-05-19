package dns

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// NamecheapProvider implements the Provider interface for Namecheap DNS.
type NamecheapProvider struct {
	apiUser  string
	apiKey   string
	endpoint string
	client   *http.Client
}

// --- XML response types ---

type namecheapResponse struct {
	XMLName xml.Name           `xml:"ApiResponse"`
	Status  string             `xml:"Status,attr"`
	Errors  []namecheapError   `xml:"Errors>Error"`
	Result  namecheapResult    `xml:"CommandResponse"`
}

type namecheapError struct {
	Number  string `xml:"Number,attr"`
	Message string `xml:",chardata"`
}

type namecheapResult struct {
	// For domains.getList
	DomainList namecheapDomainList `xml:"DomainGetListResult"`
	// For domains.dns.getHosts
	HostsList  namecheapHostsResult `xml:"DomainDnsGetHostsResult"`
	// For domains.dns.setHosts
	SetHostsResult namecheapSetHostsResult `xml:"DomainDnsSetHostsResult"`
}

type namecheapDomainList struct {
	Domains []namecheapDomain `xml:"Domain"`
}

type namecheapDomain struct {
	ID         int64  `xml:"ID,attr"`
	Name       string `xml:"Name,attr"`
	User       string `xml:"User,attr"`
	Created    string `xml:"Created,attr"`
	Expires    string `xml:"Expires,attr"`
	IsExpired  bool   `xml:"IsExpired,attr"`
	IsLocked   bool   `xml:"IsLocked,attr"`
	AutoRenew  bool   `xml:"AutoRenew,attr"`
	WhoisGuard string `xml:"WhoisGuard,attr"`
}

type namecheapHostsResult struct {
	Hosts []namecheapHost `xml:"host"`
}

type namecheapHost struct {
	HostID   string `xml:"HostId,attr"`
	Name     string `xml:"Name,attr"`
	Type     string `xml:"Type,attr"`
	Address  string `xml:"Address,attr"`
	TTL      string `xml:"TTL,attr"`
	MXPref   string `xml:"MXPref,attr"`
	IsActive string `xml:"IsActive,attr"`
	IsDDNSEnabled string `xml:"IsDDNSEnabled,attr"`
}

type namecheapSetHostsResult struct {
	IsSuccess bool `xml:"IsSuccess,attr"`
}

func init() {
	Register("namecheap", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		ep := endpoint
		if ep == "" {
			ep = "api.namecheap.com"
		}
		return &NamecheapProvider{
			apiUser:  accessKeyID,
			apiKey:   accessKeySecret,
			endpoint: ep,
			client:   &http.Client{},
		}, nil
	})
}

func (p *NamecheapProvider) GetType() string { return "namecheap" }

// buildURL constructs a Namecheap API URL with common params.
func (p *NamecheapProvider) buildURL(command string, extra map[string]string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("https://%s/xml.response?ApiUser=%s&ApiKey=%s&UserName=%s&Command=%s",
		p.endpoint, p.apiUser, p.apiKey, p.apiUser, command))
	for k, v := range extra {
		sb.WriteString(fmt.Sprintf("&%s=%s", k, v))
	}
	return sb.String()
}

func (p *NamecheapProvider) doRequest(url string) (*namecheapResponse, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp namecheapResponse
	if err := xml.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("namecheap: failed to parse XML: %w (body: %s)", err, string(body))
	}
	if apiResp.Status != "OK" {
		msgs := make([]string, 0, len(apiResp.Errors))
		for _, e := range apiResp.Errors {
			msgs = append(msgs, e.Message)
		}
		return nil, fmt.Errorf("namecheap API error: %s", strings.Join(msgs, "; "))
	}
	return &apiResp, nil
}

// ListDomains returns all domains from Namecheap.
func (p *NamecheapProvider) ListDomains() ([]Domain, error) {
	var allDomains []Domain
	page := 1
	perPage := 100

	for {
		url := p.buildURL("namecheap.domains.getList", map[string]string{
			"ListType": "ALL",
			"Page":     strconv.Itoa(page),
			"PageSize": strconv.Itoa(perPage),
		})

		resp, err := p.doRequest(url)
		if err != nil {
			return nil, err
		}

		for _, d := range resp.Result.DomainList.Domains {
			allDomains = append(allDomains, Domain{
				DomainName:  d.Name,
				RecordCount: 0, // Namecheap doesn't provide this in getList
			})
		}

		if len(resp.Result.DomainList.Domains) < perPage {
			break
		}
		page++
	}
	return allDomains, nil
}

// ListRecords returns all DNS records for a domain from Namecheap.
func (p *NamecheapProvider) ListRecords(domainName string) ([]Record, error) {
	sld, tld := splitDomain(domainName)
	url := p.buildURL("namecheap.domains.dns.getHosts", map[string]string{
		"SLD": sld,
		"TLD": tld,
	})

	resp, err := p.doRequest(url)
	if err != nil {
		return nil, err
	}

	records := make([]Record, 0, len(resp.Result.HostsList.Hosts))
	for _, h := range resp.Result.HostsList.Hosts {
		ttl, _ := strconv.ParseInt(h.TTL, 10, 64)
		var priority *int64
		if h.Type == "MX" {
			if p, err := strconv.ParseInt(h.MXPref, 10, 64); err == nil {
				priority = &p
			}
		}
		records = append(records, Record{
			RecordID: fmt.Sprintf("%s|%s|%s", domainName, h.HostID, h.Name),
			Host:     h.Name,
			Type:     h.Type,
			Value:    h.Address,
			TTL:      ttl,
			Priority: priority,
		})
	}
	return records, nil
}

// AddRecord creates a new DNS record in Namecheap.
// Namecheap uses setHosts which sets ALL records at once.
func (p *NamecheapProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	existing, err := p.ListRecords(domainName)
	if err != nil {
		return "", err
	}

	// Build the full record list including the new one.
	params := p.buildHostsParams(domainName, existing)
	nextIdx := len(existing) + 1
	params[fmt.Sprintf("HostName%d", nextIdx)] = rr
	params[fmt.Sprintf("RecordType%d", nextIdx)] = recordType
	params[fmt.Sprintf("Address%d", nextIdx)] = value
	params[fmt.Sprintf("TTL%d", nextIdx)] = strconv.FormatInt(ttl, 10)
	if priority != nil {
		params[fmt.Sprintf("MXPref%d", nextIdx)] = strconv.FormatInt(*priority, 10)
	} else {
		params[fmt.Sprintf("MXPref%d", nextIdx)] = "10"
	}

	if err := p.setHosts(domainName, params); err != nil {
		return "", err
	}
	// Namecheap doesn't return a record ID on setHosts. Return a synthetic one.
	return fmt.Sprintf("%s|%s", domainName, rr), nil
}

// UpdateRecord updates an existing DNS record in Namecheap.
// recordID format: "domain|hostID|name"
func (p *NamecheapProvider) UpdateRecord(recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	domain, hostID, _, err := parseNamecheapRecordID(recordID)
	if err != nil {
		return err
	}

	existing, err := p.ListRecords(domain)
	if err != nil {
		return err
	}

	// Rebuild all records, replacing the one matching hostID.
	params := map[string]string{}
	idx := 0
	for _, r := range existing {
		_, existingID, _ := parseNamecheapRecordIDUnsafe(r.RecordID)
		if existingID == hostID {
			// Replace with updated values.
			idx++
			params[fmt.Sprintf("HostName%d", idx)] = rr
			params[fmt.Sprintf("RecordType%d", idx)] = recordType
			params[fmt.Sprintf("Address%d", idx)] = value
			params[fmt.Sprintf("TTL%d", idx)] = strconv.FormatInt(ttl, 10)
			if priority != nil {
				params[fmt.Sprintf("MXPref%d", idx)] = strconv.FormatInt(*priority, 10)
			} else {
				params[fmt.Sprintf("MXPref%d", idx)] = "10"
			}
		} else {
			idx++
			params[fmt.Sprintf("HostName%d", idx)] = r.Host
			params[fmt.Sprintf("RecordType%d", idx)] = r.Type
			params[fmt.Sprintf("Address%d", idx)] = r.Value
			params[fmt.Sprintf("TTL%d", idx)] = strconv.FormatInt(r.TTL, 10)
			if r.Priority != nil {
				params[fmt.Sprintf("MXPref%d", idx)] = strconv.FormatInt(*r.Priority, 10)
			} else {
				params[fmt.Sprintf("MXPref%d", idx)] = "10"
			}
		}
	}

	return p.setHosts(domain, params)
}

// DeleteRecord deletes a DNS record from Namecheap.
// recordID format: "domain|hostID|name"
func (p *NamecheapProvider) DeleteRecord(recordID string) error {
	domain, hostID, _, err := parseNamecheapRecordID(recordID)
	if err != nil {
		return err
	}

	existing, err := p.ListRecords(domain)
	if err != nil {
		return err
	}

	// Rebuild all records except the one matching hostID.
	params := p.buildHostsParamsExcluding(domain, existing, hostID)
	return p.setHosts(domain, params)
}

// --- internal helpers ---

func (p *NamecheapProvider) setHosts(domainName string, params map[string]string) error {
	sld, tld := splitDomain(domainName)
	params["SLD"] = sld
	params["TLD"] = tld

	url := p.buildURL("namecheap.domains.dns.setHosts", params)
	resp, err := p.doRequest(url)
	if err != nil {
		return err
	}
	if !resp.Result.SetHostsResult.IsSuccess {
		return fmt.Errorf("namecheap setHosts failed for %s", domainName)
	}
	return nil
}

// buildHostsParams builds the numbered params for setHosts from existing records.
func (p *NamecheapProvider) buildHostsParams(domainName string, existing []Record) map[string]string {
	params := map[string]string{}
	for i, r := range existing {
		idx := i + 1
		params[fmt.Sprintf("HostName%d", idx)] = r.Host
		params[fmt.Sprintf("RecordType%d", idx)] = r.Type
		params[fmt.Sprintf("Address%d", idx)] = r.Value
		params[fmt.Sprintf("TTL%d", idx)] = strconv.FormatInt(r.TTL, 10)
		if r.Priority != nil {
			params[fmt.Sprintf("MXPref%d", idx)] = strconv.FormatInt(*r.Priority, 10)
		} else {
			params[fmt.Sprintf("MXPref%d", idx)] = "10"
		}
	}
	return params
}

// buildHostsParamsExcluding builds setHosts params excluding a record by hostID.
func (p *NamecheapProvider) buildHostsParamsExcluding(domainName string, existing []Record, excludeHostID string) map[string]string {
	params := map[string]string{}
	idx := 0
	for _, r := range existing {
		_, existingID, _ := parseNamecheapRecordIDUnsafe(r.RecordID)
		if existingID == excludeHostID {
			continue
		}
		idx++
		params[fmt.Sprintf("HostName%d", idx)] = r.Host
		params[fmt.Sprintf("RecordType%d", idx)] = r.Type
		params[fmt.Sprintf("Address%d", idx)] = r.Value
		params[fmt.Sprintf("TTL%d", idx)] = strconv.FormatInt(r.TTL, 10)
		if r.Priority != nil {
			params[fmt.Sprintf("MXPref%d", idx)] = strconv.FormatInt(*r.Priority, 10)
		} else {
			params[fmt.Sprintf("MXPref%d", idx)] = "10"
		}
	}
	return params
}

// splitDomain splits "example.com" into ("example", "com").
func splitDomain(domain string) (string, string) {
	parts := strings.SplitN(domain, ".", 2)
	if len(parts) != 2 {
		return domain, ""
	}
	return parts[0], parts[1]
}

func parseNamecheapRecordID(recordID string) (domain, hostID, name string, err error) {
	parts := strings.SplitN(recordID, "|", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid namecheap recordID: %s", recordID)
	}
	return parts[0], parts[1], parts[2], nil
}

func parseNamecheapRecordIDUnsafe(recordID string) (domain, hostID, name string) {
	parts := strings.SplitN(recordID, "|", 3)
	if len(parts) != 3 {
		return "", "", ""
	}
	return parts[0], parts[1], parts[2]
}
