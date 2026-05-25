package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

const porkbunBaseURL = "https://porkbun.com/api/json/v3"

// porkbunAuth is the authentication payload sent with every Porkbun request.
type porkbunAuth struct {
	APIKey       string `json:"apikey"`
	SecretAPIKey string `json:"secretapikey"`
}

// porkbunRecord represents a Porkbun DNS record.
type porkbunRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     string `json:"ttl"`
	Prio    string `json:"prio"`
	Notes   string `json:"notes"`
}

// porkbunResponse is the base response from Porkbun API.
type porkbunResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// porkbunDomainResponse is the response for domain listing.
type porkbunDomainResponse struct {
	porkbunResponse
	Domains []struct {
		Domain string `json:"domain"`
	} `json:"domains"`
}

// porkbunRecordsResponse is the response for DNS record retrieval.
type porkbunRecordsResponse struct {
	porkbunResponse
	Records []porkbunRecord `json:"records"`
}

// porkbunCreateBody is the request body for creating a record.
type porkbunCreateBody struct {
	porkbunAuth
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     string `json:"ttl"`
	Prio    string `json:"prio,omitempty"`
}

// porkbunEditBody is the request body for editing a record.
type porkbunEditBody struct {
	porkbunAuth
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     string `json:"ttl"`
	Prio    string `json:"prio,omitempty"`
}

// porkbunDeleteBody is the request body for deleting a record.
type porkbunDeleteBody struct {
	porkbunAuth
}

// PorkbunProvider implements the Provider interface for Porkbun DNS.
type PorkbunProvider struct {
	apiKey       string
	secretAPIKey string
	client       *http.Client
}

func init() {
	Register("porkbun", func(accessKeyID, accessKeySecret, endpoint string) (Provider, error) {
		return &PorkbunProvider{
			apiKey:       accessKeyID,
			secretAPIKey: accessKeySecret,
			client:       &http.Client{},
		}, nil
	})
}

func (p *PorkbunProvider) GetType() string { return "porkbun" }

func (p *PorkbunProvider) auth() porkbunAuth {
	return porkbunAuth{
		APIKey:       p.apiKey,
		SecretAPIKey: p.secretAPIKey,
	}
}

// doJSON performs a POST request with a JSON body and decodes the response into result.
func (p *PorkbunProvider) doJSON(url string, body interface{}, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("porkbun: failed to parse response: %w (body: %s)", err, string(respBody))
	}
	return nil
}

// ListDomains returns all domains from Porkbun.
func (p *PorkbunProvider) ListDomains() ([]Domain, error) {
	var resp porkbunDomainResponse
	if err := p.doJSON(porkbunBaseURL+"/domain/listAll", p.auth(), &resp); err != nil {
		return nil, err
	}
	if resp.Status != "SUCCESS" {
		return nil, fmt.Errorf("porkbun ListDomains: %s", resp.Message)
	}

	domains := make([]Domain, len(resp.Domains))
	for i, d := range resp.Domains {
		domains[i] = Domain{DomainName: d.Domain}
	}
	return domains, nil
}

// ListRecords returns all DNS records for a domain from Porkbun.
func (p *PorkbunProvider) ListRecords(domainName string) ([]Record, error) {
	url := fmt.Sprintf("%s/dns/retrieve/%s", porkbunBaseURL, domainName)
	var resp porkbunRecordsResponse
	if err := p.doJSON(url, p.auth(), &resp); err != nil {
		return nil, err
	}
	if resp.Status != "SUCCESS" {
		return nil, fmt.Errorf("porkbun ListRecords(%s): %s", domainName, resp.Message)
	}

	records := make([]Record, 0, len(resp.Records))
	for _, r := range resp.Records {
		ttl, _ := strconv.ParseInt(r.TTL, 10, 64)
		var priority *int64
		if r.Prio != "" && r.Prio != "0" {
			if p, err := strconv.ParseInt(r.Prio, 10, 64); err == nil {
				priority = &p
			}
		}
		host := r.Name
		// Porkbun returns the full FQDN (e.g. "sub.example.com").
		// Strip the domain suffix to get just the subdomain.
		if host == domainName {
			host = "@"
		} else if suffix := "." + domainName; len(host) > len(suffix) && host[len(host)-len(suffix):] == suffix {
			host = host[:len(host)-len(suffix)]
		}

		records = append(records, Record{
			RecordID: fmt.Sprintf("%s|%s", domainName, r.ID),
			Host:     host,
			Type:     r.Type,
			Value:    r.Content,
			TTL:      ttl,
			Priority: priority,
		})
	}
	return records, nil
}

