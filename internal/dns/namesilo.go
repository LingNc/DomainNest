package dns

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

const namesiloBaseURL = "https://www.namesilo.com/api"

// --- XML response types ---

type namesiloResponse struct {
	XMLName xml.Name         `xml:"namesilo"`
	Request namesiloRequest  `xml:"request"`
	Reply   namesiloReply    `xml:"reply"`
}

type namesiloRequest struct {
	Operation string `xml:"operation"`
	IP        string `xml:"ip"`
}

type namesiloReply struct {
	Code          int                  `xml:"code"`
	Detail        string               `xml:"detail"`
	RecordID      string               `xml:"record_id"`
	ResourceItems []namesiloResourceRecord `xml:"resource_record"`
	Domains       []namesiloDomain     `xml:"domain"`
}

type namesiloResourceRecord struct {
	RecordID string `xml:"record_id"`
	Type     string `xml:"type"`
	Host     string `xml:"host"`
	Value    string `xml:"value"`
	TTL      int64  `xml:"ttl"`
	Distance int    `xml:"distance"`
}

type namesiloDomain struct {
	Name string `xml:",chardata"`
}

// NameSiloProvider implements the Provider interface for NameSilo DNS.
type NameSiloProvider struct {
	apiKey string
	client *http.Client
}

func init() {
	Register("namesilo", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &NameSiloProvider{
			apiKey: accessKeyID,
			client: &http.Client{},
		}, nil
	})
}

func (p *NameSiloProvider) GetType() string { return "namesilo" }

// doRequest performs a GET request and parses the XML response.
func (p *NameSiloProvider) doRequest(url string) (*namesiloResponse, error) {
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

	var apiResp namesiloResponse
	if err := xml.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("namesilo: failed to parse XML: %w (body: %s)", err, string(body))
	}
	// NameSilo returns code 300 for success.
	if apiResp.Reply.Code != 300 {
		return nil, fmt.Errorf("namesilo API error (code %d): %s", apiResp.Reply.Code, apiResp.Reply.Detail)
	}
	return &apiResp, nil
}

// ListDomains returns all domains from NameSilo.
func (p *NameSiloProvider) ListDomains() ([]Domain, error) {
	url := fmt.Sprintf("%s/listDomains?version=1&type=xml&key=%s", namesiloBaseURL, p.apiKey)
	resp, err := p.doRequest(url)
	if err != nil {
		return nil, err
	}

	domains := make([]Domain, len(resp.Reply.Domains))
	for i, d := range resp.Reply.Domains {
		domains[i] = Domain{DomainName: d.Name}
	}
	return domains, nil
}

// ListRecords returns all DNS records for a domain from NameSilo.
func (p *NameSiloProvider) ListRecords(domainName string) ([]Record, error) {
	url := fmt.Sprintf("%s/dnsListRecords?version=1&type=xml&key=%s&domain=%s",
		namesiloBaseURL, p.apiKey, domainName)
	resp, err := p.doRequest(url)
	if err != nil {
		return nil, err
	}

	records := make([]Record, 0, len(resp.Reply.ResourceItems))
	for _, r := range resp.Reply.ResourceItems {
		host := r.Host
		// NameSilo returns the full hostname (e.g. "sub.example.com").
		// Strip the domain suffix to get just the subdomain.
		if host == domainName {
			host = "@"
		} else if suffix := "." + domainName; len(host) > len(suffix) && host[len(host)-len(suffix):] == suffix {
			host = host[:len(host)-len(suffix)]
		}

		var priority *int64
		if r.Distance > 0 {
			p := int64(r.Distance)
			priority = &p
		}

		records = append(records, Record{
			RecordID: fmt.Sprintf("%s|%s", domainName, r.RecordID),
			Host:     host,
			Type:     r.Type,
			Value:    r.Value,
			TTL:      r.TTL,
			Priority: priority,
		})
	}
	return records, nil
}

// AddRecord creates a new DNS record in NameSilo.
func (p *NameSiloProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	host := rr
	if host == "@" {
		host = ""
	}

	url := fmt.Sprintf("%s/dnsAddRecord?version=1&type=xml&key=%s&domain=%s&rrhost=%s&rrtype=%s&rrvalue=%s&rrttl=%d",
		namesiloBaseURL, p.apiKey, domainName, host, recordType, value, ttl)
	if priority != nil {
		url += fmt.Sprintf("&rrdistance=%d", *priority)
	}

	resp, err := p.doRequest(url)
	if err != nil {
		return "", err
	}

	// NameSilo returns the new record_id in the reply.
	recordID := resp.Reply.RecordID
	if recordID == "" {
		return "", fmt.Errorf("namesilo AddRecord: no record_id returned")
	}
	return fmt.Sprintf("%s|%s", domainName, recordID), nil
}

// UpdateRecord updates an existing DNS record in NameSilo.
// recordID format: "domain|namesiloRecordID"
func (p *NameSiloProvider) UpdateRecord(recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	domain, nsID, err := parseNameSiloRecordID(recordID)
	if err != nil {
		return err
	}

	host := rr
	if host == "@" {
		host = ""
	}

	url := fmt.Sprintf("%s/dnsUpdateRecord?version=1&type=xml&key=%s&domain=%s&rrid=%s&rrhost=%s&rrvalue=%s&rrttl=%d",
		namesiloBaseURL, p.apiKey, domain, nsID, host, value, ttl)
	if priority != nil {
		url += fmt.Sprintf("&rrdistance=%d", *priority)
	}

	_, err = p.doRequest(url)
	return err
}

// DeleteRecord deletes a DNS record from NameSilo.
// recordID format: "domain|namesiloRecordID"
func (p *NameSiloProvider) DeleteRecord(recordID string) error {
	domain, nsID, err := parseNameSiloRecordID(recordID)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/dnsDeleteRecord?version=1&type=xml&key=%s&domain=%s&rrid=%s",
		namesiloBaseURL, p.apiKey, domain, nsID)

	_, err = p.doRequest(url)
	return err
}

func parseNameSiloRecordID(recordID string) (domain, nsID string, err error) {
	parts := splitN(recordID, "|", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid namesilo recordID: %s", recordID)
	}
	return parts[0], parts[1], nil
}

