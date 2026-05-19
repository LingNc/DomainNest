package aliyun

import (
	"fmt"

	"domainnest/internal/config"

	alidns "github.com/alibabacloud-go/alidns-20150109/v5/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	"github.com/alibabacloud-go/tea/dara"
)

type Client struct {
	client  *alidns.Client
	runtime *dara.RuntimeOptions
}

func NewClient(cfg *config.AliyunConfig) (*Client, error) {
	apiConfig := &openapi.Config{
		AccessKeyId:     dara.String(cfg.AccessKeyID),
		AccessKeySecret: dara.String(cfg.AccessKeySecret),
		Endpoint:        dara.String(cfg.Endpoint),
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
