package aliyun

import (
	"fmt"

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
		return "", fmt.Errorf("aliyun AddRecord failed: %w", err)
	}

	return dara.StringValue(resp.Body.RecordId), nil
}

func (c *Client) UpdateRecord(recordID, rr, recordType, value string, ttl int64, priority *int64) error {
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
