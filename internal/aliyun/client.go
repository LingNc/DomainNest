package aliyun

import (
	"fmt"
	"strings"

	alidns "github.com/alibabacloud-go/alidns-20150109/v5/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	"github.com/alibabacloud-go/tea/dara"
)

type Client struct {
	client  *alidns.Client
	runtime *dara.RuntimeOptions
}

func (c *Client) AddRecord(domainName, rr, recordType, value string, ttl int64, priority *int64) (string, error) {
	req := &alidns.AddDomainRecordRequest{
		DomainName: dara.String(domainName),
		RR:         dara.String(rr),
		Type:       dara.String(recordType),
		Value:      dara.String(value),
		TTL:        dara.Int64(ttl),
	}
	if priority != nil {
		req.Priority = dara.Int64(*priority)
	}

	resp, err := c.client.AddDomainRecordWithOptions(req, c.runtime)
	if err != nil {
		// DomainRecordDuplicate means the record already exists on the provider.
		// This is not an error — find the existing record and return its ID.
		if strings.Contains(err.Error(), "DomainRecordDuplicate") {
			return c.findExistingRecordID(domainName, rr, recordType, value)
		}
		return "", fmt.Errorf("aliyun AddRecord failed: %w", err)
	}

	return dara.StringValue(resp.Body.RecordId), nil
}

// findExistingRecordID looks up an existing DNS record by its key fields and returns its RecordID.
func (c *Client) findExistingRecordID(domainName, rr, recordType, value string) (string, error) {
	records, err := c.ListAllRecords(domainName)
	if err != nil {
		return "", fmt.Errorf("aliyun findExistingRecord failed: %w", err)
	}
	for _, r := range records {
		if r.RR == rr && r.Type == recordType && r.Value == value {
			return r.RecordID, nil
		}
	}
	return "", fmt.Errorf("aliyun: DomainRecordDuplicate but could not find existing record for %s/%s/%s", rr, recordType, value)
}

type DuplicateRecordError struct {
	RecordID string
}

func (e *DuplicateRecordError) Error() string {
	return "record already exists: " + e.RecordID
}

func (c *Client) UpdateRecord(domainName, recordID, rr, recordType, value string, ttl int64, priority *int64) error {
	req := &alidns.UpdateDomainRecordRequest{
		RecordId: dara.String(recordID),
		RR:       dara.String(rr),
		Type:     dara.String(recordType),
		Value:    dara.String(value),
		TTL:      dara.Int64(ttl),
	}
	if priority != nil {
		req.Priority = dara.Int64(*priority)
	}

	_, err := c.client.UpdateDomainRecordWithOptions(req, c.runtime)
	if err != nil {
		// DomainRecordDuplicate means a record with the same key fields already exists.
		// Find the existing record and return DuplicateRecordError so the caller can handle it.
		if strings.Contains(err.Error(), "DomainRecordDuplicate") {
			id, findErr := c.findExistingRecordID(domainName, rr, recordType, value)
			if findErr == nil {
				return &DuplicateRecordError{RecordID: id}
			}
		}
		return fmt.Errorf("aliyun UpdateRecord failed: %w", err)
	}

	return nil
}

func NewClientFromKeys(accessKeyID, accessKeySecret, endpoint string) (*Client, error) {
	if endpoint == "" {
		endpoint = "alidns.aliyuncs.com"
	}
	apiConfig := &openapi.Config{
		AccessKeyId:     dara.String(accessKeyID),
		AccessKeySecret: dara.String(accessKeySecret),
		Endpoint:        dara.String(endpoint),
	}
	client, err := alidns.NewClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create aliyun DNS client: %w", err)
	}
	return &Client{
		client:  client,
		runtime: &dara.RuntimeOptions{},
	}, nil
}

func (c *Client) DeleteRecord(recordID string) error {
	req := &alidns.DeleteDomainRecordRequest{
		RecordId: dara.String(recordID),
	}

	_, err := c.client.DeleteDomainRecordWithOptions(req, c.runtime)
	if err != nil {
		return fmt.Errorf("aliyun DeleteRecord failed: %w", err)
	}

	return nil
}

type AliyunDomain struct {
	DomainName  string `json:"domain_name"`
	RecordCount int64  `json:"record_count"`
}

func (c *Client) DescribeDomains() ([]AliyunDomain, error) {
	req := &alidns.DescribeDomainsRequest{}
	resp, err := c.client.DescribeDomainsWithOptions(req, c.runtime)
	if err != nil {
		return nil, fmt.Errorf("aliyun DescribeDomains failed: %w", err)
	}
	var domains []AliyunDomain
	for _, d := range resp.Body.Domains.Domain {
		domains = append(domains, AliyunDomain{
			DomainName:  dara.StringValue(d.DomainName),
			RecordCount: dara.Int64Value(d.RecordCount),
		})
	}
	return domains, nil
}

func (c *Client) DescribeDomainRecords(domainName string) error {
	req := &alidns.DescribeDomainRecordsRequest{
		DomainName: dara.String(domainName),
		PageSize:   dara.Int64(1),
	}
	_, err := c.client.DescribeDomainRecordsWithOptions(req, c.runtime)
	if err != nil {
		return fmt.Errorf("aliyun DescribeDomainRecords failed: %w", err)
	}
	return nil
}

// AliyunRecord represents a DNS record returned by the Aliyun API.
type AliyunRecord struct {
	RecordID string
	RR       string
	Type     string
	Value    string
	TTL      int64
	Priority *int64
	Line     string
	Status   string // "Enable" or "Disable"
}

// ListAllRecords fetches all DNS records for a domain with pagination.
func (c *Client) ListAllRecords(domainName string) ([]AliyunRecord, error) {
	var allRecords []AliyunRecord
	pageNumber := int64(1)
	pageSize := int64(500)

	for {
		req := &alidns.DescribeDomainRecordsRequest{
			DomainName: dara.String(domainName),
			PageNumber: dara.Int64(pageNumber),
			PageSize:   dara.Int64(pageSize),
		}
		resp, err := c.client.DescribeDomainRecordsWithOptions(req, c.runtime)
		if err != nil {
			return nil, fmt.Errorf("aliyun ListAllRecords failed: %w", err)
		}

		if resp.Body == nil || resp.Body.DomainRecords == nil {
			break
		}

		for _, r := range resp.Body.DomainRecords.Record {
			if r == nil {
				continue
			}
			rec := AliyunRecord{
				RecordID: dara.StringValue(r.RecordId),
				RR:       dara.StringValue(r.RR),
				Type:     dara.StringValue(r.Type),
				Value:    dara.StringValue(r.Value),
				TTL:      dara.Int64Value(r.TTL),
				Line:     dara.StringValue(r.Line),
				Status:   dara.StringValue(r.Status),
			}
			if r.Priority != nil {
				rec.Priority = r.Priority
			}
			allRecords = append(allRecords, rec)
		}

		totalCount := dara.Int64Value(resp.Body.TotalCount)
		if int64(len(allRecords)) >= totalCount || len(resp.Body.DomainRecords.Record) == 0 {
			break
		}
		pageNumber++
	}

	return allRecords, nil
}