// AddRecord creates a new DNS record in Porkbun.
func (p *PorkbunProvider) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	url := fmt.Sprintf("%s/dns/create/%s", porkbunBaseURL, domainName)

	hostName := rr
	if hostName == "@" {
		hostName = ""
	}

	body := porkbunCreateBody{
		porkbunAuth: p.auth(),
		Name:        hostName,
		Type:        recordType,
		Content:     value,
		TTL:         strconv.FormatInt(ttl, 10),
	}
	if priority != nil {
		body.Prio = strconv.FormatInt(*priority, 10)
	}

	var resp porkbunResponse
	if err := p.doJSON(url, body, &resp); err != nil {
		return "", err
	}
	if resp.Status != "SUCCESS" {
		return "", fmt.Errorf("porkbun AddRecord: %s", resp.Message)
	}

	// Porkbun doesn't return the created record ID. Retrieve it.
	return p.findRecordID(domainName, rr, recordType, value)
}

// UpdateRecord updates an existing DNS record in Porkbun.
// recordID format: "domain|recordID"
func (p *PorkbunProvider) UpdateRecord(domainName, recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	domain, pbID, err := parsePorkbunRecordID(recordID)
	if err != nil {
		return err
	}

	hostName := rr
	if hostName == "@" {
		hostName = ""
	}

	url := fmt.Sprintf("%s/dns/edit/%s/%s", porkbunBaseURL, domain, pbID)
	body := porkbunEditBody{
		porkbunAuth: p.auth(),
		Name:        hostName,
		Type:        recordType,
		Content:     value,
		TTL:         strconv.FormatInt(ttl, 10),
	}
	if priority != nil {
		body.Prio = strconv.FormatInt(*priority, 10)
	}

	var resp porkbunResponse
	if err := p.doJSON(url, body, &resp); err != nil {
		return err
	}
	if resp.Status != "SUCCESS" {
		return fmt.Errorf("porkbun UpdateRecord: %s", resp.Message)
	}
	return nil
}

// DeleteRecord deletes a DNS record from Porkbun.
// recordID format: "domain|recordID"
func (p *PorkbunProvider) DeleteRecord(recordID string) error {
	domain, pbID, err := parsePorkbunRecordID(recordID)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/dns/delete/%s/%s", porkbunBaseURL, domain, pbID)
	var resp porkbunResponse
	if err := p.doJSON(url, p.auth(), &resp); err != nil {
		return err
	}
	if resp.Status != "SUCCESS" {
		return fmt.Errorf("porkbun DeleteRecord: %s", resp.Message)
	}
	return nil
}

// findRecordID retrieves the Porkbun record ID after creation.
func (p *PorkbunProvider) findRecordID(domainName, rr, recordType, value string) (string, error) {
	records, err := p.ListRecords(domainName)
	if err != nil {
		return "", err
	}
	for _, r := range records {
		if r.Host == rr && r.Type == recordType && r.Value == value {
			// RecordID is already in "domain|pbID" format from ListRecords.
			return r.RecordID, nil
		}
	}
	return "", fmt.Errorf("porkbun: could not find created record %s/%s", rr, recordType)
}

func parsePorkbunRecordID(recordID string) (domain, pbID string, err error) {
	parts := splitN(recordID, "|", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid porkbun recordID: %s", recordID)
	}
	return parts[0], parts[1], nil
}

// splitN is a helper that returns at most n parts.
func splitN(s, sep string, n int) []string {
	parts := make([]string, 0, n)
	for i := 0; i < n-1; i++ {
		idx := -1
		for j := 0; j < len(s); j++ {
			if s[j] == sep[0] {
				idx = j
				break
			}
		}
		if idx < 0 {
			parts = append(parts, s)
			return parts
		}
		parts = append(parts, s[:idx])
		s = s[idx+1:]
	}
	parts = append(parts, s)
	return parts
}
